package missav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pageOK = `<!DOCTYPE html>
<html>
<head><title>ABC-123 Sample Video</title>
<meta property="og:title" content="ABC-123">
</head>
<body>
<h1>ABC-123</h1>
<video controls>
	<source src="https://cdn.example.com/video.mp4" type="video/mp4">
	<source src="https://cdn.example.com/video_hd.mp4" type="video/mp4">
</video>
<iframe src="https://embed.example.com/abc123"></iframe>
</body>
</html>`

const pageWithSubtitle = `<!DOCTYPE html>
<html>
<head><title>ABC-123 Subbed</title></head>
<body>
<h1>ABC-123</h1>
<video>
	<source src="https://cdn.example.com/video.mp4" type="video/mp4">
</video>
<div class="space-y-2">
	<a class="text-nord13" href="/cn/abc-123/chinese-subtitle">Chinese Subtitle</a>
</div>
</body>
</html>`

const pageWithLeak = `<!DOCTYPE html>
<html>
<head><title>ABC-123 Leaked</title></head>
<body>
<h1>ABC-123</h1>
<video src="https://cdn.example.com/video.mp4"></video>
<div class="order-first">
	<div class="rounded-md">
		<a href="/leak/abc-123">Leak Version</a>
	</div>
</div>
</body>
</html>`

const pageWithBoth = `<!DOCTYPE html>
<html>
<head><title>ABC-123 Complete</title></head>
<body>
<h1>ABC-123</h1>
<video>
	<source src="https://cdn.example.com/video.mp4" type="video/mp4">
</video>
<div class="space-y-2">
	<a class="text-nord13" href="/cn/abc-123/chinese-subtitle">Chinese Subtitle</a>
</div>
<div class="order-first">
	<div class="rounded-md">
		<a href="/leak/abc-123">Leak Version</a>
	</div>
</div>
</body>
</html>`

const pageNotFound = `<!DOCTYPE html>
<html>
<head><title>404 Not Found</title></head>
<body>
<h1>Page not found</h1>
<p>The page you are looking for does not exist.</p>
</body>
</html>`

const pageNoTitleMatch = `<!DOCTYPE html>
<html>
<head><title>Unknown Video</title></head>
<body>
<h1>Some Other Content</h1>
<video>
	<source src="https://cdn.example.com/video.mp4" type="video/mp4">
</video>
</body>
</html>`

const pageEmpty = `<!DOCTYPE html>
<html>
<head><title>Unknown Page</title></head>
<body><p>Nothing here.</p></body>
</html>`

const pageVideoWithSrcAttr = `<!DOCTYPE html>
<html>
<head><title>XYZ-999</title></head>
<body>
<h1>XYZ-999</h1>
<video src="https://cdn.example.com/direct.mp4" controls></video>
</body>
</html>`

const pageGarbage = `<!DOCTYPE html>
<html>
<head><title>ABC-123 Video</title></head>
<body>
<h1>ABC-123</h1>
<p>大量垃圾内容 on this page</p>
<video src="https://cdn.example.com/video.mp4"></video>
</body>
</html>`

const pageWithNonVideoIframe = `<!DOCTYPE html>
<html>
<head><title>XYZ-999</title></head>
<body>
<h1>XYZ-999</h1>
<video src="https://cdn.example.com/video.mp4"></video>
<iframe src="https://ads.tracker.com/adframe"></iframe>
<iframe src="https://social.example.com/share"></iframe>
</body>
</html>`

func newTestScraper(srvURL string) *MISSAVScraper {
	return &MISSAVScraper{
		enabled:     true,
		proxyConfig: domain.ProxyConfig{},
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		baseURL:     srvURL,
		cfTested:    true,
		cfPassed:    true,
	}
}

func TestName(t *testing.T) {
	s := newTestScraper("http://localhost")
	assert.Equal(t, "missav", s.Name())
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
	s := &MISSAVScraper{
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
		assert.Equal(t, "/ABC-123/", r.URL.Path)
		assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageOK))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, "missav", r.SiteName)
	assert.Equal(t, domain.StatusSuccess, r.Status)
	assert.Equal(t, domain.VersionOriginal, r.Version)
	assert.Equal(t, srv.URL+"/ABC-123/", r.PageURL)
	assert.False(t, r.Subtitle)
	assert.False(t, r.Leak)

	require.Len(t, r.VideoSources, 3)
	assert.Equal(t, "https://cdn.example.com/video.mp4", r.VideoSources[0].URL)
	assert.Equal(t, "video/mp4", r.VideoSources[0].Type)
	assert.Equal(t, "https://cdn.example.com/video_hd.mp4", r.VideoSources[1].URL)
	assert.Equal(t, "https://embed.example.com/abc123", r.VideoSources[2].URL)
	assert.Equal(t, "text/html", r.VideoSources[2].Type)
}

func TestSearchWithSubtitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageWithSubtitle))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	assert.False(t, results[0].Subtitle)

	assert.Equal(t, domain.VersionCNSub, results[1].Version)
	assert.True(t, results[1].Subtitle)
	assert.False(t, results[1].Leak)
	assert.Len(t, results[1].VideoSources, 1)
}

func TestSearchWithLeak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageWithLeak))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	assert.False(t, results[0].Leak)

	assert.Equal(t, domain.VersionMosaicReduce, results[1].Version)
	assert.True(t, results[1].Leak)
	assert.False(t, results[1].Subtitle)
	assert.Len(t, results[1].VideoSources, 1)
}

func TestSearchWithBothVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageWithBoth))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	assert.Equal(t, domain.VersionMosaicReduce, results[1].Version)
	assert.True(t, results[1].Leak)
}

func TestSearch404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "NONEXISTENT")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchNotFoundInBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageNotFound))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchNotFoundInTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><head><title>Not Found - Site</title></head><body></body></html>`))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestSearchDisabled(t *testing.T) {
	s := &MISSAVScraper{enabled: false}
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestSearchNoTitleMatchButHasSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageNoTitleMatch))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Empty(t, results[0].Error)
	assert.NotEmpty(t, results[0].VideoSources)
}

func TestSearchNoTitleMatchNoSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageEmpty))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, domain.StatusNotFound, results[0].Status)
	assert.Contains(t, results[0].Error, "title does not match")
}

func TestSearchVideoSrcAttr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageVideoWithSrcAttr))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "XYZ-999")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	require.Len(t, results[0].VideoSources, 1)
	assert.Equal(t, "https://cdn.example.com/direct.mp4", results[0].VideoSources[0].URL)
}

func TestSearchDeduplicatesSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>DUP-001</title></head>
<body><h1>DUP-001</h1>
<video src="https://cdn.example.com/same.mp4"></video>
<video><source src="https://cdn.example.com/same.mp4" type="video/mp4"></video>
</body></html>`))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "DUP-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].VideoSources, 1, "duplicate sources should be deduplicated")
}

func TestSearchOGTitleMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Generic</title><meta property="og:title" content="ABC-123 Video"></head>
<body><video src="https://cdn.example.com/video.mp4"></video></body></html>`))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestSearchH1Match(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Generic</title></head>
<body><h1>Watch ABC-123 Online</h1>
<video src="https://cdn.example.com/video.mp4"></video></body></html>`))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestVerifyTitleUnit(t *testing.T) {
	t.Run("title tag match", func(t *testing.T) {
		doc := parseHTML(t, `<html><head><title>ABC-123 Video</title></head><body></body></html>`)
		assert.True(t, verifyTitle(doc, "abc123"))
	})

	t.Run("og:title match", func(t *testing.T) {
		doc := parseHTML(t, `<html><head><title>Generic</title><meta property="og:title" content="ABC-123"></head><body></body></html>`)
		assert.True(t, verifyTitle(doc, "abc123"))
	})

	t.Run("h1 match", func(t *testing.T) {
		doc := parseHTML(t, `<html><head><title>Generic</title></head><body><h1>ABC-123 HD</h1></body></html>`)
		assert.True(t, verifyTitle(doc, "abc123"))
	})

	t.Run("no match", func(t *testing.T) {
		doc := parseHTML(t, `<html><head><title>Something Else</title></head><body><h1>Other</h1></body></html>`)
		assert.False(t, verifyTitle(doc, "abc123"))
	})

	t.Run("case insensitive match", func(t *testing.T) {
		doc := parseHTML(t, `<html><head><title>abc-123 video</title></head><body></body></html>`)
		assert.True(t, verifyTitle(doc, "abc123"))
	})
}

func TestSearchGarbage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageGarbage))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchFilterNonVideoIframe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageWithNonVideoIframe))
	}))
	defer srv.Close()

	s := newTestScraper(srv.URL)
	results, err := s.Search(context.Background(), "XYZ-999")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	require.Len(t, results[0].VideoSources, 1)
	assert.Equal(t, "https://cdn.example.com/video.mp4", results[0].VideoSources[0].URL)
}

func TestIsVideoDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://missav.ws/embed/abc", true},
		{"https://asg.to/player/abc", true},
		{"https://avmeet.com/video/abc", true},
		{"https://player.example.com/video.m3u8", true},
		{"https://cdn.stream.com/video.mp4", true},
		{"https://ads.tracker.com/ad", false},
		{"https://social.example.com/share", false},
		{"https://google.com/", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, isVideoDomain(tt.url))
		})
	}
}

func parseHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)
	return doc
}
