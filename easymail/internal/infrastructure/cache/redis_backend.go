package cache

import (
	"context"
	"errors"
	"time"

	"easymail/internal/infrastructure/persistence"
	redisrepo "easymail/internal/infrastructure/persistence/redis"

	"github.com/redis/go-redis/v9"
)

type RedisBackend struct {
	client *redis.Client
}

func NewRedisBackend(client *redis.Client) *RedisBackend {
	return &RedisBackend{client: client}
}

func (r *RedisBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (r *RedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisBackend) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisBackend) Close() error {
	return r.client.Close()
}

// NewRedisBackendFromProvider creates a RedisBackend from an abstract RedisProvider.
// Returns an error if the underlying provider is not backed by go-redis.
func NewRedisBackendFromProvider(rp persistence.RedisProvider) (*RedisBackend, error) {
	client, err := redisrepo.GoRedis(rp)
	if err != nil {
		return nil, err
	}
	return &RedisBackend{client: client}, nil
}

