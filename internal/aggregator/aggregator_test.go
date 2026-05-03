package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockJavDB struct {
	searchFn   func(ctx context.Context, code string) (*domain.Movie, error)
	getMovieFn func(ctx context.Context, movieID string) (*domain.Movie, error)
}

func (m *mockJavDB) Search(ctx context.Context, code string) (*domain.Movie, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, code)
	}
	return nil, nil
}
func (m *mockJavDB) GetMovie(ctx context.Context, movieID string) (*domain.Movie, error) {
	if m.getMovieFn != nil {
		return m.getMovieFn(ctx, movieID)
	}
	return nil, nil
}

type mockCache struct {
	data map[string]*domain.SearchResponse
}

func newMockCache() *mockCache { return &mockCache{data: make(map[string]*domain.SearchResponse)} }
func (m *mockCache) Get(ctx context.Context, key string) (*domain.SearchResponse, bool) {
	v, ok := m.data[key]
	return v, ok
}
func (m *mockCache) Set(ctx context.Context, key string, value *domain.SearchResponse, ttl time.Duration) error {
	m.data[key] = value
	return nil
}
func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func testMovie(code string) *domain.Movie {
	return &domain.Movie{ID: "id-" + code, Number: code, Title: "Test " + code}
}

func TestCacheHit(t *testing.T) {
	cache := newMockCache()
	cacheKey := "abc123"
	expected := &domain.SearchResponse{Code: "ABC-123", Cache: domain.CacheInfo{Hit: true}}
	cache.data[cacheKey] = expected

	svc := NewService(&mockJavDB{}, cache, 6)
	resp, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.True(t, resp.Cache.Hit)
}

func TestCacheMiss(t *testing.T) {
	cache := newMockCache()
	javdb := &mockJavDB{
		searchFn: func(ctx context.Context, code string) (*domain.Movie, error) {
			return testMovie(code), nil
		},
	}
	svc := NewService(javdb, cache, 6)
	resp, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.False(t, resp.Cache.Hit)
	assert.NotNil(t, resp.Movie)
	assert.Equal(t, "Test ABC-123", resp.Movie.Title)

	resp2, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.True(t, resp2.Cache.Hit)
}

func TestPartialSuccess(t *testing.T) {
	cache := newMockCache()
	javdb := &mockJavDB{
		searchFn: func(ctx context.Context, code string) (*domain.Movie, error) {
			return testMovie(code), nil
		},
		getMovieFn: func(ctx context.Context, movieID string) (*domain.Movie, error) {
			return testMovie("ABC-123"), nil
		},
	}
	svc := NewService(javdb, cache, 6)
	resp, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.NotNil(t, resp.Movie)
}

func TestAllFail(t *testing.T) {
	cache := newMockCache()
	javdb := &mockJavDB{
		searchFn: func(ctx context.Context, code string) (*domain.Movie, error) {
			return nil, errors.New("down")
		},
	}
	svc := NewService(javdb, cache, 6)
	_, err := svc.Aggregate(context.Background(), "XYZ-999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all sources failed")
}

func TestMultiVersion(t *testing.T) {
	cache := newMockCache()
	cacheKey := "abc123"
	resp := &domain.SearchResponse{
		Code: "ABC-123",
		Videos: []domain.VideoResult{
			{SiteName: "test", Status: domain.StatusSuccess, Version: domain.VersionOriginal},
			{SiteName: "test", Status: domain.StatusSuccess, Version: domain.VersionCNSub},
			{SiteName: "test", Status: domain.StatusSuccess, Version: domain.VersionMosaicReduce},
		},
	}
	cache.data[cacheKey] = resp
	svc := NewService(&mockJavDB{}, cache, 6)
	result, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.Len(t, result.Videos, 3)
}

func TestJavDBOnly(t *testing.T) {
	cache := newMockCache()
	javdb := &mockJavDB{
		searchFn: func(ctx context.Context, code string) (*domain.Movie, error) {
			return testMovie(code), nil
		},
		getMovieFn: func(ctx context.Context, movieID string) (*domain.Movie, error) {
			return testMovie("ABC-123"), nil
		},
	}
	svc := NewService(javdb, cache, 6)
	resp, err := svc.Aggregate(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.NotNil(t, resp.Movie)
	assert.Equal(t, "Test ABC-123", resp.Movie.Title)
}
