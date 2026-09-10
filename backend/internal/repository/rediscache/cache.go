// Package rediscache implements domain.Cache on top of Redis. It is used
// for cache-aside reads (e.g. property lookups) and for tracking revoked
// refresh tokens so logout/rotation takes effect immediately instead of
// waiting for JWT expiry.
package rediscache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"propertymanagement/internal/domain"
)

type Cache struct {
	client *redis.Client
}

func New(addr, password string, db int) *Cache {
	return &Cache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

var _ domain.Cache = (*Cache)(nil)

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	v, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", fmt.Errorf("redis get %q: %w", key, err)
	}
	return v, nil
}

func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set %q: %w", key, err)
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del %q: %w", key, err)
	}
	return nil
}
