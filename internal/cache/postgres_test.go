package cache

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// postgresTestConnStr tries known PG env vars; returns empty string if none set.
func postgresTestConnStr() string {
	for _, env := range []string{"POSTGRES_URL", "DATABASE_URL", "PGURL"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}

// newTestPostgresCache attempts to create a PostgresCache for testing.
// If no PostgreSQL is available, the test is skipped.
func newTestPostgresCache(t *testing.T, ctx context.Context) *PostgresCache {
	t.Helper()

	connStr := postgresTestConnStr()
	if connStr == "" {
		t.Skip("no PostgreSQL connection string set (POSTGRES_URL, DATABASE_URL, or PGURL)")
	}

	cache, err := NewPostgresCache(ctx, connStr)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	// Clean up any leftover test data.
	_, _ = cache.db.ExecContext(ctx, "DELETE FROM search_cache")

	t.Cleanup(func() {
		cache.db.ExecContext(context.Background(), "DELETE FROM search_cache WHERE code LIKE $1", "test-%")
		cache.Close()
	})

	return cache
}

func TestPostgresCache_GetSet(t *testing.T) {
	ctx := context.Background()
	c := newTestPostgresCache(t, ctx)

	original := testResponse("test-abc123")
	err := c.Set(ctx, "test-abc123", original, 5*time.Minute)
	require.NoError(t, err)

	got, found := c.Get(ctx, "test-abc123")
	require.True(t, found)
	require.NotNil(t, got)
	assert.Equal(t, original.Code, got.Code)
	assert.Equal(t, original.Movie.Number, got.Movie.Number)
	assert.Equal(t, original.Movie.Title, got.Movie.Title)
	assert.Equal(t, original.TookMs, got.TookMs)
	assert.Equal(t, original.Cache.Hit, got.Cache.Hit)
}

func TestPostgresCache_GetMissing(t *testing.T) {
	ctx := context.Background()
	c := newTestPostgresCache(t, ctx)

	got, found := c.Get(ctx, "test-nonexistent")
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestPostgresCache_Delete(t *testing.T) {
	ctx := context.Background()
	c := newTestPostgresCache(t, ctx)

	original := testResponse("test-delete-me")
	err := c.Set(ctx, "test-delete-me", original, 5*time.Minute)
	require.NoError(t, err)

	// Verify it exists first
	_, found := c.Get(ctx, "test-delete-me")
	assert.True(t, found)

	// Delete and verify it's gone
	err = c.Delete(ctx, "test-delete-me")
	require.NoError(t, err)

	got, found := c.Get(ctx, "test-delete-me")
	assert.False(t, found)
	assert.Nil(t, got)
}

func TestPostgresCache_Expiry(t *testing.T) {
	ctx := context.Background()
	c := newTestPostgresCache(t, ctx)

	original := testResponse("test-expiry")

	// Use very short TTL
	err := c.Set(ctx, "test-expiry", original, 100*time.Millisecond)
	require.NoError(t, err)

	// Should be immediately available
	_, found := c.Get(ctx, "test-expiry")
	assert.True(t, found)

	// Wait for expiry
	time.Sleep(250 * time.Millisecond)

	got, found := c.Get(ctx, "test-expiry")
	assert.False(t, found, "expected key to have expired")
	assert.Nil(t, got)
}

func TestPostgresCache_Upsert(t *testing.T) {
	ctx := context.Background()
	c := newTestPostgresCache(t, ctx)

	first := testResponse("test-upsert")
	first.TookMs = 100

	err := c.Set(ctx, "test-upsert", first, 5*time.Minute)
	require.NoError(t, err)

	// Upsert with a new value for the same key
	second := testResponse("test-upsert")
	second.TookMs = 200
	err = c.Set(ctx, "test-upsert", second, 5*time.Minute)
	require.NoError(t, err)

	got, found := c.Get(ctx, "test-upsert")
	require.True(t, found)
	assert.Equal(t, int64(200), got.TookMs, "upsert should update existing row")
}

func TestPostgresCache_ConnectionFails(t *testing.T) {
	ctx := context.Background()

	_, err := NewPostgresCache(ctx, "postgres://nonexistent:5432/invalid?connect_timeout=1")
	assert.Error(t, err, "expected connection error for invalid DSN")
}
