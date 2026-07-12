package sqlite

import "testing"

func TestConfigDefaults(t *testing.T) {
	c := Config{}
	if got := c.busyTimeout(); got != 5000 {
		t.Errorf("busyTimeout() = %d, want 5000", got)
	}
	if got := c.maxOpen(); got != 1 {
		t.Errorf("maxOpen() = %d, want 1", got)
	}
	// WAL off -> maxIdle defaults to 1
	if got := c.maxIdle(); got != 1 {
		t.Errorf("maxIdle() with WAL=false = %d, want 1", got)
	}
}

func TestConfigWALMaxIdle(t *testing.T) {
	c := Config{WAL: true}
	if got := c.maxIdle(); got != 4 {
		t.Errorf("maxIdle() with WAL=true = %d, want 4", got)
	}
}

func TestConfigExplicitValues(t *testing.T) {
	c := Config{
		BusyTimeoutMs: 10000,
		MaxOpenConns:  5,
		MaxIdleConns:  3,
		WAL:           true,
	}
	if got := c.busyTimeout(); got != 10000 {
		t.Errorf("busyTimeout() = %d, want 10000", got)
	}
	if got := c.maxOpen(); got != 5 {
		t.Errorf("maxOpen() = %d, want 5", got)
	}
	if got := c.maxIdle(); got != 3 {
		t.Errorf("maxIdle() = %d, want 3", got)
	}
}
