package domain

import (
	"context"
	"time"
)

// Cache defines the interface for caching search responses.
type Cache interface {
	// Get retrieves a cached search response. Returns nil, false if not found.
	Get(ctx context.Context, key string) (*SearchResponse, bool)

	// Set stores a search response with the given TTL.
	Set(ctx context.Context, key string, value *SearchResponse, ttl time.Duration) error

	// Delete removes a cached entry by key.
	Delete(ctx context.Context, key string) error
}
