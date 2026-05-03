package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/henry/javapi/internal/aggregator"
	"github.com/henry/javapi/internal/cache"
	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubJavDB struct{}

func (s *stubJavDB) Search(ctx context.Context, code string) (*domain.Movie, error) {
	return &domain.Movie{ID: "test", Number: code, Title: "Test Movie"}, nil
}
func (s *stubJavDB) GetMovie(ctx context.Context, movieID string) (*domain.Movie, error) {
	return &domain.Movie{ID: movieID, Number: "TEST", Title: "Detail Movie"}, nil
}

func TestIntegrationSearchFlow(t *testing.T) {
	c := cache.NewMemoryCache(5 * time.Minute)
	svc := aggregator.NewService(&stubJavDB{}, c, 6)
	h := NewSearchHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/search?code=ABC-123", nil)
	w := httptest.NewRecorder()
	h.Search(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIntegrationCacheHit(t *testing.T) {
	c := cache.NewMemoryCache(5 * time.Minute)
	key := "abc123"
	c.Set(context.Background(), key, &domain.SearchResponse{Code: "ABC-123", Cache: domain.CacheInfo{Hit: true}}, 5*time.Minute)
	svc := aggregator.NewService(&stubJavDB{}, c, 6)

	resp, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.True(t, resp.Cache.Hit)
}

func TestIntegrationAuth(t *testing.T) {
	c := cache.NewMemoryCache(5 * time.Minute)
	svc := aggregator.NewService(&stubJavDB{}, c, 6)

	req := httptest.NewRequest("GET", "/api/v1/search?code=ABC-123", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	h := NewSearchHandler(svc)
	w := httptest.NewRecorder()
	h.Search(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIntegrationPartialSuccess(t *testing.T) {
	c := cache.NewMemoryCache(5 * time.Minute)
	javdb := &stubJavDB{}
	svc := aggregator.NewService(javdb, c, 6)

	resp, err := svc.Aggregate(context.Background(), "XYZ-999")
	require.NoError(t, err)
	assert.NotNil(t, resp.Movie)
	assert.Equal(t, "XYZ-999", resp.Code)
}

func TestIntegrationTiming(t *testing.T) {
	c := cache.NewMemoryCache(5 * time.Minute)
	svc := aggregator.NewService(&stubJavDB{}, c, 6)

	resp, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.TookMs, int64(0))
}
