package av01

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestScraper(srv *httptest.Server) *Scraper {
	return NewWithClient(srv.Client(), srv.URL)
}

func jsonSearchResponse(videos ...searchVideo) []byte {
	b, _ := json.Marshal(struct {
		Videos []searchVideo `json:"videos"`
	}{Videos: videos})
	return b
}

func TestName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	s := newTestScraper(srv)
	assert.Equal(t, "av01", s.Name())
}

func TestFormatCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	s := newTestScraper(srv)
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC-123", "ABC-123"},
		{"abc123", "abc123"},
		{"SSIS-001", "SSIS-001"},
		{"MIDE-999", "MIDE-999"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, s.FormatCode(tt.input))
		})
	}
}

func TestIsEnabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	s := newTestScraper(srv)
	assert.True(t, s.IsEnabled())

	s.enabled = false
	assert.False(t, s.IsEnabled())
}

func TestRequiresCFBypass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	s := newTestScraper(srv)
	assert.False(t, s.RequiresCFBypass())
}

func TestGetProxyConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	s := newTestScraper(srv)
	cfg := s.GetProxyConfig()
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.URL)
}

func TestImplementsScraperInterface(t *testing.T) {
	var s domain.Scraper = NewWithClient(http.DefaultClient, "http://example.com")
	assert.NotNil(t, s)
	assert.Equal(t, "av01", s.Name())
}

func TestInitRegistration(t *testing.T) {
	all := scraper.GetAll()
	found := false
	for _, s := range all {
		if s.Name() == "av01" {
			found = true
			break
		}
	}
	assert.True(t, found, "av01 scraper should be registered via init()")
}

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/videos/search", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "lang=cn&comp=true", r.URL.RawQuery)

		bodyBytes, _ := io.ReadAll(r.Body)
		var req searchRequest
		json.Unmarshal(bodyBytes, &req)
		assert.Equal(t, "MIDA-492", req.Query)
		assert.Equal(t, 20, req.Pagination.Limit)
		assert.Equal(t, 1, req.Pagination.Page)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse(searchVideo{ID: 203184, DvdID: "MIDA-492", DmmID: "mida00492"}))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "MIDA-492")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, "av01", r.SiteName)
	assert.Equal(t, domain.StatusSuccess, r.Status)
	assert.Equal(t, domain.VersionOriginal, r.Version)
	assert.Contains(t, r.PageURL, "/cn/video/203184/mida-492")
	require.Len(t, r.VideoSources, 1)
	assert.Equal(t, "application/x-mpegURL", r.VideoSources[0].Type)
	assert.Contains(t, r.VideoSources[0].URL, "/api/v1/videos/203184/manifest/master.m3u8")
	assert.Empty(t, r.Error)
}

func TestSearch_SuccessMatchByDmmID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse(searchVideo{ID: 42, DvdID: "OTHER-001", DmmID: "TEST-042"}))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "TEST-042")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Contains(t, results[0].VideoSources[0].URL, "/api/v1/videos/42/manifest/master.m3u8")
}

func TestSearch_SuccessMatchByNormalizedCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse(searchVideo{ID: 77, DvdID: "ABC_123", DmmID: ""}))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestSearch_SuccessWithMixedCase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse(searchVideo{ID: 99, DvdID: "ssis-001", DmmID: ""}))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "SSIS-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestSearch_NoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse(
			searchVideo{ID: 1, DvdID: "OTHER-001", DmmID: ""},
			searchVideo{ID: 2, DvdID: "OTHER-002", DmmID: ""},
		))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
	assert.Equal(t, "av01", results[0].SiteName)
}

func TestSearch_EmptySearchResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse())
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
	assert.Contains(t, results[0].Error, "HTTP 500")
}

func TestSearch_HTTPNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
	assert.Contains(t, results[0].Error, "HTTP 404")
}

func TestSearch_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse())
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := newTestScraper(srv)
	results, err := s.Search(ctx, "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
	assert.Contains(t, results[0].Error, "request failed")
}

func TestSearch_DisabledScraper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	s := newTestScraper(srv)
	s.enabled = false

	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scraper is disabled")
}

func TestSearch_MatchesFirstResultOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse(
			searchVideo{ID: 1, DvdID: "DUP-001", DmmID: ""},
			searchVideo{ID: 2, DvdID: "DUP-001", DmmID: ""},
		))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "DUP-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].PageURL, "/cn/video/1/dup-001")
	assert.Contains(t, results[0].VideoSources[0].URL, "/api/v1/videos/1/manifest/master.m3u8")
}

func TestNewWithClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonSearchResponse(searchVideo{ID: 1, DvdID: "CODE-001", DmmID: ""}))
	}))
	defer srv.Close()

	s := NewWithClient(srv.Client(), srv.URL)
	assert.Equal(t, "av01", s.Name())
	assert.True(t, s.IsEnabled())
	assert.False(t, s.RequiresCFBypass())
	assert.Empty(t, s.GetProxyConfig().URL)

	results, err := s.Search(context.Background(), "CODE-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].PageURL, srv.URL)
}
