package hayav

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatCode(t *testing.T) {
	s := New(domain.ProxyConfig{})
	tests := []struct {
		input string
	}{
		{"ABC-123"},
		{"abc123"},
		{"ABC_123"},
		{"SSIS-001"},
		{"ipx-999"},
		{"anything"},
		{""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := s.FormatCode(tt.input)
			assert.Equal(t, tt.input, got, "FormatCode should return code unchanged")
		})
	}
}

func TestName(t *testing.T) {
	s := New(domain.ProxyConfig{})
	assert.Equal(t, Name, s.Name())
	assert.Equal(t, "hayav", s.Name())
}

func TestIsEnabled(t *testing.T) {
	s := New(domain.ProxyConfig{})
	assert.True(t, s.IsEnabled())
}

func TestRequiresCFBypass(t *testing.T) {
	s := New(domain.ProxyConfig{})
	assert.False(t, s.RequiresCFBypass())
}

func TestGetProxyConfig(t *testing.T) {
	s := New(domain.ProxyConfig{})
	cfg := s.GetProxyConfig()
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.URL)
}

func TestImplementsScraperInterface(t *testing.T) {
	var s domain.Scraper = New(domain.ProxyConfig{})
	assert.NotNil(t, s)
	assert.Equal(t, "hayav", s.Name())
}

func TestInitRegistration(t *testing.T) {
	// Clear registry for isolation
	scraperMu := &sync.RWMutex{}
	_ = scraperMu

	all := scraper.GetAll()
	found := false
	for _, s := range all {
		if s.Name() == Name {
			found = true
			break
		}
	}
	assert.True(t, found, "hayav scraper should be registered via init()")
}

// ---------------------------------------------------------------------------
// Search tests with mocked HTML pages
// ---------------------------------------------------------------------------

func newTestScraper(srv *httptest.Server) *Scraper {
	return NewWithClient(srv.Client(), srv.URL)
}

func htmlPage(head, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
%s
</head>
<body>
%s
</body>
</html>`, head, body)
}

func TestSearch_FullPageWithVideoElement(t *testing.T) {
	head := `<meta property="og:title" content="SSIS-001 Sample Video">
	<title>HAYAV - SSIS-001</title>`
	body := `<h1>SSIS-001</h1>
	<video src="https://cdn.example.com/video.mp4" controls></video>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "SSIS-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	assert.Equal(t, "SSIS-001 Sample Video", results[0].Label)
	require.Len(t, results[0].VideoSources, 1)
	assert.Contains(t, results[0].VideoSources[0].URL, "video.mp4")
}

func TestSearch_FullPageWithVideoSourceElements(t *testing.T) {
	head := `<meta property="og:title" content="Test Video">
	<title>HAYAV</title>`
	body := `<h1>Test Video</h1>
	<video id="player">
		<source src="https://cdn.example.com/720p.mp4" type="video/mp4">
		<source src="https://cdn.example.com/1080p.mp4" type="video/mp4">
	</video>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	assert.Len(t, results[0].VideoSources, 2)
	assert.Equal(t, "video/mp4", results[0].VideoSources[0].Type)
}

func TestSearch_FullPageWithIframe(t *testing.T) {
	head := `<title>HAYAV</title>`
	body := `<h1>Embedded Video</h1>
	<iframe src="https://player.example.com/embed/abc123"></iframe>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].VideoSources, 1)
	assert.Contains(t, results[0].VideoSources[0].URL, "player.example.com")
}

func TestSearch_JWPlayerConfig(t *testing.T) {
	head := `<title>HAYAV</title>`
	body := `<h1>JWPlayer Test</h1>
	<script>
		jwplayer("myplayer").setup({
			file: "https://stream.example.com/hls/master.m3u8",
			image: "https://example.com/poster.jpg"
		});
	</script>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].VideoSources, 1)
	assert.Contains(t, results[0].VideoSources[0].URL, "master.m3u8")
}

func TestSearch_PlyrConfig(t *testing.T) {
	head := `<title>HAYAV</title>`
	body := `<h1>Plyr Test</h1>
	<script>
		new Plyr("#player", {
			sources: [{ src: "https://cdn.example.com/video.mp4" }]
		});
	</script>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].VideoSources, 1)
	assert.Contains(t, results[0].VideoSources[0].URL, "video.mp4")
}

func TestSearch_MultiVersionCNSub(t *testing.T) {
	head := `<title>HAYAV</title>`
	body := `<h1>Multi Version Video</h1>
	<!-- Original version -->
	<div class="version-tab" id="original">
		<video src="https://cdn.example.com/original.mp4"></video>
	</div>
	<!-- CN Sub version -->
	<div class="version-tab cnsub-tab" id="cnsub">
		<video src="https://cdn.example.com/cnsub.mp4"></video>
	</div>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 2)

	versions := make(map[domain.VideoVersion]bool)
	for _, r := range results {
		versions[r.Version] = true
	}
	assert.True(t, versions[domain.VersionOriginal], "should have original version")
	assert.True(t, versions[domain.VersionCNSub], "should have cnsub version")
}

func TestSearch_TitlePriority_OgTitle(t *testing.T) {
	head := `<meta property="og:title" content="OG Title">
	<title>Page Title</title>`
	body := `<h1>H1 Title</h1>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	// No video elements, so NotFound
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
	assert.Equal(t, "OG Title", results[0].Label)
}

func TestSearch_TitlePriority_H1Fallback(t *testing.T) {
	head := `<title>Page Title</title>`
	body := `<h1>H1 Title</h1>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.Equal(t, "H1 Title", results[0].Label)
}

func TestSearch_TitlePriority_PageTitleFallback(t *testing.T) {
	head := `<title>Page Title</title>`
	body := ``

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	assert.Equal(t, "Page Title", results[0].Label)
}

func TestSearch_CloudflareBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>Just a moment... Checking your browser</body></html>`))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusBlocked, results[0].Status)
	assert.Contains(t, results[0].Error, "cloudflare")
}

func TestSearch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	_, err := s.Search(context.Background(), "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestSearch_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := newTestScraper(srv)
	_, err := s.Search(ctx, "ABC-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hayav fetch")
}

func TestSearch_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><head><title>T</title></head><body></body></html>"))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearch_DeduplicatesSources(t *testing.T) {
	head := `<title>Test</title>`
	body := `<video src="https://cdn.example.com/same.mp4"></video>
	<source src="https://cdn.example.com/same.mp4">`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].VideoSources, 1, "duplicate URLs should be deduplicated")
}

func TestSearch_AllPlayerTypesCombined(t *testing.T) {
	head := `<meta property="og:title" content="Combined Test">
	<title>HAYAV</title>`
	body := `<h1>Combined Test</h1>
	<video src="https://cdn.example.com/video_direct.mp4"></video>
	<video id="multi">
		<source src="https://cdn.example.com/video_source.mp4" type="video/mp4">
	</video>
	<iframe src="https://embed.example.com/player?id=xyz"></iframe>
	<script>
		jwplayer("jw").setup({ file: "https://stream.example.com/jw.m3u8" });
	</script>
	<script>
		new Plyr("#p", { sources: [{ src: "https://cdn.example.com/plyr.mp4" }] });
	</script>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage(head, body)))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].VideoSources, 5, "should extract all 5 unique sources")
	assert.Equal(t, "Combined Test", results[0].Label)
}

func TestSearch_PageURLInResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage("<title>T</title>", "")))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].PageURL, "/video/ABC-123/")
}

func TestNewWithClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlPage("<title>T</title>", "")))
	}))
	defer srv.Close()

	s := NewWithClient(srv.Client(), srv.URL)
	assert.Equal(t, Name, s.Name())
	assert.True(t, s.IsEnabled())
	assert.False(t, s.RequiresCFBypass())

	results, err := s.Search(context.Background(), "TEST-001")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].PageURL, srv.URL)
}

func TestClassifyMIME(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"video/mp4", "video/mp4"},
		{"application/x-mpegURL", "application/x-mpegURL"},
		{"", "video/mp4"},
		{"text/html", "text/html"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := classifyMIME(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "a", firstNonEmpty("a", "b"))
	assert.Equal(t, "b", firstNonEmpty("", "b"))
	assert.Equal(t, "", firstNonEmpty("", ""))
}

func TestResolveURL(t *testing.T) {
	pageURL := "https://hayav.com/video/ABC-123/"
	tests := []struct {
		raw      string
		expected string
	}{
		{"https://cdn.example.com/video.mp4", "https://cdn.example.com/video.mp4"},
		{"//cdn.example.com/video.mp4", "//cdn.example.com/video.mp4"},
		{"/assets/video.mp4", "https://hayav.com/assets/video.mp4"},
		{"video.mp4", "https://hayav.com/video/video.mp4"},
		{"  https://cdn.example.com/video.mp4  ", "https://cdn.example.com/video.mp4"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := resolveURL(tt.raw, pageURL)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveURL_Empty(t *testing.T) {
	assert.Equal(t, "", resolveURL("", "https://example.com/"))
	assert.Equal(t, "", resolveURL("   ", "https://example.com/"))
}
