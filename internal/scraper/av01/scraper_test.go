package av01

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/cn/search" {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
			bodyBytes, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(bodyBytes), "q=MIDA-492")

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body>
				<a href="/cn/video/203184/mida-492">MIDA-492</a>
			</body></html>`))
			return
		}

		if strings.HasPrefix(path, "/cn/video/") {
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><script>var src="/api/v1/videos/203184/manifest/master.m3u8";</script></html>`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
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
	assert.Equal(t, srv.URL+"/cn/video/203184/mida-492", r.PageURL)
	assert.Len(t, r.VideoSources, 1)
	assert.Equal(t, "application/x-mpegURL", r.VideoSources[0].Type)
	assert.Contains(t, r.VideoSources[0].URL, "master.m3u8")
	assert.Empty(t, r.Error)
}

func TestSearch_SuccessWithM3U8InSourceTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/42/test-001">TEST-001</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<video><source src="/api/v1/videos/42/manifest/master.m3u8" type="application/x-mpegURL"></video>`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "TEST-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Len(t, results[0].VideoSources, 1)
	assert.Contains(t, results[0].VideoSources[0].URL, "master.m3u8")
}

func TestSearch_SuccessWithNormalizedMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/77/abc-123">ABC 123</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><script>var u="/api/v1/videos/77/manifest/master.m3u8";</script></html>`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestSearch_NoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cn/search", r.URL.Path)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>
			<a href="/cn/video/1/other-001">OTHER-001</a>
			<a href="/cn/video/2/other-002">OTHER-002</a>
		</body></html>`))
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
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><p>No results found</p></body></html>`))
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

func TestSearch_FallbackToConstructedM3U8(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/999/fall-001">FALL-001</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body><p>No video player here</p></body></html>`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "FALL-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Len(t, results[0].VideoSources, 1)
	assert.Equal(t, "application/x-mpegURL", results[0].VideoSources[0].Type)
	assert.Contains(t, results[0].VideoSources[0].URL, "/api/v1/videos/999/manifest/master.m3u8")
}

func TestSearch_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html></html>`))
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

func TestSearch_MatchesByCodeInHref(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/555/my-code-001">Some Title</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<script>"/api/v1/videos/555/manifest/master.m3u8"</script>`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "MY-CODE-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, srv.URL+"/cn/video/555/my-code-001", results[0].PageURL)
}

func TestSearch_MatchesFirstResultOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html>
				<a href="/cn/video/1/dup-001">DUP-001</a>
				<a href="/cn/video/2/dup-001">DUP-001</a>
			</html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`/api/v1/videos/1/manifest/master.m3u8`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "DUP-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, srv.URL+"/cn/video/1/dup-001", results[0].PageURL)
}

func TestSearch_VideoPageErrorUsesFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/88/vperr-001">VPERR-001</a></html>`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "VPERR-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Len(t, results[0].VideoSources, 1)
	assert.Contains(t, results[0].VideoSources[0].URL, "/api/v1/videos/88/manifest/master.m3u8")
}

func TestSearch_NoVideoLinksInSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><a href="/other/page">Other</a></body></html>`))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestNewWithClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/1/code-001">CODE-001</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`/api/v1/videos/1/manifest/master.m3u8`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
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

func TestSearch_MatchesCodeInLinkText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/42/generic-slug">TEXT-001 - Some Movie Title</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`/api/v1/videos/42/manifest/master.m3u8`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "TEXT-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, srv.URL+"/cn/video/42/text-001", results[0].PageURL)
}

func TestSearch_DeduplicatesM3U8Sources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/10/dedup-001">DEDUP-001</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
				<script>/api/v1/videos/10/manifest/master.m3u8</script>
				<script>/api/v1/videos/10/manifest/master.m3u8</script>
				<script>/api/v1/videos/10/manifest/master.m3u8</script>
			`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "DEDUP-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].VideoSources, 1)
}

func TestSearch_MatchesByNormalizedCodeInHref(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cn/search" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><a href="/cn/video/33/abc_123">Click Here</a></html>`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/cn/video/") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`/api/v1/videos/33/manifest/master.m3u8`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}
