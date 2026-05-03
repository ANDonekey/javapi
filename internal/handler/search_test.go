package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/henry/javapi/internal/aggregator"
	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCache struct{}

func (t *testCache) Get(ctx context.Context, key string) (*domain.SearchResponse, bool) {
	return nil, false
}
func (t *testCache) Set(ctx context.Context, key string, value *domain.SearchResponse, ttl time.Duration) error {
	return nil
}
func (t *testCache) Delete(ctx context.Context, key string) error {
	return nil
}

type testJavDB struct{}

func (t *testJavDB) Search(ctx context.Context, code string) (*domain.Movie, error) {
	return &domain.Movie{ID: "id-" + code, Number: code, Title: "Test " + code}, nil
}
func (t *testJavDB) GetMovie(ctx context.Context, movieID string) (*domain.Movie, error) {
	return &domain.Movie{ID: movieID, Title: "Test"}, nil
}

func newTestService() *aggregator.Service {
	return aggregator.NewService(&testJavDB{}, &testCache{}, 6)
}

func TestSearchSuccess(t *testing.T) {
	h := NewSearchHandler(newTestService())
	req := httptest.NewRequest("GET", "/api/v1/search?code=ABC-123", nil)
	w := httptest.NewRecorder()
	h.Search(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp domain.SearchResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ABC-123", resp.Code)
}

func TestSearchMissingCode(t *testing.T) {
	h := NewSearchHandler(newTestService())
	req := httptest.NewRequest("GET", "/api/v1/search", nil)
	w := httptest.NewRecorder()
	h.Search(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchInvalidCode(t *testing.T) {
	h := NewSearchHandler(newTestService())
	req := httptest.NewRequest("GET", "/api/v1/search?code=<script>", nil)
	w := httptest.NewRecorder()
	h.Search(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchEmptyCode(t *testing.T) {
	h := NewSearchHandler(newTestService())
	req := httptest.NewRequest("GET", "/api/v1/search?code=", nil)
	w := httptest.NewRecorder()
	h.Search(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
