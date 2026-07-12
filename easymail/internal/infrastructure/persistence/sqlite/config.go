package sqlite

// Config holds SQLite open parameters.
type Config struct {
	BusyTimeoutMs int
	MaxOpenConns  int
	MaxIdleConns  int
	WAL           bool
}

func (c Config) busyTimeout() int {
	if c.BusyTimeoutMs <= 0 {
		return 5000
	}
	return c.BusyTimeoutMs
}

func (c Config) maxOpen() int {
	if c.MaxOpenConns <= 0 {
		return 1
	}
	return c.MaxOpenConns
}

func (c Config) maxIdle() int {
	if c.MaxIdleConns <= 0 {
		if c.WAL {
			return 4
		}
		return 1
	}
	return c.MaxIdleConns
}
