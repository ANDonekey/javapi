package cache

import (
	"context"
	"time"

	gocache "github.com/patrickmn/go-cache"

	"github.com/henry/javapi/internal/domain"
)

// MemoryCache implements domain.Cache using an in-memory store.
// It wraps github.com/patrickmn/go-cache for TTL-based expiration.
type MemoryCache struct {
	c *gocache.Cache
}

// NewMemoryCache creates a MemoryCache with the given default TTL.
// The cleanup interval is set to 2x the default TTL for periodic eviction.
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	return &MemoryCache{
		c: gocache.New(defaultTTL, defaultTTL*2),
	}
}

// Get retrieves a cached SearchResponse. Returns nil, false if not found
// or if the cached value is not of type *domain.SearchResponse.
func (m *MemoryCache) Get(_ context.Context, key string) (*domain.SearchResponse, bool) {
	val, found := m.c.Get(key)
	if !found {
		return nil, false
	}
	resp, ok := val.(*domain.SearchResponse)
	if !ok {
		return nil, false
	}
	return resp, true
}

// Set stores a SearchResponse with the given TTL. If ttl is 0, the cache's
// default expiration is used.
func (m *MemoryCache) Set(_ context.Context, key string, value *domain.SearchResponse, ttl time.Duration) error {
	m.c.Set(key, value, ttl)
	return nil
}

// Delete removes a cached entry by key. No-op if the key does not exist.
func (m *MemoryCache) Delete(_ context.Context, key string) error {
	m.c.Delete(key)
	return nil
}
