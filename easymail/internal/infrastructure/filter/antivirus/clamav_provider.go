package antivirus

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"easymail/internal/domain/filter/antivirus"
)

const (
	defaultChunkSize         = 64 * 1024
	defaultConnectTimeout    = 5 * time.Second
	defaultPoolSize          = 4
	defaultPoolIdleTimeout   = 30 * time.Second
	circuitBreakerThreshold  = 3
	circuitBreakerResetAfter = 30 * time.Second
)

var (
	errTimeout     = errors.New("operation timeout")
	errProtocol    = errors.New("clamd protocol error")
	errCircuitOpen = errors.New("clamav circuit breaker open")
)

// ClamAVProvider implements antivirus.AntivirusEngine backed by a clamd daemon.
// Uses a connection pool for reuse across Scan calls.
type ClamAVProvider struct {
	cfg       Config
	pool      *connPool
	failures  atomic.Int64
	lastFail  atomic.Value // time.Time
	mu        sync.Mutex
	circuitAt time.Time
}

// connPool manages a pool of reusable ClamAV connections.
type connPool struct {
	mu       sync.Mutex
	addr     string
	timeout  time.Duration
	conns    []*clamConn
	maxSize  int
	idleTime time.Duration
}

func newConnPool(addr string, timeout time.Duration, maxSize int) *connPool {
	if maxSize <= 0 {
		maxSize = defaultPoolSize
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &connPool{
		addr:     addr,
		timeout:  timeout,
		conns:    make([]*clamConn, 0, maxSize),
		maxSize:  maxSize,
		idleTime: defaultPoolIdleTimeout,
	}
}

func (p *connPool) get(ctx context.Context) (*clamConn, error) {
	p.mu.Lock()
	// Pop the last idle connection
	if n := len(p.conns); n > 0 {
		c := p.conns[n-1]
		p.conns = p.conns[:n-1]
		p.mu.Unlock()
		// Check if the connection is still alive
		_ = c.raw.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if _, err := c.bufReader.Peek(1); err == nil {
			return c, nil
		}
		c.close()
		// Connection is stale, create a new one
		return p.dial(ctx)
	}
	p.mu.Unlock()
	return p.dial(ctx)
}

func (p *connPool) put(c *clamConn) {
	if c == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.conns) >= p.maxSize {
		c.close()
		return
	}
	_ = c.raw.SetReadDeadline(time.Now().Add(p.idleTime))
	p.conns = append(p.conns, c)
}

func (p *connPool) dial(ctx context.Context) (*clamConn, error) {
	dialer := &net.Dialer{Timeout: defaultConnectTimeout}
	raw, err := dialer.DialContext(ctx, "tcp", p.addr)
	if err != nil {
		return nil, fmt.Errorf("ClamAV connect to %s: %w", p.addr, err)
	}
	return &clamConn{
		raw:       raw,
		bufReader: bufio.NewReaderSize(raw, 64*1024),
		writer:    bufio.NewWriterSize(raw, 64*1024),
		timeout:   p.timeout,
		pool:      p,
	}, nil
}

func (p *connPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		c.close()
	}
	p.conns = nil
}

var _ antivirus.AntivirusEngine = (*ClamAVProvider)(nil)

func NewClamAVProvider(cfg Config) *ClamAVProvider {
	t := cfg.Timeout
	if t <= 0 {
		t = 30 * time.Second
	}
	addr := cfg.Addr
	if addr == "" {
		addr = "127.0.0.1:3310"
	}
	p := &ClamAVProvider{
		cfg:  cfg,
		pool: newConnPool(addr, t, defaultPoolSize),
	}
	return p
}

// isCircuitOpen checks if the circuit breaker is open (too many failures).
func (p *ClamAVProvider) isCircuitOpen() bool {
	failures := p.failures.Load()
	if failures < int64(circuitBreakerThreshold) {
		return false
	}
	// Check if enough time has passed to reset
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.circuitAt.IsZero() && time.Since(p.circuitAt) > circuitBreakerResetAfter {
		p.failures.Store(0)
		p.circuitAt = time.Time{}
		return false
	}
	return true
}

func (p *ClamAVProvider) recordFailure() {
	p.failures.Add(1)
	if p.failures.Load() >= int64(circuitBreakerThreshold) {
		p.mu.Lock()
		if p.circuitAt.IsZero() {
			p.circuitAt = time.Now()
		}
		p.mu.Unlock()
	}
}

func (p *ClamAVProvider) Ping(ctx context.Context) error {
	if p.isCircuitOpen() {
		return errCircuitOpen
	}
	conn, err := p.pool.get(ctx)
	if err != nil {
		p.recordFailure()
		return err
	}
	err = conn.ping(ctx)
	p.pool.put(conn)
	return err
}

func (p *ClamAVProvider) Version(ctx context.Context) (string, error) {
	if p.isCircuitOpen() {
		return "", errCircuitOpen
	}
	conn, err := p.pool.get(ctx)
	if err != nil {
		p.recordFailure()
		return "", err
	}
	ver, err := conn.version(ctx)
	p.pool.put(conn)
	return ver, err
}

func (p *ClamAVProvider) Scan(ctx context.Context, req *antivirus.VirusScanRequest) (*antivirus.VirusScanResult, error) {
	if len(req.Data) == 0 {
		return &antivirus.VirusScanResult{ScanOK: true}, nil
	}
	if p.isCircuitOpen() {
		return nil, errCircuitOpen
	}
	conn, err := p.pool.get(ctx)
	if err != nil {
		p.recordFailure()
		return nil, err
	}
	result, err := conn.scanStream(ctx, req.Data)
	if err != nil {
		p.recordFailure()
		conn.close() // Don't return broken connection to pool
		return nil, err
	}
	p.pool.put(conn)
	return result, nil
}

func (p *ClamAVProvider) Close() error {
	p.pool.close()
	return nil
}

type clamConn struct {
	raw       net.Conn
	bufReader *bufio.Reader
	writer    *bufio.Writer
	timeout   time.Duration
	pool      *connPool
}

func (c *clamConn) close() {
	_ = c.raw.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, _ = c.writer.WriteString("QUIT\n")
	_ = c.writer.Flush()
	_ = c.raw.Close()
}

func (c *clamConn) ping(ctx context.Context) error {
	if err := c.sendCommand(ctx, "PING"); err != nil {
		return err
	}
	reply, err := c.readReply(ctx)
	if err != nil {
		return err
	}
	if reply == "PONG" {
		return nil
	}
	return fmt.Errorf("%w: expected PONG, got: %s", errProtocol, reply)
}

func (c *clamConn) version(ctx context.Context) (string, error) {
	if err := c.sendCommand(ctx, "VERSION"); err != nil {
		return "", err
	}
	reply, err := c.readReply(ctx)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(reply, "ERROR") {
		return "", fmt.Errorf("%w: %s", errProtocol, reply)
	}
	return reply, nil
}

func (c *clamConn) scanStream(ctx context.Context, data []byte) (*antivirus.VirusScanResult, error) {
	if err := c.sendCommand(ctx, "INSTREAM"); err != nil {
		return nil, fmt.Errorf("send INSTREAM: %w", err)
	}
	if err := c.sendDataChunks(ctx, data); err != nil {
		return nil, fmt.Errorf("send data: %w", err)
	}
	// Zero-length end chunk
	if _, err := c.raw.Write([]byte{0, 0, 0, 0}); err != nil {
		return nil, fmt.Errorf("send end chunk: %w", err)
	}
	reply, err := c.readReply(ctx)
	if err != nil {
		return nil, fmt.Errorf("read scan result: %w", err)
	}
	return parseResult(reply), nil
}

func (c *clamConn) sendCommand(ctx context.Context, cmd string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	_ = c.raw.SetWriteDeadline(time.Now().Add(c.timeout))
	ncmd := "n" + cmd + "\n"
	if _, err := c.writer.WriteString(ncmd); err != nil {
		return fmt.Errorf("write command: %w", err)
	}
	return c.writer.Flush()
}

func (c *clamConn) sendDataChunks(ctx context.Context, data []byte) error {
	chunkSize := defaultChunkSize
	for offset := 0; offset < len(data); {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[offset:end]
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(chunk)))
		if _, err := c.raw.Write(length[:]); err != nil {
			return fmt.Errorf("write chunk length: %w", err)
		}
		if len(chunk) > 0 {
			if _, err := c.raw.Write(chunk); err != nil {
				return fmt.Errorf("write chunk data: %w", err)
			}
		}
		offset = end
	}
	return nil
}

func (c *clamConn) readReply(ctx context.Context) (string, error) {
	_ = c.raw.SetReadDeadline(time.Now().Add(c.timeout))
	line, err := c.bufReader.ReadString('\n')
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return "", fmt.Errorf("%w: read timeout", errTimeout)
		}
		if err == io.EOF {
			return "", fmt.Errorf("connection closed by server: %w", err)
		}
		return "", fmt.Errorf("read reply: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func parseResult(reply string) *antivirus.VirusScanResult {
	r := &antivirus.VirusScanResult{RawReply: reply}
	parts := strings.SplitN(reply, ": ", 2)
	if len(parts) != 2 {
		r.ScanOK = false
		r.Error = fmt.Errorf("%w: invalid reply: %s", errProtocol, reply)
		return r
	}
	response := strings.TrimSpace(parts[1])
	switch {
	case strings.HasSuffix(response, "OK"):
		r.ScanOK = true
	case strings.HasSuffix(response, "FOUND"):
		r.IsVirus = true
		r.VirusName = strings.TrimSuffix(response, " FOUND")
	case strings.HasSuffix(response, "ERROR"):
		r.Error = fmt.Errorf("%w: %s", errProtocol, reply)
	default:
		r.Error = fmt.Errorf("%w: unknown response: %s", errProtocol, reply)
	}
	return r
}
