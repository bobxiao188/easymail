package persistence

import "context"

// DBProvider is the abstract database port.
// Domain and application layers depend on this interface, not on GORM or any specific DB.
type DBProvider interface {
	// DB returns a database connection handle.
	// The concrete type depends on the driver implementation (e.g., *gorm.DB, *sql.DB).
	DB(ctx context.Context) (any, error)

	// Ping checks connectivity.
	Ping(ctx context.Context) error

	// Close shuts down the connection pool gracefully.
	Close() error
}
