package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/lib/pq"

	"github.com/henry/javapi/internal/domain"
)

// PostgresCache implements domain.Cache using PostgreSQL JSONB storage.
// It connects to a PostgreSQL database and auto-creates the required table on first use.
// When no PostgreSQL is available, the application falls back to MemoryCache instead.
type PostgresCache struct {
	db *sql.DB
}

// NewPostgresCache creates a new PostgresCache connected to the given PostgreSQL DSN.
// It verifies the connection and ensures the search_cache table exists.
func NewPostgresCache(ctx context.Context, connStr string) (*PostgresCache, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if err := createTable(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return &PostgresCache{db: db}, nil
}

// createTable creates the search_cache table and index if they do not exist.
func createTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS search_cache (
			code TEXT PRIMARY KEY,
			response JSONB NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_search_cache_expires_at ON search_cache(expires_at);
	`)
	return err
}

// Get retrieves a cached SearchResponse. Returns nil, false if the key is not found
// or if the cached entry has expired.
func (p *PostgresCache) Get(ctx context.Context, key string) (*domain.SearchResponse, bool) {
	var data []byte
	err := p.db.QueryRowContext(ctx,
		"SELECT response FROM search_cache WHERE code = $1 AND expires_at > NOW()",
		key,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	var resp domain.SearchResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, false
	}
	return &resp, true
}

// Set stores a SearchResponse with the given TTL. Uses UPSERT (ON CONFLICT DO UPDATE)
// so that re-caching the same code overwrites the previous entry.
func (p *PostgresCache) Set(ctx context.Context, key string, value *domain.SearchResponse, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO search_cache (code, response, expires_at)
		 VALUES ($1, $2, NOW() + $3)
		 ON CONFLICT (code) DO UPDATE SET response = $2, expires_at = NOW() + $3`,
		key, data, ttl,
	)
	return err
}

// Delete removes a cached entry by code. No-op if the key does not exist.
func (p *PostgresCache) Delete(ctx context.Context, key string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM search_cache WHERE code = $1", key)
	return err
}

// Close closes the underlying database connection.
func (p *PostgresCache) Close() error {
	return p.db.Close()
}
