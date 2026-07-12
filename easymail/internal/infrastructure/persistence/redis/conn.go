package redis

import (
	"context"
	"errors"
	"time"

	"easymail/internal/infrastructure/persistence"

	"github.com/redis/go-redis/v9"
)

func init() {
	persistence.RegisterRedis(Factory{})
}

// Factory creates standalone Redis connections.
type Factory struct{}

func (Factory) Driver() string { return "standalone" }

func (Factory) Open(ctx context.Context, addr string) (persistence.RedisProvider, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &provider{client: client}, nil
}

type provider struct {
	client *redis.Client
}

func (p *provider) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func (p *provider) Close() error {
	return p.client.Close()
}

// ErrNotRedisProvider is returned when GoRedis is called with a non-redis provider.
var ErrNotRedisProvider = errors.New("RedisProvider is not backed by go-redis")

// GoRedis extracts the underlying *redis.Client from a RedisProvider.
// Returns ErrNotRedisProvider if the provider is not a redis-backed implementation.
func GoRedis(rp persistence.RedisProvider) (*redis.Client, error) {
	p, ok := rp.(*provider)
	if !ok {
		return nil, ErrNotRedisProvider
	}
	return p.client, nil
}
