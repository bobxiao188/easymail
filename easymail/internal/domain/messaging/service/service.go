package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"easymail/internal/domain/mailbox"
	"easymail/internal/domain/messaging"
	mailexception "easymail/internal/domain/messaging/exception"
	"easymail/internal/domain/messaging/repository"
	"easymail/internal/domain/shared"
	"easymail/pkg/config"
	"easymail/pkg/constants"
)

// pendingJob carries an outbound email that needs to be sent via SMTP asynchronously.
type pendingJob struct {
	userID     shared.GlobalID
	emailID    int64
	fromEmail  string
	rawContent []byte
	recipients []string
}

// mailService implements Backend by bridging int64 IDs to domain repositories.
type mailService struct {
	getDataPath       func(ctx context.Context, userID shared.GlobalID) (root, dataPath string, err error)
	folderRepo        mailbox.FolderRepository
	emailRepo         repository.EmailRepository
	mailUserFinder    MailUserFinder
	dkimSigner        DKIMSigner
	root              string
	smtpConfig        *config.SMTPConfig
	folderIDMu        sync.Mutex
	folderIDMap       map[shared.GlobalID]int64
	folderIDByNumeric map[int64]shared.GlobalID
	pendingCh         chan pendingJob
}

// signEmailWithDKIM signs the email with DKIM using the sender's domain.
// It extracts the domain from the sender email address and calls the DKIMSigner.
// If DKIM signer is not configured or the domain has no DKIM key, the email is left unchanged.
func (s *mailService) signEmailWithDKIM(ctx context.Context, email *[]byte, sender string) {
	if s.dkimSigner == nil {
		return
	}
	// Extract domain from sender email (e.g. "user@example.com" -> "example.com")
	domain := extractDomain(sender)
	if domain == "" {
		return
	}
	if err := s.dkimSigner.Sign(ctx, email, domain); err != nil {
		log.Printf("Warning: DKIM signing failed for sender=%s domain=%s: %v", sender, domain, err)
	}
}

// extractDomain extracts the domain part from an email address.
// Handles formats like "user@example.com" or "Name <user@example.com>"
func extractDomain(email string) string {
	email = strings.TrimSpace(email)
	// Handle "Name <email>" format
	if idx := strings.LastIndex(email, "<"); idx >= 0 {
		if end := strings.LastIndex(email, ">"); end > idx {
			email = email[idx+1 : end]
		} else {
			email = email[idx+1:]
		}
	}
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return strings.ToLower(strings.TrimSpace(parts[1]))
	}
	return ""
}

// encodeRFC2047Header encodes a header field according to RFC 2047
// Handles formats like "Name <email>" or just "email"
func encodeRFC2047Header(header string) string {
	if header == "" {
		return ""
	}

	// Check if it's in "Name <email>" format
	if strings.Contains(header, "<") && strings.Contains(header, ">") {
		// Parse "Name <email>"
		parts := strings.Split(header, "<")
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			emailPart := "<" + parts[1]

			// Encode the name part using mime.QEncoding
			encodedName := mime.QEncoding.Encode("UTF-8", name)
			return encodedName + emailPart
		}
	}

	// Check if the header contains non-ASCII characters
	needsEncoding := false
	for _, r := range header {
		if r > 127 {
			needsEncoding = true
			break
		}
	}
	if !needsEncoding {
		return header
	}

	// Encode using mime.QEncoding
	return mime.QEncoding.Encode("UTF-8", header)
}

// NewService creates a concrete Backend using the given repositories.
func NewService(
	getDataPath func(ctx context.Context, userID shared.GlobalID) (root, dataPath string, err error),
	folderRepo mailbox.FolderRepository,
	emailRepo repository.EmailRepository,
	mailUserFinder MailUserFinder,
	dkimSigner DKIMSigner,
	root string,
	smtpConfig *config.SMTPConfig,
) Backend {
	s := &mailService{
		getDataPath:       getDataPath,
		folderRepo:        folderRepo,
		emailRepo:         emailRepo,
		mailUserFinder:    mailUserFinder,
		dkimSigner:        dkimSigner,
		root:              root,
		smtpConfig:        smtpConfig,
		folderIDMap:       make(map[shared.GlobalID]int64),
		folderIDByNumeric: make(map[int64]shared.GlobalID),
		pendingCh:         make(chan pendingJob, 100),
	}

	// start async sender goroutine
	go s.processPendingJobs(context.Background())

	return s
}

// --- helpers ---

func (s *mailService) toGlobalID(id int64) shared.GlobalID {
	s.folderIDMu.Lock()
	defer s.folderIDMu.Unlock()
	if gid, ok := s.folderIDByNumeric[id]; ok {
		return gid
	}
	return shared.GlobalID(fmt.Sprintf("%d", id))
}

func (s *mailService) dataPathForUser(ctx context.Context, userID shared.GlobalID) (root, dataPath string, err error) {
	return s.getDataPath(ctx, userID)
}

func (s *mailService) folderToDTO(ctx context.Context, userID shared.GlobalID, f *mailbox.Folder) *FolderDTO {
	var id int64
	if !mailbox.IsSystemFolderKind(f.FolderKind) {
		id = s.folderGlobalIDToInt64(ctx, userID, f.ID)
	} else {
		// System folders: map mailbox.FolderKind to constants.FolderID values
		// because LMTP stores emails with constants.FolderID.
		switch f.FolderKind {
		case mailbox.FolderInbox:
			id = int64(constants.Inbox)
		case mailbox.FolderSent:
			id = int64(constants.Sent)
		case mailbox.FolderDraft:
			id = int64(constants.Draft)
		case mailbox.FolderTrash:
			id = int64(constants.Trash)
		case mailbox.FolderSpam:
			id = int64(constants.Spam)
		case mailbox.FolderQuarantine:
			id = int64(constants.Quarantine)
		default:
			id = int64(f.FolderKind)
		}
		// Populate reverse cache for system folders
		s.folderIDMu.Lock()
		s.folderIDMap[f.ID] = id
		s.folderIDByNumeric[id] = f.ID
		s.folderIDMu.Unlock()
	}
	return &FolderDTO{
		ID:          id,
		Name:        f.FolderName,
		IMAPName:    mailbox.IMAPNameForKind(f.FolderKind),
		Kind:        constants.FolderID(id),
		UIDValidity: f.UIDValidity,
	}
}

func (s *mailService) folderGlobalIDToInt64(ctx context.Context, userID shared.GlobalID, id shared.GlobalID) int64 {
	s.folderIDMu.Lock()
	defer s.folderIDMu.Unlock()
	if cached, ok := s.folderIDMap[id]; ok {
		return cached
	}
	// Check if the GlobalID is already a valid int64 (system folders)
	n, err := strconv.ParseInt(string(id), 10, 64)
	if err == nil {
		s.folderIDMap[id] = n
		s.folderIDByNumeric[n] = id
		return n
	}
	// Try loading persisted mapping from database
	nid, dbErr := s.emailRepo.GetFolderNumericID(ctx, userID, string(id))
	if dbErr == nil && nid >= 100 {
		s.folderIDMap[id] = nid
		s.folderIDByNumeric[nid] = id
		return nid
	}
	// Assign next sequential ID starting from FolderUserCustomMin (100)
	nid, err = s.emailRepo.GetNextCustomFolderID(ctx, userID)
	if err != nil {
		nid = int64(constants.FolderUserCustomMin)
	}
	_ = s.emailRepo.SetFolderNumericID(ctx, userID, string(id), nid)
	s.folderIDMap[id] = nid
	s.folderIDByNumeric[nid] = id
	return nid
}

// --- Folder operations ---

func (s *mailService) ListFolders(ctx context.Context, userID shared.GlobalID) ([]FolderDTO, error) {
	folders, err := s.folderRepo.ListByMailUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Batch query counts for all folders in one SQL
	counts, _ := s.emailRepo.CountByFolder(ctx, userID)
	result := make([]FolderDTO, 0, len(folders))
	for _, f := range folders {
		dto := s.folderToDTO(ctx, userID, f)
		if c, ok := counts[dto.ID]; ok {
			dto.UnreadCount = c.Unread
			dto.TotalCount = c.Total
		}
		result = append(result, *dto)
	}
	return result, nil
}

func (s *mailService) CreateFolder(ctx context.Context, userID shared.GlobalID, name string) (*FolderDTO, error) {
	// Check for duplicate folder name (case-insensitive)
	existing, err := s.folderRepo.ListByMailUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, ef := range existing {
		if strings.EqualFold(ef.FolderName, name) {
			return nil, mailexception.ErrAlreadyExists
		}
	}

	f, err := mailbox.NewFolder(userID, name, mailbox.FolderKind(constants.FolderUserCustomMin))
	if err != nil {
		return nil, mailexception.ErrInvalidArgument
	}
	if err := s.folderRepo.Save(ctx, f); err != nil {
		return nil, err
	}
	return s.folderToDTO(ctx, userID, f), nil
}

func (s *mailService) RenameFolder(ctx context.Context, userID shared.GlobalID, folderID int64, name string) error {
	f, err := s.folderRepo.FindByID(ctx, userID, s.toGlobalID(folderID))
	if err != nil {
		return mailexception.ErrNotFound
	}
	if !f.BelongsToMailUser(userID) {
		return mailexception.ErrNotFound
	}
	if err := f.Rename(name); err != nil {
		return convertFolderError(err)
	}
	return s.folderRepo.Update(ctx, f)
}

func (s *mailService) DeleteFolder(ctx context.Context, userID shared.GlobalID, folderID int64) error {
	f, err := s.folderRepo.FindByID(ctx, userID, s.toGlobalID(folderID))
	if err != nil {
		return mailexception.ErrNotFound
	}
	if !f.BelongsToMailUser(userID) {
		return mailexception.ErrNotFound
	}
	if err := f.CanDelete(); err != nil {
		return convertFolderError(err)
	}
	// Check if folder has active emails
	count, err := s.emailRepo.CountActiveInFolder(ctx, userID, folderID)
	if err != nil {
		return err
	}
	if count > 0 {
		return mailexception.ErrFolderNotEmpty
	}
	return s.folderRepo.SoftDelete(ctx, userID, f.ID)
}

// --- Message operations ---

func (s *mailService) ListMessages(ctx context.Context, userID shared.GlobalID, folderID int64, query ListQuery) (total int64, unread int64, items []MessageDTO, err error) {
	total, unreadCount, emails, err := s.emailRepo.QueryByFolder(ctx, userID, folderID, query.OrderField, query.OrderDir, query.Page, query.PageSize, query.Search, query.LabelID)
	if err != nil {
		return 0, 0, nil, err
	}
	items = make([]MessageDTO, len(emails))
	for i, e := range emails {
		items[i] = MessageDTO{
			ID:             e.ID,
			Sender:         decodeMIMEWords(e.Sender),
			Recipient:      decodeMIMEWords(e.Recipient),
			CarbonCopy:     decodeMIMEWords(e.CarbonCopy),
			BlindCopy:      decodeMIMEWords(e.BlindCopy),
			Subject:        decodeMIMEWords(e.Subject),
			Snippet:        e.Snippet,
			MailTime:       e.MailTime,
			MailSize:       e.MailSize,
			FolderID:       e.FolderID,
			ReadStatus:     e.ReadStatus,
			Flagged:        e.Flagged,
			HasAttachments: e.HasAttachments,
		}
	}
	return total, unreadCount, items, nil
}

func (s *mailService) GetMessage(ctx context.Context, userID shared.GlobalID, mailID int64) (*messaging.Email, error) {
	e, err := s.emailRepo.GetMail(ctx, userID, mailID)
	if err != nil {
		return nil, err
	}
	e.Subject = decodeMIMEWords(e.Subject)
	e.Sender = decodeMIMEWords(e.Sender)
	e.Recipient = decodeMIMEWords(e.Recipient)
	e.CarbonCopy = decodeMIMEWords(e.CarbonCopy)
	return e, nil
}

func (s *mailService) GetMessageBodyHTML(ctx context.Context, userID shared.GlobalID, mailID int64) (string, error) {
	e, err := s.emailRepo.GetMail(ctx, userID, mailID)
	if err != nil {
		return "", err
	}
	// If body is stored in DB, return it directly
	if e.Body != "" {
		return e.Body, nil
	}
	// Otherwise extract body from raw .eml file
	root, dataPath, err := s.dataPathForUser(ctx, userID)
	if err != nil {
		return "", err
	}
	// Build full path: root + dataPath + filename
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(rootAbs, dataPath, e.FilePath)
	body, err := extractBodyFromFile(fullPath)
	if err != nil {
		return "", err
	}
	return body, nil
}

func (s *mailService) OpenMessageRaw(ctx context.Context, userID shared.GlobalID, mailID int64) (io.ReadCloser, int64, error) {
	e, err := s.emailRepo.GetMail(ctx, userID, mailID)
	if err != nil {
		return nil, 0, err
	}
	root, dataPath, err := s.dataPathForUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return openMessageFile(filepath.Join(root, dataPath), e.FilePath)
}

func (s *mailService) MarkRead(ctx context.Context, userID shared.GlobalID, mailID int64, status constants.ReadStatus) error {
	return s.emailRepo.MarkRead(ctx, userID, mailID, status)
}

func (s *mailService) MoveMessage(ctx context.Context, userID shared.GlobalID, mailID int64, folderID int64) error {
	return s.emailRepo.MoveMail(ctx, userID, mailID, folderID)
}

func (s *mailService) SetMessageFlagged(ctx context.Context, userID shared.GlobalID, mailID int64, flagged bool) error {
	return s.emailRepo.SetFlagged(ctx, userID, mailID, flagged)
}

func (s *mailService) SetMessageDeleted(ctx context.Context, userID shared.GlobalID, mailID int64, deleted bool) error {
	return s.emailRepo.SetDeleted(ctx, userID, mailID, deleted)
}

func (s *mailService) MoveToTrash(ctx context.Context, userID shared.GlobalID, mailID int64) error {
	return s.emailRepo.DeleteMail(ctx, userID, mailID)
}

func (s *mailService) PurgeMessage(ctx context.Context, userID shared.GlobalID, mailID int64) error {
	return s.emailRepo.HardDeleteMail(ctx, userID, mailID)
}

// --- Attachment operations ---

func (s *mailService) ListMessageAttachments(ctx context.Context, userID shared.GlobalID, mailID int64) ([]AttachmentDTO, error) {
	e, err := s.emailRepo.GetMail(ctx, userID, mailID)
	if err != nil {
		return nil, err
	}
	root, dataPath, err := s.dataPathForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return listAttachmentsFromFile(filepath.Join(rootAbs, dataPath), e.FilePath, e.MailSize)
}

func (s *mailService) OpenAttachment(ctx context.Context, userID shared.GlobalID, mailID int64, index int) (io.ReadCloser, string, string, int64, error) {
	e, err := s.emailRepo.GetMail(ctx, userID, mailID)
	if err != nil {
		return nil, "", "", 0, err
	}
	root, dataPath, err := s.dataPathForUser(ctx, userID)
	if err != nil {
		return nil, "", "", 0, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, "", "", 0, err
	}
	return openAttachmentFromFile(filepath.Join(rootAbs, dataPath), e.FilePath, index)
}

// --- Batch operations ---

func (s *mailService) GetAttachmentsZip(ctx context.Context, userID shared.GlobalID, mailID int64) ([]byte, string, error) {
	e, err := s.emailRepo.GetMail(ctx, userID, mailID)
	if err != nil {
		return nil, "", err
	}
	root, dataPath, err := s.dataPathForUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}
	return createAttachmentsZip(filepath.Join(rootAbs, dataPath), e.FilePath)
}

func (s *mailService) BatchMoveToTrash(ctx context.Context, userID shared.GlobalID, mailIDs []int64) error {
	for _, mid := range mailIDs {
		if err := s.emailRepo.DeleteMail(ctx, userID, mid); err != nil {
			return err
		}
	}
	return nil
}

func (s *mailService) BatchMove(ctx context.Context, userID shared.GlobalID, mailIDs []int64, folderID int64) error {
	for _, mid := range mailIDs {
		if err := s.emailRepo.MoveMail(ctx, userID, mid, folderID); err != nil {
			return err
		}
	}
	return nil
}

func (s *mailService) BatchMarkRead(ctx context.Context, userID shared.GlobalID, mailIDs []int64, read bool) error {
	status := constants.WebRead
	if !read {
		status = constants.UnRead
	}
	for _, mid := range mailIDs {
		if err := s.emailRepo.MarkRead(ctx, userID, mid, status); err != nil {
			return err
		}
	}
	return nil
}

// --- Label operations ---

func (s *mailService) ListLabels(ctx context.Context, userID shared.GlobalID) ([]shared.LabelDTO, error) {
	return s.emailRepo.ListLabels(ctx, userID)
}

func (s *mailService) CreateLabel(ctx context.Context, userID shared.GlobalID, name, color string) (*shared.LabelDTO, error) {
	return s.emailRepo.CreateLabel(ctx, userID, name, color)
}

func (s *mailService) UpdateLabel(ctx context.Context, userID shared.GlobalID, labelID int64, name, color string) error {
	return s.emailRepo.UpdateLabel(ctx, userID, labelID, name, color)
}

func (s *mailService) DeleteLabel(ctx context.Context, userID shared.GlobalID, labelID int64) error {
	return s.emailRepo.DeleteLabel(ctx, userID, labelID)
}

func (s *mailService) SetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64, labelIDs []int64) error {
	return s.emailRepo.SetEmailLabels(ctx, userID, emailID, labelIDs)
}

func (s *mailService) GetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64) ([]shared.LabelDTO, error) {
	return s.emailRepo.GetEmailLabels(ctx, userID, emailID)
}

func (s *mailService) GetLabelsForEmails(ctx context.Context, userID shared.GlobalID, emailIDs []int64) (map[int64][]shared.LabelDTO, error) {
	return s.emailRepo.GetLabelsForEmails(ctx, userID, emailIDs)
}

func (s *mailService) ListAllMailIDs(ctx context.Context, userID shared.GlobalID, folderID int64) ([]int64, error) {
	return s.emailRepo.ListAllIDs(ctx, userID, folderID)
}

func (s *mailService) BatchSetFlagged(ctx context.Context, userID shared.GlobalID, mailIDs []int64, flagged bool) error {
	for _, mid := range mailIDs {
		if err := s.emailRepo.SetFlagged(ctx, userID, mid, flagged); err != nil {
			return err
		}
	}
	return nil
}

// --- Compose ---

// SaveMessage persists a new email into a folder, returning the assigned ID.
// Used by IMAP APPEND to store raw messages.
func (s *mailService) SaveMessage(ctx context.Context, email *messaging.Email) (int64, error) {
	// Auto-extract snippet if not already set
	if email.Snippet == "" && email.Body != "" {
		email.Snippet = extractSnippetFromText(email.Body)
	}
	// Detect attachments if not already set
	if !email.HasAttachments && email.FilePath != "" {
		root, dataPath, err := s.dataPathForUser(ctx, email.MailUserID)
		if err == nil {
			rootAbs, err := filepath.Abs(root)
			if err == nil {
				atts, err := listAttachmentsFromFile(filepath.Join(rootAbs, dataPath), email.FilePath, email.MailSize)
				if err == nil && len(atts) > 0 {
					email.HasAttachments = true
				}
			}
		}
	}
	if err := s.emailRepo.Save(ctx, email); err != nil {
		return 0, err
	}
	return email.ID, nil
}

// UpdateMessage updates an existing email message in-place.
func (s *mailService) UpdateMessage(ctx context.Context, email *messaging.Email) error {
	return s.emailRepo.Save(ctx, email)
}

func (s *mailService) SendCompose(ctx context.Context, userID shared.GlobalID, email string, req ComposeRequest) (int64, error) {
	now := time.Now()
	body := req.Text
	if body == "" {
		body = req.HTML
	}

	// Append signature if provided
	if req.Signature != "" {
		separator := "\n\n-- \n"
		// Check if signature separator already exists
		if !strings.Contains(body, separator) {
			body = body + separator + req.Signature
		}
	}

	// Use req.From for Sender if provided
	sender := email
	if req.From != nil && req.From.Email != "" {
		sender = req.From.Email
	}

	// Check if request has attachments
	hasAttachments := len(req.Attachments) > 0

	// Update existing draft if ID is provided
	if req.ID > 0 {
		existingEmail := &messaging.Email{
			ID:             req.ID,
			MailUserID:     userID,
			Sender:         sender,
			Subject:        req.Subject,
			Body:           body,
			Snippet:        extractSnippetFromText(body),
			Recipient:      req.To,
			CarbonCopy:     req.Cc,
			BlindCopy:      req.Bcc,
			FolderID:       req.FolderID,
			MailSize:       int64(len(body)),
			MailTime:       now.Format(time.RFC3339),
			HasAttachments: hasAttachments,
		}
		if err := s.emailRepo.Save(ctx, existingEmail); err != nil {
			return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeSaveFailed, err)
		}
		// Write .eml file so extractBodyFromEmail can read it
		rawContent := buildRawEmailContent(existingEmail, req)
		if err := s.writeMessageFile(ctx, userID, existingEmail, rawContent); err != nil {
			log.Printf("Warning: failed to write eml for updated draft %d: %v", existingEmail.ID, err)
			return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeWriteFile, err)
		}
		return req.ID, nil
	}

	// Save to a specific folder (used by SaveDraft)
	if req.FolderID != 0 {
		draftEmail := &messaging.Email{
			MailUserID:     userID,
			FolderID:       req.FolderID,
			Sender:         sender,
			Recipient:      req.To,
			CarbonCopy:     req.Cc,
			BlindCopy:      req.Bcc,
			Subject:        req.Subject,
			Body:           body,
			Snippet:        extractSnippetFromText(body),
			MailTime:       now.Format(time.RFC3339),
			MailSize:       int64(len(body)),
			HasAttachments: hasAttachments,
		}
		if err := s.emailRepo.Save(ctx, draftEmail); err != nil {
			return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeSaveFailed, err)
		}
		// Write .eml file so extractBodyFromEmail can read it
		rawContent := buildRawEmailContent(draftEmail, req)
		if err := s.writeMessageFile(ctx, userID, draftEmail, rawContent); err != nil {
			log.Printf("Warning: failed to write eml for draft %d: %v", draftEmail.ID, err)
			return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeWriteFile, err)
		}

		return draftEmail.ID, nil
	}

	// Evaluate SMTP configuration once
	smtpConfigured := s.smtpConfig != nil && s.smtpConfig.Enable

	// Save to sender's Sent folder if SaveSent is true
	var sentID int64
	if req.SaveSent {
		smtpStatus := ""
		if smtpConfigured {
			smtpStatus = messaging.SMTPStatusPending
		}
		sentEmail := &messaging.Email{
			MailUserID:     userID,
			FolderID:       int64(constants.Sent),
			Sender:         sender,
			Recipient:      req.To,
			CarbonCopy:     req.Cc,
			Subject:        req.Subject,
			Body:           body,
			Snippet:        extractSnippetFromText(body),
			MailTime:       now.Format(time.RFC3339),
			MailSize:       int64(len(body)),
			HasAttachments: hasAttachments,
			SMTPStatus:     smtpStatus,
		}
		if err := s.emailRepo.Save(ctx, sentEmail); err != nil {
			return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeSaveFailed, err)
		}
		sentID = sentEmail.ID
		// Write .eml file so extractBodyFromEmail can read it
		rawContent := buildRawEmailContent(sentEmail, req)
		// Sign with DKIM before storing
		s.signEmailWithDKIM(ctx, &rawContent, sender)
		if err := s.writeMessageFile(ctx, userID, sentEmail, rawContent); err != nil {
			log.Printf("Warning: failed to write eml for sent email %d: %v", sentEmail.ID, err)
			return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeWriteFile, err)
		}
	}

	// Send via SMTP asynchronously if configured and enabled
	if smtpConfigured {
		// Build sending message
		sendBody := req.Text
		if sendBody == "" {
			sendBody = req.HTML
		}
		sendContent := s.buildSendMessage(sender, sendBody, req)
		s.signEmailWithDKIM(ctx, &sendContent, sender)

		// Build flat recipient list for SMTP envelope
		var recipients []string
		if req.To != "" {
			recipients = append(recipients, req.To)
		}
		if req.Cc != "" {
			recipients = append(recipients, strings.Split(req.Cc, ",")...)
		}
		if req.Bcc != "" {
			recipients = append(recipients, strings.Split(req.Bcc, ",")...)
		}

		s.pendingCh <- pendingJob{
			userID:     userID,
			emailID:    sentID,
			fromEmail:  sender,
			rawContent: sendContent,
			recipients: recipients,
		}
		return sentID, nil
	}

	// Deliver to recipient's Inbox if they are a local user
	if s.mailUserFinder != nil && req.To != "" {
		recipientID, lookupErr := s.mailUserFinder.FindByFullEmail(ctx, req.To)
		if lookupErr == nil && recipientID != "" {
			// Avoid self-delivery (already saved to Sent)
			if recipientID != userID {
				inboxEmail := &messaging.Email{
					MailUserID:     recipientID,
					FolderID:       int64(constants.Inbox),
					Sender:         sender,
					Recipient:      req.To,
					CarbonCopy:     req.Cc,
					Subject:        req.Subject,
					Body:           body,
					Snippet:        extractSnippetFromText(body),
					MailTime:       now.Format(time.RFC3339),
					MailSize:       int64(len(body)),
					HasAttachments: hasAttachments,
				}
				if err := s.emailRepo.Save(ctx, inboxEmail); err != nil {
					return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeSaveFailed, err)
				}
				// Write .eml file so extractBodyFromEmail can read it
				rawContent := buildRawEmailContent(inboxEmail, req)
				// Sign with DKIM before storing in recipient inbox
				s.signEmailWithDKIM(ctx, &rawContent, sender)
				if err := s.writeMessageFile(ctx, recipientID, inboxEmail, rawContent); err != nil {
					log.Printf("Warning: failed to write eml for inbox email %d: %v", inboxEmail.ID, err)
					return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeWriteFile, err)
				}
			} else {
				return 0, nil
			}
		} else {
			if lookupErr != nil {
				return 0, fmt.Errorf("%w: %w", mailexception.ErrComposeRecipientNotFound, lookupErr)
			}
		}
	}

	return 0, nil
}

// decodeMIMEWords decodes RFC 2047 encoded header values (e.g. =?utf-8?b?...?=)
func decodeMIMEWords(s string) string {
	dec := mime.WordDecoder{}
	if strings.Contains(s, "=?utf-8") || strings.Contains(s, "=?UTF-8") || strings.Contains(s, "=?gbk") || strings.Contains(s, "=?GBK") {
		d, err := dec.DecodeHeader(s)
		if err == nil {
			return d
		}
	}
	return s
}

// extractSnippetFromText extracts a plain-text snippet from email body text.
// It returns the first 200 characters of the text, stripping HTML tags if present.
func extractSnippetFromText(body string) string {
	const maxSnippetLength = 200

	if body == "" {
		return ""
	}

	cleaned := body

	// Check if it's HTML content
	trimmedBody := strings.TrimSpace(cleaned)
	if strings.HasPrefix(trimmedBody, "<") || strings.Contains(cleaned, "<html") || strings.Contains(cleaned, "<body") || strings.Contains(cleaned, "<div") {
		// Remove HTML tags
		cleaned = stripHTMLTags(cleaned)
	}

	// Remove excessive whitespace
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	// Extract first non-empty lines
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

// stripHTMLTags removes HTML tags from a string
func stripHTMLTags(s string) string {
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

func convertFolderError(err error) error {
	switch {
	case strings.Contains(err.Error(), "system folder"):
		return mailexception.ErrFolderSystem
	case strings.Contains(err.Error(), "not found"):
		return mailexception.ErrNotFound
	default:
		return mailexception.ErrInvalidArgument
	}
}

func openMessageFile(root, filePath string) (io.ReadCloser, int64, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, 0, err
	}
	full := filepath.Join(rootAbs, filePath)
	f, err := os.Open(full)
	if err != nil {
		return nil, 0, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, fi.Size(), nil
}

func (s *mailService) GetMailUsage(ctx context.Context, userID shared.GlobalID) (int64, error) {
	return s.emailRepo.GetMailUsage(ctx, userID)
}

var _ Backend = (*mailService)(nil)

// buildSendMessage constructs SMTP message bytes (headers + body) for a ComposeRequest.
func (s *mailService) buildSendMessage(fromEmail string, body string, req ComposeRequest) []byte {
	var message []byte

	// From header
	if req.From != nil {
		fromHeader := ""
		if req.From.Name != "" {
			fromHeader = encodeRFC2047Header(req.From.Name + " <" + req.From.Email + ">")
		} else {
			fromHeader = req.From.Email
		}
		message = append(message, []byte(fmt.Sprintf("From: %s\r\n", fromHeader))...)
	} else {
		message = append(message, []byte(fmt.Sprintf("From: %s\r\n", encodeRFC2047Header(fromEmail)))...)
	}

	// To header
	if len(req.ToRecipients) > 0 {
		var toHeaders []string
		for _, r := range req.ToRecipients {
			if r.Name != "" {
				toHeaders = append(toHeaders, encodeRFC2047Header(r.Name+" <"+r.Email+">"))
			} else {
				toHeaders = append(toHeaders, r.Email)
			}
		}
		message = append(message, []byte(fmt.Sprintf("To: %s\r\n", strings.Join(toHeaders, ", ")))...)
	} else {
		message = append(message, []byte(fmt.Sprintf("To: %s\r\n", encodeRFC2047Header(req.To)))...)
	}

	// Cc header (optional)
	if len(req.CcRecipients) > 0 {
		var ccHeaders []string
		for _, r := range req.CcRecipients {
			if r.Name != "" {
				ccHeaders = append(ccHeaders, encodeRFC2047Header(r.Name+" <"+r.Email+">"))
			} else {
				ccHeaders = append(ccHeaders, r.Email)
			}
		}
		message = append(message, []byte(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccHeaders, ", ")))...)
	} else if req.Cc != "" {
		message = append(message, []byte(fmt.Sprintf("Cc: %s\r\n", encodeRFC2047Header(req.Cc)))...)
	}

	message = append(message, []byte(fmt.Sprintf("Subject: %s\r\n", encodeRFC2047Header(req.Subject)))...)
	message = append(message, []byte(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))...)
	message = append(message, []byte(fmt.Sprintf("Message-ID: <%s.%d@%s>\r\n", time.Now().Format("20060102150405"), time.Now().UnixNano(), extractDomain(fromEmail)))...)
	message = append(message, []byte("MIME-Version: 1.0\r\n")...)
	message = append(message, []byte("Content-Type: text/html; charset=\"UTF-8\"\r\n")...)
	message = append(message, []byte("\r\n")...)
	message = append(message, []byte(body)...)

	return message
}

// sendSMTPToServer connects to the configured SMTP relay and sends the raw email content.
func (s *mailService) sendSMTPToServer(fromEmail string, rawContent []byte, recipients []string) error {
	if s.smtpConfig == nil || !s.smtpConfig.Enable {
		return mailexception.ErrComposeSMTPConnect
	}

	addr := fmt.Sprintf("%s:%d", s.smtpConfig.Server, s.smtpConfig.Port)

	// Connect to SMTP server
	var conn net.Conn
	var client *smtp.Client
	var err error

	if s.smtpConfig.UseTLS {
		// SSL/TLS connection (port 465)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: s.smtpConfig.InsecureSkipVerify,
			ServerName:         s.smtpConfig.Server,
		}
		conn, err = tls.Dial("tcp", addr, tlsConfig)
		if err == nil {
			client, err = smtp.NewClient(conn, s.smtpConfig.Server)
		}
	} else if s.smtpConfig.UseSTARTTLS {
		// STARTTLS connection (port 587)
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			client, err = smtp.NewClient(conn, s.smtpConfig.Server)
			if err == nil {
				err = client.StartTLS(&tls.Config{
					InsecureSkipVerify: s.smtpConfig.InsecureSkipVerify,
					ServerName:         s.smtpConfig.Server,
				})
			}
		}
	} else {
		// Plain TCP connection; auto-upgrade to STARTTLS if the server advertises it
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			client, err = smtp.NewClient(conn, s.smtpConfig.Server)
			if err == nil {
				if ok, _ := client.Extension("STARTTLS"); ok {
					err = client.StartTLS(&tls.Config{
						InsecureSkipVerify: s.smtpConfig.InsecureSkipVerify,
						ServerName:         s.smtpConfig.Server,
					})
				}
			}
		}
	}

	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return fmt.Errorf("%w: %w", mailexception.ErrComposeSMTPConnect, err)
	}
	defer conn.Close()
	defer client.Close()

	// Send email
	if err = client.Mail(fromEmail); err != nil {
		return fmt.Errorf("%w: %w", mailexception.ErrComposeSMTPSend, err)
	}

	for _, recipient := range recipients {
		recipient = strings.TrimSpace(recipient)
		if recipient != "" {
			if err = client.Rcpt(recipient); err != nil {
				return fmt.Errorf("%w: %w", mailexception.ErrComposeSMTPSend, err)
			}
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("%w: %w", mailexception.ErrComposeSMTPSend, err)
	}

	_, err = w.Write(rawContent)
	if err != nil {
		return fmt.Errorf("%w: %w", mailexception.ErrComposeSMTPSend, err)
	}
	err = client.Quit()
	if err != nil {
		// Check if it's just a response message (not an actual error)
		// SMTP servers often return "250 2.0.0 Ok" as the error string when quitting successfully
		errStr := err.Error()
		if len(errStr) > 0 && (errStr[0] >= '2' && errStr[0] <= '2') {
			// This is a successful SMTP response (2xx), not an actual error
		} else {
			// This is a real error
			return fmt.Errorf("%w: %w", mailexception.ErrComposeSMTPSend, err)
		}
	}

	return nil
}

// processPendingJobs runs in a goroutine and processes outbound emails from the pending channel.
// Each email is sent via SMTP and the delivery status is updated in the database.
func (s *mailService) processPendingJobs(ctx context.Context) {
	for job := range s.pendingCh {
		s.sendOne(ctx, job)
	}
}

// sendOne sends a single pending email via SMTP and updates the DB status.
func (s *mailService) sendOne(ctx context.Context, job pendingJob) {
	err := s.sendSMTPToServer(job.fromEmail, job.rawContent, job.recipients)
	if err != nil {
		log.Printf("Async send failed: emailID=%d userID=%s error=%v", job.emailID, job.userID, err)
		if job.emailID > 0 {
			if dbErr := s.emailRepo.UpdateSMTPStatus(ctx, job.userID, job.emailID, messaging.SMTPStatusFailed, err.Error(), ""); dbErr != nil {
				log.Printf("Failed to update SMTP status to failed: %v", dbErr)
			}
		}
		return
	}
	log.Printf("Async send succeeded: emailID=%d userID=%s", job.emailID, job.userID)
	if job.emailID > 0 {
		if dbErr := s.emailRepo.UpdateSMTPStatus(ctx, job.userID, job.emailID, messaging.SMTPStatusSent, "", time.Now().Format(time.RFC3339)); dbErr != nil {
			log.Printf("Failed to update SMTP status to sent: %v", dbErr)
		}
	}
}

// writeMessageFile writes the raw .eml file to disk after the DB record is saved.
func (s *mailService) writeMessageFile(ctx context.Context, userID shared.GlobalID, email *messaging.Email, rawBody []byte) error {
	root, dataPath, err := s.dataPathForUser(ctx, userID)
	if err != nil {
		return err
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	// Security: ensure the path is within root directory
	absDataPath := filepath.Join(rootAbs, filepath.Clean(dataPath))
	rel, err := filepath.Rel(rootAbs, absDataPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("unsafe data path %q", absDataPath)
	}

	fullPath := filepath.Join(absDataPath, email.FilePath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(fullPath, rawBody, 0644); err != nil {
		return fmt.Errorf("write %s: %w", fullPath, err)
	}
	return nil
}

// generateMessageID generates a Message-ID for an email.
func generateMessageID(domain string) string {
	if strings.Contains(domain, "@") {
		domain = strings.Split(domain, "@")[1]
	}
	domain = strings.TrimSpace(strings.ToLower(domain))

	timestamp := time.Now().UnixNano()
	randBytes := make([]byte, 6)
	rand.Read(randBytes)
	randHex := hex.EncodeToString(randBytes)

	return fmt.Sprintf("<%d.%s@%s>",
		timestamp,
		randHex,
		domain,
	)
}

// buildRawEmailContent constructs a minimal RFC 822 raw email from the compose request.
// Bcc is omitted from headers as it should not be stored in the raw message.
// When there are attachments, the email is built as multipart/mixed with base64-encoded
// attachment parts, so that listAttachmentsFromFile and openAttachmentFromFile can read them.
func buildRawEmailContent(email *messaging.Email, req ComposeRequest) []byte {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", email.Sender))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", email.Recipient))
	if email.CarbonCopy != "" {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", email.CarbonCopy))
	}
	if email.Subject != "" {
		buf.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	}
	// Re-format MailTime (RFC 3339) to RFC 5322 for the Date header
	var dateHeader string
	if t, err := time.Parse(time.RFC3339, email.MailTime); err == nil {
		dateHeader = t.Format(time.RFC1123Z)
	} else {
		dateHeader = email.MailTime // fallback
	}
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", dateHeader))
	buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", generateMessageID(extractDomain(email.Sender))))
	buf.WriteString("MIME-Version: 1.0\r\n")

	body := req.Text
	if body == "" {
		body = req.HTML
	}

	if len(req.Attachments) > 0 {
		// Multipart/mixed email with attachments
		boundary := fmt.Sprintf("=_%d", time.Now().UnixNano())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")
		buf.WriteString("--")
		buf.WriteString(boundary)
		buf.WriteString("\r\n")

		// Body part
		if req.HTML != "" && req.Text == "" {
			buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		} else {
			buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		}
		buf.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(body)
		buf.WriteString("\r\n")

		// Attachment parts
		for _, att := range req.Attachments {
			buf.WriteString("--")
			buf.WriteString(boundary)
			buf.WriteString("\r\n")
			ct := att.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", ct))
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", att.Name))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString("\r\n")

			// Base64 encode the binary data with standard 76-char line wrapping
			encoded := base64.StdEncoding.EncodeToString(att.Data)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				buf.WriteString(encoded[i:end])
				buf.WriteString("\r\n")
			}
			buf.WriteString("\r\n")
		}

		buf.WriteString("--")
		buf.WriteString(boundary)
		buf.WriteString("--\r\n")
	} else {
		// Simple email without attachments
		if req.HTML != "" && req.Text == "" {
			buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		} else {
			buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		}
		buf.WriteString("\r\n")
		buf.WriteString(body)
	}

	return buf.Bytes()
}
