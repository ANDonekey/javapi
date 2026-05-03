package av01

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestSearch_SuccessByDVDID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, userAgent, r.Header.Get("User-Agent"))

		var body searchRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "ABC-123", body.Query)
		assert.Equal(t, 1, body.Pagination.Page)
		assert.Equal(t, 24, body.Pagination.Limit)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "vid001", DVDID: "ABC-123", DMMID: "dmm001"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, "av01", r.SiteName)
	assert.Equal(t, domain.StatusSuccess, r.Status)
	assert.Equal(t, domain.VersionOriginal, r.Version)
	assert.Equal(t, srv.URL+"/jp/video/vid001/ABC-123", r.PageURL)
	assert.Empty(t, r.Error)
}

func TestSearch_SuccessByDMMID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "x777", DVDID: "XYZ-999", DMMID: "ABC-123"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, srv.URL+"/jp/video/x777/ABC-123", results[0].PageURL)
}

func TestSearch_SuccessNormalizedMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "n001", DVDID: "abc_123"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, srv.URL+"/jp/video/n001/ABC-123", results[0].PageURL)
}

func TestSearch_NoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "a", DVDID: "OTHER-001"},
				{ID: "b", DVDID: "OTHER-002"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, domain.StatusNotFound, results[0].Status)
	assert.Equal(t, "av01", results[0].SiteName)
}

func TestSearch_EmptyVideos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{})
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

func TestSearch_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("this is not valid json"))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, domain.StatusError, results[0].Status)
	assert.Contains(t, results[0].Error, "invalid response JSON")
}

func TestSearch_EmptyJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearch_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{})
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

func TestSearch_MultipleVideosMatchFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "first", DVDID: "OTHER-001"},
				{ID: "target", DVDID: "ABC-123"},
				{ID: "third", DVDID: "OTHER-002"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, srv.URL+"/jp/video/target/ABC-123", results[0].PageURL)
}

func TestSearch_MatchPrefersFirstResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "a1", DVDID: "ABC-123", DMMID: "dmm-a"},
				{ID: "b2", DVDID: "OTHER", DMMID: "ABC-123"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, srv.URL+"/jp/video/a1/ABC-123", results[0].PageURL)
}

func TestSearch_PageURLInResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "pg001", DVDID: "PAGE-001"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "PAGE-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, srv.URL+"/jp/video/pg001/PAGE-001", results[0].PageURL)
}

func TestSearch_EmptyDVDDAndDMMID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "empty", DVDID: "", DMMID: ""},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearch_NotFoundHasCorrectSiteName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "NONEXIST")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "av01", results[0].SiteName)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearch_NormalizedMatchAcrossSeparators(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "ns1", DVDID: "ABC 123"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)

	t.Run("hyphen_input", func(t *testing.T) {
		results, err := s.Search(context.Background(), "ABC-123")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, domain.StatusSuccess, results[0].Status)
	})

	t.Run("underscore_input", func(t *testing.T) {
		results, err := s.Search(context.Background(), "ABC_123")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, domain.StatusSuccess, results[0].Status)
	})

	t.Run("space_input", func(t *testing.T) {
		results, err := s.Search(context.Background(), "ABC 123")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, domain.StatusSuccess, results[0].Status)
	})

	t.Run("no_separator_input", func(t *testing.T) {
		results, err := s.Search(context.Background(), "abc123")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, domain.StatusSuccess, results[0].Status)
	})
}

func TestSearch_SuccessHasNoVideoSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "ns1", DVDID: "ABC-123"},
			},
		})
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Nil(t, results[0].VideoSources, "AV01 scraper returns page URL only, no embed sources")
}

func TestNewWithClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Videos: []videoEntry{
				{ID: "id", DVDID: "CODE-001"},
			},
		})
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

func helperJSONResponse(videos []videoEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{Videos: videos})
	}
}

func TestSearch_BenchmarkScenarios(t *testing.T) {
	generateVideos := func(n int, withMatch bool) []videoEntry {
		vids := make([]videoEntry, n)
		for i := 0; i < n; i++ {
			vids[i] = videoEntry{
				ID:    fmt.Sprintf("id%d", i),
				DVDID: fmt.Sprintf("OTHER-%03d", i),
			}
		}
		if withMatch {
			vids[n/2] = videoEntry{ID: "match", DVDID: "TARGET-001"}
		}
		return vids
	}

	t.Run("large_response_no_match", func(t *testing.T) {
		srv := httptest.NewServer(helperJSONResponse(generateVideos(100, false)))
		defer srv.Close()

		s := newTestScraper(srv)
		results, err := s.Search(context.Background(), "TARGET-001")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, domain.StatusNotFound, results[0].Status)
	})

	t.Run("large_response_middle_match", func(t *testing.T) {
		srv := httptest.NewServer(helperJSONResponse(generateVideos(100, true)))
		defer srv.Close()

		s := newTestScraper(srv)
		results, err := s.Search(context.Background(), "TARGET-001")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, domain.StatusSuccess, results[0].Status)
		assert.Equal(t, srv.URL+"/jp/video/match/TARGET-001", results[0].PageURL)
	})
}
