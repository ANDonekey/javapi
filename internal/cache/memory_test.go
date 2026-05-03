package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/henry/javapi/internal/domain"
)

func testResponse(code string) *domain.SearchResponse {
	return &domain.SearchResponse{
		Code: code,
		Movie: &domain.Movie{
			Number: code,
			Title:  "Test Movie",
		},
		Videos: []domain.VideoResult{},
		Cache:  domain.CacheInfo{Hit: false},
		TookMs: 100,
	}
}

func TestMemoryCache_GetSet(t *testing.T) {
	c := NewMemoryCache(5 * time.Minute)
	ctx := context.Background()

	original := testResponse("abc123")
	err := c.Set(ctx, "abc123", original, 5*time.Minute)
	require.NoError(t, err)

	got, found := c.Get(ctx, "abc123")
	require.True(t, found)
	require.NotNil(t, got)
	assert.Equal(t, original.Code, got.Code)
	assert.Equal(t, original.Movie.Number, got.Movie.Number)
	assert.Equal(t, original.Movie.Title, got.Movie.Title)
	assert.Equal(t, original.TookMs, got.TookMs)
	assert.Equal(t, original.Cache.Hit, got.Cache.Hit)
}

func TestMemoryCache_GetMissing(t *testing.T) {
	c := NewMemoryCache(5 * time.Minute)
	ctx := context.Background()

	got, found := c.Get(ctx, "nonexistent")
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestMemoryCache_Delete(t *testing.T) {
	c := NewMemoryCache(5 * time.Minute)
	ctx := context.Background()

	original := testResponse("abc123")
	err := c.Set(ctx, "abc123", original, 5*time.Minute)
	require.NoError(t, err)

	// Verify it exists first
	_, found := c.Get(ctx, "abc123")
	assert.True(t, found)

	// Delete and verify it's gone
	err = c.Delete(ctx, "abc123")
	require.NoError(t, err)

	got, found := c.Get(ctx, "abc123")
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestMemoryCache_Expiry(t *testing.T) {
	c := NewMemoryCache(100 * time.Millisecond)
	ctx := context.Background()

	original := testResponse("abc123")

	// Use short TTL
	err := c.Set(ctx, "abc123", original, 100*time.Millisecond)
	require.NoError(t, err)

	// Should be immediately available
	_, found := c.Get(ctx, "abc123")
	assert.True(t, found)

	// Wait for expiry
	time.Sleep(250 * time.Millisecond)

	got, found := c.Get(ctx, "abc123")
	assert.False(t, found, "expected key to have expired")
	assert.Nil(t, got)
}

func TestMemoryCache_CacheHit(t *testing.T) {
	c := NewMemoryCache(5 * time.Minute)
	ctx := context.Background()

	// Miss on non-existent key
	_, hit := c.Get(ctx, "nosuchkey")
	assert.False(t, hit, "expected cache miss")

	// Set and verify hit
	original := testResponse("hit-test")
	err := c.Set(ctx, "hit-test", original, 5*time.Minute)
	require.NoError(t, err)

	_, hit = c.Get(ctx, "hit-test")
	assert.True(t, hit, "expected cache hit")
}
