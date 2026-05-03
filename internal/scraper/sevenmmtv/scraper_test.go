package sevenmmtv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// HTML fixture helpers
// ---------------------------------------------------------------------------

func wrapHTML(head, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>%s</head>
<body>%s</body>
</html>`, head, body)
}

func testServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

const (
	pageSearchResults = `<!DOCTYPE html>
<html>
<head><title>搜索结果 - ABC-123</title></head>
<body>
<div class="video-list">
	<div class="video-item">
		<a href="/zh/video/abc123.html" target="_top">
			<img src="/thumb/abc123.jpg" alt="ABC-123 独占配信" />
			<span class="title">ABC-123</span>
		</a>
	</div>
	<div class="video-item">
		<a href="/zh/video/abc123.html" target="_top">
			<img src="/thumb/abc123_2.jpg" alt="ABC-123 HD" />
		</a>
	</div>
	<a href="/zh/searchall_search/abc123.html" target="_top">
		Search All Results
	</a>
	<div class="video-item">
		<a href="/zh/video/xyz999.html" target="_top">
			<img src="/thumb/xyz999.jpg" alt="XYZ-999" />
		</a>
	</div>
	<a href="/external/page.html" target="_blank">External Link</a>
</div>
</body>
</html>`

	pageDetailWithIframe = `<!DOCTYPE html>
<html>
<head><title>ABC-123 独占配信 - 7mmtv</title></head>
<body>
<h1>ABC-123 独占配信</h1>
<iframe src="https://player.example.com/embed/abc123" width="800" height="600"></iframe>
</body>
</html>`

	pageDetailWithVideo = `<!DOCTYPE html>
<html>
<head><title>ABC-123 Video</title></head>
<body>
<h1>ABC-123</h1>
<video controls>
	<source src="https://cdn.example.com/720p.mp4" type="video/mp4">
	<source src="https://cdn.example.com/1080p.mp4" type="video/mp4">
</video>
</body>
</html>`

	pageDetailWithDirectVideo = `<!DOCTYPE html>
<html>
<head><title>ABC-123 Direct</title></head>
<body>
<h1>ABC-123</h1>
<video src="https://cdn.example.com/direct.mp4" controls></video>
</body>
</html>`

	pageDetailNoSources = `<!DOCTYPE html>
<html>
<head><title>ABC-123 No Sources</title></head>
<body>
<h1>ABC-123</h1>
<p>No video available.</p>
</body>
</html>`

	pageSearchEmpty = `<!DOCTYPE html>
<html>
<head><title>搜索结果</title></head>
<body>
<p>No results found.</p>
</body>
</html>`

	pageSearchExcludedOnly = `<!DOCTYPE html>
<html>
<head><title>搜索结果</title></head>
<body>
<a href="/zh/searchall_search/abc123.html" target="_top">
	<img src="/thumb/abc123.jpg" alt="ABC-123" />
</a>
</body>
</html>`

	pageSearchNoImgAlt = `<!DOCTYPE html>
<html>
<head><title>搜索结果</title></head>
<body>
<div class="video-item">
	<a href="/zh/video/abc123.html" target="_top">
		<!-- No img element -->
		<span class="title">ABC-123</span>
	</a>
</div>
</body>
</html>`

	pageSearchAltMismatch = `<!DOCTYPE html>
<html>
<head><title>搜索结果</title></head>
<body>
<div class="video-item">
	<a href="/zh/video/abc123.html" target="_top">
		<img src="/thumb/wrong.jpg" alt="XYZ-999 Different Video" />
	</a>
</div>
</body>
</html>`

	pageSearchRelativeHref = `<!DOCTYPE html>
<html>
<head><title>搜索结果</title></head>
<body>
<div class="video-item">
	<a href="video/abc123.html" target="_top">
		<img src="/thumb/abc123.jpg" alt="ABC-123" />
	</a>
</div>
</body>
</html>`

	pageCFBlocked = `<!DOCTYPE html>
<html>
<head><title>Just a moment...</title></head>
<body>
<p>Checking your browser before accessing 7mmtv.sx.</p>
<div class="cf-browser-verification">Please wait...</div>
</body>
</html>`
)

// ---------------------------------------------------------------------------
// Interface contract tests
// ---------------------------------------------------------------------------

func TestName(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchResults))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	assert.Equal(t, "7mmtv", s.Name())
}

func TestFormatCode_Unchanged(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchResults))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	tests := []struct {
		input string
	}{
		{"ABC-123"},
		{"abc123"},
		{"SSIS-001"},
		{"MIDE-999"},
		{"IPX-888"},
		{""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.input, s.FormatCode(tt.input))
		})
	}
}

func TestIsEnabled(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchResults))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	assert.True(t, s.IsEnabled())

	s.enabled = false
	assert.False(t, s.IsEnabled())
}

func TestRequiresCFBypass(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchResults))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	assert.True(t, s.RequiresCFBypass())
}

func TestGetProxyConfig(t *testing.T) {
	cfg := domain.ProxyConfig{
		URL:     "http://proxy:8080",
		Enabled: true,
	}
	s := &Scraper{
		client:      &http.Client{},
		baseURL:     baseURL,
		enabled:     true,
		proxyConfig: cfg,
	}
	assert.Equal(t, cfg, s.GetProxyConfig())
}

func TestImplementsScraperInterface(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchResults))
	})

	var s domain.Scraper = newTestScraper(srv.Client(), srv.URL)
	assert.NotNil(t, s)
	assert.Equal(t, "7mmtv", s.Name())
}

func TestInitRegistration(t *testing.T) {
	all := scraper.GetAll()
	found := false
	for _, s := range all {
		if s.Name() == siteName {
			found = true
			break
		}
	}
	assert.True(t, found, "7mmtv scraper should be registered via init()")

	_ = &sync.RWMutex{} // ensure sync is imported
}

// ---------------------------------------------------------------------------
// Search — two-phase flow tests
// ---------------------------------------------------------------------------

func TestSearch_Success_Iframe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/searchform_search/"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/zh/video/abc123.html"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageDetailWithIframe))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, "7mmtv", r.SiteName)
	assert.Equal(t, domain.StatusSuccess, r.Status)
	assert.Equal(t, domain.VersionOriginal, r.Version)
	assert.Contains(t, r.PageURL, "/zh/video/abc123.html")
	require.Len(t, r.VideoSources, 1)
	assert.Contains(t, r.VideoSources[0].URL, "player.example.com")
	assert.Equal(t, "text/html", r.VideoSources[0].Type)
}

func TestSearch_Success_VideoSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/zh/video/abc123.html"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageDetailWithVideo))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, domain.StatusSuccess, r.Status)
	require.Len(t, r.VideoSources, 2)
	assert.Contains(t, r.VideoSources[0].URL, "720p.mp4")
	assert.Equal(t, "video/mp4", r.VideoSources[0].Type)
	assert.Contains(t, r.VideoSources[1].URL, "1080p.mp4")
	assert.Equal(t, "video/mp4", r.VideoSources[1].Type)
}

func TestSearch_Success_DirectVideoSrc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/zh/video/abc123.html"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageDetailWithDirectVideo))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, domain.StatusSuccess, r.Status)
	require.Len(t, r.VideoSources, 1)
	assert.Contains(t, r.VideoSources[0].URL, "direct.mp4")
}

func TestSearch_NotFound_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchEmpty))
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "NONEXISTENT")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearch_NotFound_DetailPage404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearch_NotFound_NoVideoSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageDetailNoSources))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
	assert.Contains(t, results[0].Error, "no video sources")
}

func TestSearch_CloudflareBlocked_SearchPage(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageCFBlocked))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusBlocked, results[0].Status)
	assert.Contains(t, results[0].Error, "Cloudflare challenge")
}

func TestSearch_CloudflareBlocked_DetailPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageCFBlocked))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusBlocked, results[0].Status)
	assert.Contains(t, results[0].Error, "detail page")
}

func TestSearch_HTTPError_SearchPage(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	s := newTestScraper(srv.Client(), srv.URL)
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search HTTP 500")
}

func TestSearch_HTTPError_DetailPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detail HTTP 500")
}

func TestSearch_Disabled(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should not be called"))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	s.enabled = false
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestSearch_ContextCanceled(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchResults))
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := newTestScraper(srv.Client(), srv.URL)
	_, err := s.Search(ctx, "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "7mmtv")
}

// ---------------------------------------------------------------------------
// POST method verification
// ---------------------------------------------------------------------------

func TestSearch_POSTMethodAndFormBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/zh/searchform_search/all/index.html", r.URL.Path)

			ct := r.Header.Get("Content-Type")
			assert.Contains(t, ct, "application/x-www-form-urlencoded")

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			bodyStr := string(body)

			assert.Contains(t, bodyStr, "search_keyword=ABC-123")
			assert.Contains(t, bodyStr, "search_type=searchall")
			assert.Contains(t, bodyStr, "op=search")

			values, err := url.ParseQuery(bodyStr)
			require.NoError(t, err)
			assert.Equal(t, "ABC-123", values.Get("search_keyword"))
			assert.Equal(t, "searchall", values.Get("search_type"))
			assert.Equal(t, "search", values.Get("op"))

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
			return
		}

		// Detail page GET
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageDetailWithIframe))
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, srv.URL+"/zh/video/abc123.html", results[0].PageURL)
}

func TestSearch_POSTHeaders(t *testing.T) {
	var baseURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baseURL = "http://" + r.Host
		if r.Method == http.MethodPost {
			assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
			assert.Contains(t, r.Header.Get("Origin"), baseURL)
			assert.Contains(t, r.Header.Get("Referer"), "/zh/")
			assert.Contains(t, r.Header.Get("Content-Type"), "UTF-8")

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageDetailWithIframe))
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	_, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Search result parsing — unit tests
// ---------------------------------------------------------------------------

func TestFindMatchingResult_DeduplicatesByHref(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageSearchResults))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageDetailWithIframe))
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	// Should pick abc123.html (not a duplicate because it resolves the same but seen map handles it)
	assert.Contains(t, results[0].PageURL, "/zh/video/abc123.html")
}

func TestFindMatchingResult_ExcludesSearchLinks(t *testing.T) {
	// Only search links — should return not found
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageSearchExcludedOnly))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestFindMatchingResult_MatchByHrefSlug(t *testing.T) {
	// Case-insensitive slug match
	html := wrapHTML("<title>Search</title>", `
		<a href="/zh/video/ABC-123.html" target="_top">
			<img src="/thumb/abc.jpg" alt="ABC-123" />
		</a>
	`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageDetailWithIframe))
	}))
	defer srv.Close()

	// Search with lowercase, href has uppercase
	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "abc123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestFindMatchingResult_MatchByImgAlt(t *testing.T) {
	// slug matches, alt also matches
	html := wrapHTML("<title>Search</title>", `
		<a href="/zh/video/abc123.html" target="_top">
			<img src="/thumb/abc.jpg" alt="ABC-123 Exclusive Release" />
		</a>
	`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageDetailWithIframe))
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestFindMatchingResult_NoImgAlt_StillMatches(t *testing.T) {
	// slug matches, but no img element — should still match
	html := pageSearchNoImgAlt
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageDetailWithIframe))
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestFindMatchingResult_AltMismatch_Skip(t *testing.T) {
	// slug matches but alt text is different — should skip
	html := pageSearchAltMismatch
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestFindMatchingResult_NonMatchingSlug(t *testing.T) {
	html := wrapHTML("<title>Search</title>", `
		<a href="/zh/video/xyz999.html" target="_top">
			<img src="/thumb/xyz.jpg" alt="XYZ-999" />
		</a>
	`)
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestFindMatchingResult_ExcludesNonTargetLinks(t *testing.T) {
	// Links without target="_top" should be excluded by the selector
	html := wrapHTML("<title>Search</title>", `
		<a href="/zh/video/abc123.html" target="_blank">
			<img src="/thumb/abc.jpg" alt="ABC-123" />
		</a>
	`)
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestFindMatchingResult_ExcludesNonHtmlLinks(t *testing.T) {
	// Links not ending in .html should be excluded
	html := wrapHTML("<title>Search</title>", `
		<a href="/zh/video/abc123.mp4" target="_top">
			<img src="/thumb/abc.jpg" alt="ABC-123" />
		</a>
	`)
	srv := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestFindMatchingResult_RelativeHref(t *testing.T) {
	// Relative href should be resolved against base URL
	html := pageSearchRelativeHref
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		if strings.Contains(r.URL.Path, "video/abc123.html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pageDetailWithIframe))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}



func TestFindMatchingResult_CodeWithHyphens(t *testing.T) {
	// Input "ABC-123" → normalized "abc123" → matches slug "abc123"
	html := wrapHTML("<title>Search</title>", `
		<a href="/zh/video/abc123.html" target="_top">
			<img src="/thumb/abc.jpg" alt="ABC-123" />
		</a>
	`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(html))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageDetailWithIframe))
	}))
	defer srv.Close()

	s := newTestScraper(srv.Client(), srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

// ---------------------------------------------------------------------------
// Helper unit tests
// ---------------------------------------------------------------------------

func TestExtractSlug(t *testing.T) {
	tests := []struct {
		href     string
		expected string
	}{
		{"/zh/video/abc-123.html", "abc-123"},
		{"/zh/video/mide-999.html", "mide-999"},
		{"/video/ssis-001.html", "ssis-001"},
		{"/abc-123.html?q=1", "abc-123"},
		{"abc-123.html", "abc-123"},
		{"/zh/video/ABC-123.html", "ABC-123"},
		{"/just/slug.html", "slug"},
		{"/no-extension", "no-extension"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			got := extractSlug(tt.href)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveURL(t *testing.T) {
	base := "https://7mmtv.sx"
	tests := []struct {
		href     string
		expected string
	}{
		{"https://other.example.com/path", "https://other.example.com/path"},
		{"//cdn.example.com/video.mp4", "//cdn.example.com/video.mp4"},
		{"/zh/video/abc123.html", "https://7mmtv.sx/zh/video/abc123.html"},
		{"video/abc123.html", "https://7mmtv.sx/video/abc123.html"},
		{"abc123.html", "https://7mmtv.sx/abc123.html"},
	}
	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			got := resolveURL(base, tt.href)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveURL_Empty(t *testing.T) {
	// url.ResolveReference drops trailing slash on base URLs
	assert.Equal(t, "https://base.example.com", resolveURL("https://base.example.com", ""))
}

func TestIsCFBlocked(t *testing.T) {
	tests := []struct {
		body     string
		expected bool
	}{
		{"<html><body>Just a moment...</body></html>", true},
		{"<p>Checking your browser before accessing</p>", true},
		{"<div class='cf-browser-verification'></div>", true},
		{"<script>challenge-platform</script>", true},
		{"<h1>Attention Required! | Cloudflare</h1>", true},
		{"<html><head><title>Video Page</title></head><body>Content</body></html>", false},
		{"", false},
		{"Normal page content with no CF markers.", false},
	}
	for _, tt := range tests {
		t.Run(tt.body[:min(len(tt.body), 30)], func(t *testing.T) {
			got := isCFBlocked(tt.body)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetectVideoType(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://cdn.example.com/video.m3u8", "application/x-mpegURL"},
		{"https://cdn.example.com/stream/master.m3u8?token=abc", "application/x-mpegURL"},
		{"https://cdn.example.com/video.mp4", "video/mp4"},
		{"https://cdn.example.com/video.webm", "video/webm"},
		{"https://cdn.example.com/stream.ts", "video/mp2t"},
		{"https://cdn.example.com/unknown", "video/mp4"},
		{"", "video/mp4"},
	}
	for _, tt := range tests {
		t.Run(tt.url[:min(len(tt.url), 40)], func(t *testing.T) {
			got := detectVideoType(tt.url)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func parseHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)
	return doc
}

func TestExtractVideoSources_Deduplicates(t *testing.T) {
	// Same URL appearing in iframe and video should only be captured once
	html := wrapHTML("<title>Test</title>", `
		<iframe src="https://player.example.com/embed/dup"></iframe>
		<video src="https://player.example.com/embed/dup"></video>
		<source src="https://player.example.com/embed/dup">
	`)
	doc := parseHTML(t, html)

	sources := extractVideoSources(doc)
	assert.Len(t, sources, 1, "duplicate URLs should be deduplicated")
	assert.Equal(t, "https://player.example.com/embed/dup", sources[0].URL)
}

func TestExtractVideoSources_AllTypes(t *testing.T) {
	html := wrapHTML("<title>All Sources</title>", `
		<iframe src="https://embed.example.com/player"></iframe>
		<video src="https://cdn.example.com/direct.mp4"></video>
		<video>
			<source src="https://cdn.example.com/stream.m3u8" type="application/x-mpegURL">
		</video>
		<source src="https://cdn.example.com/standalone.mp4" type="video/mp4">
	`)
	doc := parseHTML(t, html)

	sources := extractVideoSources(doc)
	assert.Len(t, sources, 4, "should extract all unique sources")

	types := make(map[string]bool)
	for _, s := range sources {
		types[s.Type] = true
	}
	assert.True(t, types["text/html"], "should have iframe type")
	assert.True(t, types["video/mp4"], "should have mp4 type")
	assert.True(t, types["application/x-mpegURL"], "should have m3u8 type")
}

func TestExtractVideoSources_Empty(t *testing.T) {
	html := wrapHTML("<title>Empty</title>", `<p>No videos here.</p>`)
	doc := parseHTML(t, html)

	sources := extractVideoSources(doc)
	assert.Empty(t, sources)
}
