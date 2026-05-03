package javgg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const searchPageOK = `<!DOCTYPE html>
<html>
<head><title>Search Results</title></head>
<body>
<article>
	<div class="details">
		<h3 class="title"><a href="https://javgg.net/jav/abc-123/">ABC-123 Sample Video</a></h3>
	</div>
</article>
</body>
</html>`

const searchPageMultiResults = `<!DOCTYPE html>
<html>
<head><title>Search Results</title></head>
<body>
<article>
	<div class="details">
		<h3 class="title"><a href="/jav/xyz-000/">XYZ-000 Some Other Video</a></h3>
	</div>
</article>
<article>
	<div class="details">
		<h3 class="title"><a href="https://javgg.net/jav/abc-123/">ABC-123 Sample Video</a></h3>
	</div>
</article>
<article>
	<div class="details">
		<h3 class="title"><a href="/jav/def-456/">DEF-456 Another Video</a></h3>
	</div>
</article>
</body>
</html>`

const searchPageCaseInsensitive = `<!DOCTYPE html>
<html>
<head><title>Search Results</title></head>
<body>
<article>
	<div class="details">
		<h3 class="title"><a href="https://javgg.net/jav/abc-123/">abc-123 sample video</a></h3>
	</div>
</article>
</body>
</html>`

const searchPageNoResults = `<!DOCTYPE html>
<html>
<head><title>Search Results</title></head>
<body>
<p>No results found for your query.</p>
</body>
</html>`

const searchPageNoMatchingLink = `<!DOCTYPE html>
<html>
<head><title>Search Results</title></head>
<body>
<article>
	<div class="details">
		<h3 class="title"><a href="/jav/def-456/">DEF-456 Video</a></h3>
	</div>
</article>
</body>
</html>`

const videoPageOK = `<!DOCTYPE html>
<html>
<head><title>ABC-123 - Watch Online</title></head>
<body>
<video controls>
	<source src="https://cdn.example.com/video.mp4" type="video/mp4">
	<source src="https://cdn.example.com/video_720p.mp4" type="video/mp4">
</video>
<iframe src="https://embed.example.com/abc123"></iframe>
</body>
</html>`

const videoPageSingleSource = `<!DOCTYPE html>
<html>
<head><title>ABC-123</title></head>
<body>
<video src="https://cdn.example.com/direct.mp4" controls></video>
</body>
</html>`

const videoPageDedup = `<!DOCTYPE html>
<html>
<head><title>ABC-123</title></head>
<body>
<video src="https://cdn.example.com/same.mp4"></video>
<video><source src="https://cdn.example.com/same.mp4" type="video/mp4"></video>
</body>
</html>`

func newTestScraper(srvURL string) *Scraper {
	return &Scraper{
		enabled:     true,
		proxyConfig: domain.ProxyConfig{},
		httpClient:  srvURLClient(srvURL),
		baseURL:     srvURL,
	}
}

func srvURLClient(srvURL string) *http.Client {
	return &http.Client{
		Transport: &roundTripper{base: srvURL},
	}
}

type roundTripper struct {
	base string
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	u.Scheme = "http"
	if rt.base != "" {
		u.Host = strings.TrimPrefix(strings.TrimPrefix(rt.base, "http://"), "https://")
	}
	req.URL = &u
	return http.DefaultTransport.RoundTrip(req)
}

func TestName(t *testing.T) {
	s := newTestScraper("http://localhost")
	assert.Equal(t, "javgg", s.Name())
}

func TestIsEnabled(t *testing.T) {
	s := newTestScraper("http://localhost")
	assert.True(t, s.IsEnabled())

	s.enabled = false
	assert.False(t, s.IsEnabled())
}

func TestRequiresCFBypass(t *testing.T) {
	s := newTestScraper("http://localhost")
	assert.True(t, s.RequiresCFBypass())
}

func TestGetProxyConfig(t *testing.T) {
	cfg := domain.ProxyConfig{
		URL:     "http://proxy:8080",
		Enabled: true,
	}
	s := &Scraper{
		enabled:     true,
		proxyConfig: cfg,
		httpClient:  &http.Client{},
	}
	assert.Equal(t, cfg, s.GetProxyConfig())
}

func TestFormatCode(t *testing.T) {
	s := newTestScraper("http://localhost")
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC-123", "ABC-123"},
		{"abc123", "abc123"},
		{"MIDE-999", "MIDE-999"},
		{"SSIS-001", "SSIS-001"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, s.FormatCode(tt.input))
		})
	}
}

func TestSearchSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			assert.Equal(t, "1", r.Header.Get("Upgrade-Insecure-Requests"))
			assert.NotEmpty(t, r.Header.Get("Referer"))
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(videoPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, "javgg", r.SiteName)
	assert.Equal(t, domain.StatusSuccess, r.Status)
	assert.Equal(t, domain.VersionOriginal, r.Version)
	assert.Contains(t, r.PageURL, "/jav/abc-123/")
	assert.False(t, r.Subtitle)
	assert.False(t, r.Leak)
	require.Len(t, r.VideoSources, 3)
	assert.Equal(t, "https://cdn.example.com/video.mp4", r.VideoSources[0].URL)
	assert.Equal(t, "video/mp4", r.VideoSources[0].Type)
	assert.Equal(t, "https://cdn.example.com/video_720p.mp4", r.VideoSources[1].URL)
	assert.Equal(t, "https://embed.example.com/abc123", r.VideoSources[2].URL)
	assert.Equal(t, "text/html", r.VideoSources[2].Type)
}

func TestSearchMultiResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageMultiResults)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(videoPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].PageURL, "/jav/abc-123/")
}

func TestSearchCaseInsensitive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageCaseInsensitive)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(videoPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "abc-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestSearchNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(searchPageNoResults)); err != nil {
			t.Errorf("write failed: %v", err)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "NONEXISTENT")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchNotFoundNoMatchingLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(searchPageNoMatchingLink)); err != nil {
			t.Errorf("write failed: %v", err)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchHTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestSearchHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("Internal Server Error")); err != nil {
			t.Errorf("write failed: %v", err)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestSearchDisabled(t *testing.T) {
	s := &Scraper{enabled: false}
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestSearchCFBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<html><body>Just a moment... Checking your browser before accessing javgg.net. Cloudflare challenge-platform</body></html>`)); err != nil {
			t.Errorf("write failed: %v", err)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchCustomHeaders(t *testing.T) {
	var capturedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(videoPageSingleSource)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	_, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)

	assert.Equal(t, "1", capturedHeaders.Get("Upgrade-Insecure-Requests"))
	assert.Contains(t, capturedHeaders.Get("Referer"), "javgg.net")
	assert.NotEmpty(t, capturedHeaders.Get("User-Agent"))
}

func TestSearchVideoPageHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte("Server Error")); err != nil {
				t.Errorf("write failed: %v", err)
			}
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
	assert.Contains(t, results[0].Error, "video page")
}

func TestSearchVideoPage404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
}

func TestSearchVideoPageNoSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`<html><body><p>No player found</p></body></html>`)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchDeduplicatesSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path + "?" + r.URL.RawQuery
		if strings.Contains(path, "?s=") {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(searchPageOK)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(videoPageDedup)); err != nil {
				t.Errorf("write failed: %v", err)
			}
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].VideoSources, 1)
}

func TestFindVideoLinkUnit(t *testing.T) {
	t.Run("matches href", func(t *testing.T) {
		doc := parseHTML(t, searchPageOK)
		link := findVideoLink(doc, "abc123")
		assert.Equal(t, "https://javgg.net/jav/abc-123/", link)
	})

	t.Run("matches text", func(t *testing.T) {
		doc := parseHTML(t, `<article><div class="details"><h3 class="title"><a href="/jav/some-id/">ABC-123 Video</a></h3></div></article>`)
		link := findVideoLink(doc, "abc123")
		assert.Equal(t, "/jav/some-id/", link)
	})

	t.Run("case insensitive", func(t *testing.T) {
		doc := parseHTML(t, `<article><div class="details"><h3 class="title"><a href="/jav/AbC-123/">AbC-123</a></h3></div></article>`)
		link := findVideoLink(doc, "abc123")
		assert.Equal(t, "/jav/AbC-123/", link)
	})

	t.Run("no match", func(t *testing.T) {
		doc := parseHTML(t, `<article><div class="details"><h3 class="title"><a href="/jav/xyz-000/">XYZ-000</a></h3></div></article>`)
		link := findVideoLink(doc, "abc123")
		assert.Empty(t, link)
	})

	t.Run("empty document", func(t *testing.T) {
		doc := parseHTML(t, `<html><body></body></html>`)
		link := findVideoLink(doc, "abc123")
		assert.Empty(t, link)
	})

	t.Run("fallback without article", func(t *testing.T) {
		doc := parseHTML(t, `<html><body><a href="/jav/abc-123/">ABC-123</a></body></html>`)
		link := findVideoLink(doc, "abc123")
		assert.Equal(t, "/jav/abc-123/", link)
	})
}

func TestExtractVideoSourcesUnit(t *testing.T) {
	t.Run("video source elements", func(t *testing.T) {
		doc := parseHTML(t, `<video><source src="https://cdn.example.com/vid.mp4" type="video/mp4"></video>`)
		sources := extractVideoSources(doc)
		require.Len(t, sources, 1)
		assert.Equal(t, "https://cdn.example.com/vid.mp4", sources[0].URL)
		assert.Equal(t, "video/mp4", sources[0].Type)
	})

	t.Run("video src attribute", func(t *testing.T) {
		doc := parseHTML(t, `<video src="https://cdn.example.com/vid.mp4"></video>`)
		sources := extractVideoSources(doc)
		require.Len(t, sources, 1)
		assert.Equal(t, "https://cdn.example.com/vid.mp4", sources[0].URL)
		assert.Equal(t, "video/mp4", sources[0].Type)
	})

	t.Run("iframe src attribute", func(t *testing.T) {
		doc := parseHTML(t, `<iframe src="https://embed.example.com/player"></iframe>`)
		sources := extractVideoSources(doc)
		require.Len(t, sources, 1)
		assert.Equal(t, "https://embed.example.com/player", sources[0].URL)
		assert.Equal(t, "text/html", sources[0].Type)
	})

	t.Run("deduplicates", func(t *testing.T) {
		doc := parseHTML(t, `<video src="https://cdn.example.com/vid.mp4"></video><video><source src="https://cdn.example.com/vid.mp4" type="video/mp4"></video>`)
		sources := extractVideoSources(doc)
		assert.Len(t, sources, 1)
	})

	t.Run("no sources", func(t *testing.T) {
		doc := parseHTML(t, `<html><body><p>No video</p></body></html>`)
		sources := extractVideoSources(doc)
		assert.Empty(t, sources)
	})
}

func parseHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)
	return doc
}
