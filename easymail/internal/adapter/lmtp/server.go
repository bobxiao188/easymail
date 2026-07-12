package lmtp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	filtersvc "easymail/internal/app/filter"
	managementSvc "easymail/internal/app/management"
	"easymail/internal/domain/management"
	"easymail/internal/domain/messaging"
	"easymail/internal/domain/messaging/repository"
	"easymail/internal/domain/messaging/storagepath"
	html2text "easymail/internal/pkg/html2text"
	lmtpproto "easymail/internal/protocol/lmtp"
	"easymail/pkg/logger/easylog"

	enmime "github.com/jhillyerd/enmime/v2"
)

type Server struct {
	Accounts    management.MailUserRepository
	Hostname    string
	Log         *easylog.Logger
	InboundLMTP *filtersvc.LMTPRouteOptions

	Provision managementSvc.UserProvisionService
	EmailRepo repository.EmailRepository
	Root      string
}

const (
	lmtpCmdReadDeadline  = 100 * time.Second
	lmtpDataReadDeadline = 10 * time.Minute
	maxMessageSize       = 50 << 20
	maxDataLine          = 1024 * 1024
)

func (s *Server) infof(format string, args ...interface{}) {
	if s != nil && s.Log != nil {
		s.Log.Infof(format, args...)
	} else {
		log.Printf("lmtp: "+format, args...)
	}
}

func (s *Server) warnf(format string, args ...interface{}) {
	if s != nil && s.Log != nil {
		s.Log.Warnf(format, args...)
	} else {
		log.Printf("lmtp: "+format, args...)
	}
}

func (s *Server) hostname() string {
	if strings.TrimSpace(s.Hostname) != "" {
		return strings.TrimSpace(s.Hostname)
	}
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "easymail"
	}
	return h
}

func (s *Server) validateRecipient(ctx context.Context, addr string) bool {
	addr = strings.ToLower(strings.TrimSpace(addr))
	if addr == "" || s.Accounts == nil {
		return false
	}
	_, err := s.Accounts.FindByFullEmail(ctx, addr)
	if err != nil {
		s.infof("rcpt check addr=%q result=not_found", addr)
		return false
	}
	s.infof("rcpt check addr=%q result=ok", addr)
	return true
}

func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			s.warnf("accept error: %v", err)
			continue
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			s.warnf("panic in handleConn: %v", r)
		}
		conn.Close()
	}()
	ra := conn.RemoteAddr().String()
	s.infof("connection opened remote=%s", ra)
	c := &session{srv: s, conn: conn, r: bufio.NewReader(conn), w: bufio.NewWriter(conn)}
	c.reply(220, fmt.Sprintf("%s LMTP ready", s.hostname()))
	_ = c.flush()
	var mailFrom string
	var mailFromSet bool
	var rcpts []string
	for {
		select {
		case <-ctx.Done():
			s.infof("connection cancelled remote=%s", ra)
			return
		default:
		}
		_ = conn.SetDeadline(time.Now().Add(lmtpCmdReadDeadline))
		line, err := lmtpproto.ReadLineLimited(c.r, 64*1024)
		if err != nil {
			if err == io.EOF || isReadIdleOrClosed(err) {
				s.infof("connection closed remote=%s err=%v", ra, err)
				return
			}
			s.warnf("read from %s: %v", ra, err)
			return
		}
		lineTrim := strings.TrimSpace(line)
		upper := strings.ToUpper(lineTrim)
		switch {
		case strings.HasPrefix(upper, "LHLO") || strings.HasPrefix(upper, "HELO") || strings.HasPrefix(upper, "EHLO"):
			s.infof("command LHLO remote=%s", ra)
			c.reply(250, "OK")
			_ = c.flush()
		case strings.HasPrefix(upper, "MAIL FROM"):
			addr, ok := lmtpproto.ParseMailFrom(lineTrim)
			if !ok {
				c.reply(501, "Bad syntax in MAIL FROM")
			} else {
				mailFrom = addr
				mailFromSet = true
				rcpts = nil
				s.infof("MAIL FROM remote=%s from=%q", ra, mailFrom)
				c.reply(250, "OK")
			}
			_ = c.flush()
		case strings.HasPrefix(upper, "RCPT TO"):
			addr, ok := lmtpproto.ParseRcptTo(lineTrim)
			if !ok {
				c.reply(501, "Bad syntax in RCPT TO")
			} else if !mailFromSet {
				c.reply(503, "Bad sequence of commands")
			} else if s.validateRecipient(ctx, addr) {
				rcpts = append(rcpts, strings.ToLower(strings.TrimSpace(addr)))
				c.reply(250, "OK")
			} else {
				c.reply(550, "Mailbox unavailable")
			}
			_ = c.flush()
		case upper == "DATA":
			if !mailFromSet || len(rcpts) == 0 {
				c.reply(503, "Bad sequence of commands")
				_ = c.flush()
				continue
			}
			s.infof("DATA start remote=%s rcpt_count=%d", ra, len(rcpts))
			if err := c.readDataAndDeliver(ctx, rcpts); err != nil {
				s.warnf("DATA session remote=%s: %v", ra, err)
			}
			mailFrom = ""
			mailFromSet = false
			rcpts = nil
		case upper == "RSET":
			mailFrom = ""
			mailFromSet = false
			rcpts = nil
			c.reply(250, "OK")
			_ = c.flush()
		case upper == "QUIT":
			c.reply(221, "Bye")
			_ = c.flush()
			return
		case upper == "NOOP":
			c.reply(250, "OK")
			_ = c.flush()
		default:
			c.reply(500, "Syntax error")
			_ = c.flush()
		}
	}
}

type session struct {
	srv  *Server
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer
}

func (c *session) reply(code int, text string) { _, _ = fmt.Fprintf(c.w, "%d %s\r\n", code, text) }
func (c *session) flush() error                { return c.w.Flush() }

var errMessageTooLarge = errors.New("message too large")

func (c *session) readDataAndDeliver(ctx context.Context, rcpts []string) error {
	// Per-message delivery timeout
	deliveryCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	c.reply(354, "End data with <CRLF>.<CRLF>")
	if err := c.flush(); err != nil {
		return err
	}
	_ = c.conn.SetReadDeadline(time.Now().Add(lmtpDataReadDeadline))
	body, err := readSMTPBody(c.r)
	if err != nil {
		if errors.Is(err, errMessageTooLarge) {
			c.reply(552, "Message too large")
		} else {
			c.reply(451, "Error reading message")
		}
		_ = c.flush()
		return nil
	}
	c.srv.infof("DATA body received bytes=%d rcpt_count=%d", len(body), len(rcpts))
	for _, rcpt := range rcpts {
		select {
		case <-ctx.Done():
			c.reply(451, "Aborted")
			_ = c.flush()
			return ctx.Err()
		default:
		}
		if err := c.srv.deliverMessage(deliveryCtx, rcpt, body); err != nil {
			c.srv.warnf("deliver to %q: %v", rcpt, err)
			c.reply(451, "Error processing message")
		} else {
			c.srv.infof("delivered rcpt=%q size=%d", rcpt, len(body))
			c.reply(250, "2.0.0 Message accepted")
		}
		_ = c.flush()
	}
	return nil
}

func readSMTPBody(r *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer
	for {
		line, err := lmtpproto.ReadLineLimited(r, maxDataLine)
		if err != nil {
			return nil, err
		}
		if line == "." {
			break
		}
		line = strings.TrimPrefix(line, ".")
		if buf.Len()+len(line)+2 > maxMessageSize {
			return nil, errMessageTooLarge
		}
		buf.WriteString(line)
		buf.WriteString("\r\n")
	}
	return buf.Bytes(), nil
}

func (s *Server) deliverMessage(ctx context.Context, rcpt string, body []byte) error {
	act, ruleID, traceID := filtersvc.ParseFilterHeadersFromBody(body)
	kind := filtersvc.FolderKindForInbound(body, s.InboundLMTP)
	s.infof("deliver filter_action=%q rule_id=%q trace_id=%q resolved_folder_kind=%d body_bytes=%d", act, ruleID, traceID, int(kind), len(body))

	rcpt = strings.ToLower(strings.TrimSpace(rcpt))
	user, err := s.Accounts.FindByFullEmail(ctx, rcpt)
	if err != nil {
		return fmt.Errorf("lmtp: user %q not found: %w", rcpt, err)
	}

	if err := s.Provision.EnsureFolders(ctx, user.ID); err != nil {
		s.warnf("lmtp: provision for %q: %v", rcpt, err)
	}

	sender, subject := extractSMTPHeaders(body)
	snippet := extractSnippet(body)
	hasAttachments := detectAttachments(body)

	now := time.Now()
	email := &messaging.Email{
		MailUserID:     user.ID,
		FolderID:       int64(kind),
		Sender:         sender,
		Recipient:      rcpt,
		Subject:        subject,
		Snippet:        snippet,
		MailTime:       now.Format(time.RFC3339),
		MailSize:       int64(len(body)),
		JobID:          ruleID,
		QueueID:        traceID,
		HasAttachments: hasAttachments,
	}
	if err := s.EmailRepo.Save(ctx, email); err != nil {
		return fmt.Errorf("lmtp: save email for %q: %w", rcpt, err)
	}
	s.infof("lmtp: email saved to DB id=%d folder_id=%d file_path=%q has_attachments=%v", email.ID, email.FolderID, email.FilePath, hasAttachments)

	// Use stored data path, fallback to computed path for existing users
	dp := user.DataPath
	if dp == "" {
		parts := strings.SplitN(user.Email, "@", 2)
		if len(parts) != 2 {
			return fmt.Errorf("lmtp: invalid email format %q", user.Email)
		}
		dp = storagepath.MailUserDataPath(parts[1], user.Email)
	}

	// Clean the data path to remove leading slashes and normalize separators
	cleanDP := cleanDataPath(dp)

	// Resolve root to absolute path for safe path construction
	rootAbs, err := filepath.Abs(s.Root)
	if err != nil {
		return fmt.Errorf("lmtp: cannot resolve root path %q: %w", s.Root, err)
	}

	// Join with root to get absolute path
	absPath := filepath.Join(rootAbs, cleanDP)

	// Security check: ensure the path is within the root directory
	rel, err := filepath.Rel(rootAbs, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("lmtp: unsafe data path %q (root=%q)", absPath, rootAbs)
	}
	fullPath := filepath.Join(absPath, email.FilePath)
	dir := filepath.Dir(fullPath)

	// Log directory creation attempt
	s.infof("lmtp: creating directory %s for user %s", dir, user.Email)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.warnf("lmtp: failed to create directory %s for user %s: %v", dir, user.Email, err)
		return fmt.Errorf("lmtp: mkdir %s: %w", dir, err)
	}

	// Log file write attempt
	s.infof("lmtp: writing email file %s for user %s (size=%d bytes)", fullPath, user.Email, len(body))
	if err := os.WriteFile(fullPath, body, 0644); err != nil {
		s.warnf("lmtp: failed to write email file %s for user %s: %v", fullPath, user.Email, err)
		return fmt.Errorf("lmtp: write %s: %w", fullPath, err)
	}

	s.infof("delivered rcpt=%q folder_kind=%d path=%s size=%d msg_id=%d", rcpt, int(kind), fullPath, len(body), email.ID)
	return nil
}

// extractSMTPHeaders extracts sender (From) and Subject from raw email headers.
func extractSMTPHeaders(body []byte) (sender, subject string) {
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			break
		}
		low := strings.ToLower(trimmed)
		if strings.HasPrefix(low, "from:") {
			sender = strings.TrimSpace(trimmed[5:])
		} else if strings.HasPrefix(low, "subject:") {
			subject = strings.TrimSpace(trimmed[8:])
		}
	}
	// Decode MIME-encoded subject so DB stores plain text (searchable by INSTR/LIKE)
	if strings.Contains(subject, "=?") {
		dec := mime.WordDecoder{}
		if d, err := dec.DecodeHeader(subject); err == nil {
			subject = d
		}
	}
	return
}

// detectAttachments checks if the raw email body contains attachments.
func detectAttachments(body []byte) bool {
	if len(body) == 0 {
		return false
	}

	// Parse with enmime to detect attachments
	env, err := enmime.ReadEnvelope(bytes.NewReader(body))
	if err != nil {
		return false
	}

	// Check if envelope has attachments
	return len(env.Attachments) > 0
}

// extractSnippet extracts a plain-text snippet from raw email body using enmime.
// It parses the MIME structure, extracts TextBody or HTMLBody (converted to text),
// then returns the first 200 characters. This approach correctly handles multipart emails.
func extractSnippet(body []byte) string {
	const maxSnippetLength = 200

	if len(body) == 0 {
		return ""
	}

	// Parse with enmime to get structured body
	env, err := enmime.ReadEnvelope(bytes.NewReader(body))
	if err != nil {
		// Fallback: best-effort from raw body
		return extractSnippetFallback(body, maxSnippetLength)
	}

	// Determine the text to use for snippet
	var textContent string
	htmlBody := strings.TrimSpace(env.HTML)
	rowText := strings.TrimSpace(env.Text)

	if htmlBody == "" {
		// Only text body exists
		textContent = rowText
	} else {
		// HTML exists: convert HTML to text using html2text
		h2t := html2text.NewHtml2Text(nil)
		textParts, _ := h2t.Parse(htmlBody)
		var b strings.Builder
		for _, p := range textParts {
			p = strings.TrimSpace(p)
			if p != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(p)
			}
		}
		textContent = strings.TrimSpace(b.String())
	}

	if textContent == "" {
		return ""
	}

	// Extract snippet from clean text content
	snippet := textContent
	if len(snippet) > maxSnippetLength {
		snippet = snippet[:maxSnippetLength] + "..."
	}

	return snippet
}

// extractSnippetFallback is a fallback when enmime parsing fails.
func extractSnippetFallback(body []byte, maxSnippetLength int) string {
	bodyStr := string(body)
	if idx := strings.Index(bodyStr, "\r\n\r\n"); idx != -1 {
		bodyStr = bodyStr[idx+4:]
	} else if idx := strings.Index(bodyStr, "\n\n"); idx != -1 {
		bodyStr = bodyStr[idx+2:]
	}

	cleaned := bodyStr
	if strings.Contains(cleaned, "<html") || strings.Contains(cleaned, "<body") || strings.Contains(cleaned, "<div") {
		cleaned = stripHTMLEntities(cleaned)
	}
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	lines := strings.Split(cleaned, "\n")
	var snippetBuilder strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && snippetBuilder.Len() < maxSnippetLength {
			if snippetBuilder.Len() > 0 {
				snippetBuilder.WriteString(" ")
			}
			snippetBuilder.WriteString(trimmed)
		}
		if snippetBuilder.Len() >= maxSnippetLength {
			break
		}
	}

	snippet := snippetBuilder.String()
	if len(snippet) > maxSnippetLength {
		snippet = snippet[:maxSnippetLength] + "..."
	}
	return snippet
}

// stripHTMLEntities removes HTML tags from a string
func stripHTMLEntities(s string) string {
	var result strings.Builder
	inTag := false
	for _, ch := range s {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
			result.WriteString(" ")
		} else if !inTag {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func isReadIdleOrClosed(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	return errors.Is(err, net.ErrClosed)
}

// cleanDataPath sanitizes a user data path against directory traversal.
func cleanDataPath(dp string) string {
	return filepath.Clean(dp)
}
