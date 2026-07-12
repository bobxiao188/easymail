package mysql

import (
	"context"
	"time"

	"easymail/internal/infrastructure/persistence"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultMaxOpenConns    = 50
	defaultMaxIdleConns    = 10
	defaultConnMaxLifetime = time.Hour
	defaultConnMaxIdleTime = 10 * time.Minute
)

func init() {
	persistence.Register(ConnectionFactory{})
}

// ConnectionFactory creates MySQL-backed DBProvider instances.
type ConnectionFactory struct{}

func (ConnectionFactory) Driver() string { return "mysql" }

func (ConnectionFactory) Open(ctx context.Context, dsn string) (persistence.DBProvider, error) {
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Warn),
		NowFunc:     func() time.Time { return time.Now().Local() },
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(defaultMaxOpenConns)
	sqlDB.SetMaxIdleConns(defaultMaxIdleConns)
	sqlDB.SetConnMaxLifetime(defaultConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	return persistence.NewGormProvider(gdb), nil
}
