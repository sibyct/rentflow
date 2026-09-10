package domain

import (
	"context"
	"time"
)

// Cache is the port a generic cache-aside store must satisfy. It is
// implemented by internal/repository/rediscache and consumed by the
// service layer; the domain has no knowledge that Redis exists.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
