package javmenu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Interface compliance tests
// ---------------------------------------------------------------------------

func TestName(t *testing.T) {
	s := New(true, domain.ProxyConfig{})
	assert.Equal(t, "javmenu", s.Name())
}

func TestFormatCodeReturnsUnchanged(t *testing.T) {
	s := New(true, domain.ProxyConfig{})
	tests := []string{"ABC-123", "abc123", "SSIS-001", "ABW-999", "raw code"}
	for _, code := range tests {
		assert.Equal(t, code, s.FormatCode(code), "FormatCode(%q)", code)
	}
}

func TestIsEnabled(t *testing.T) {
	assert.True(t, New(true, domain.ProxyConfig{}).IsEnabled())
	assert.False(t, New(false, domain.ProxyConfig{}).IsEnabled())
}

func TestRequiresCFBypass(t *testing.T) {
	assert.True(t, New(true, domain.ProxyConfig{}).RequiresCFBypass())
}

func TestGetProxyConfig(t *testing.T) {
	s := New(true, domain.ProxyConfig{URL: "http://proxy:8080", Enabled: true})
	pc := s.GetProxyConfig()
	assert.Equal(t, "http://proxy:8080", pc.URL)
	assert.True(t, pc.Enabled)

	s2 := New(true, domain.ProxyConfig{})
	pc2 := s2.GetProxyConfig()
	assert.Equal(t, "", pc2.URL)
	assert.False(t, pc2.Enabled)
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func TestDetectVideoType(t *testing.T) {
	tests := []struct {
		src, expected string
	}{
		{"https://example.com/video.mp4", "video/mp4"},
		{"https://example.com/video.m3u8", "application/x-mpegURL"},
		{"https://example.com/playlist.m3u8?token=abc", "application/x-mpegURL"},
		{"https://example.com/video.webm", "video/webm"},
		{"https://example.com/video.ts", "video/mp4"},
		{"https://example.com/video", "video/mp4"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, detectVideoType(tt.src), tt.src)
	}
}

func TestSourceExists(t *testing.T) {
	sources := []domain.VideoSource{
		{URL: "http://a.com/1.mp4"},
		{URL: "http://a.com/2.m3u8"},
	}
	assert.True(t, sourceExists(sources, "http://a.com/1.mp4"))
	assert.False(t, sourceExists(sources, "http://a.com/3.mp4"))
	assert.True(t, sourceExists(sources, "http://a.com/2.m3u8"))
}

func TestIsCNSubVariant(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"中文字幕", true},
		{"中文", true},
		{"Chinese Sub", true},
		{"cnsub", true},
		{"CN-SUB", true},
		{"漢化", true},
		{"chinese subtitle", true},
		{"original", false},
		{"HD", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, isCNSubVariant(tt.text), tt.text)
	}
}

func TestIsMosaicReduceVariant(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"去码", true},
		{"無碼", true},
		{"无码", true},
		{"破解", true},
		{"uncensored", true},
		{"leak", true},
		{"流出", true},
		{"mosaic reducing", true},
		{"破坏版", true},
		{"無修正", true},
		{"original", false},
		{"中文字幕", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, isMosaicReduceVariant(tt.text), tt.text)
	}
}

func TestIsCFBlocked(t *testing.T) {
	assert.True(t, isCFBlocked("Just a moment..."))
	assert.True(t, isCFBlocked("Checking your browser before accessing"))
	assert.True(t, isCFBlocked(`<div id="cf-browser-verification">`))
	assert.True(t, isCFBlocked("challenge-platform"))
	assert.True(t, isCFBlocked("Attention Required! | Cloudflare"))
	assert.False(t, isCFBlocked("<html><body>Welcome</body></html>"))
	assert.False(t, isCFBlocked(""))
}

// ---------------------------------------------------------------------------
// Search tests with httptest mock server
// ---------------------------------------------------------------------------

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func testScraperForServer(srv *httptest.Server, enabled bool, proxyEnabled bool) *Scraper {
	pc := domain.ProxyConfig{}
	if proxyEnabled {
		transport := &http.Transport{Proxy: http.ProxyURL(nil)}
		pc.Enabled = true
		return &Scraper{
			client: &http.Client{
				Timeout:   5 * time.Second,
				Transport: transport,
			},
			enabled:     enabled,
			proxyConfig: pc,
		}
	}
	return &Scraper{
		client: &http.Client{
			Timeout:   5 * time.Second,
		},
		enabled:     enabled,
		proxyConfig: pc,
	}
}

func TestSearchPrimaryPlayerVideo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
<div id="primary-player">
	<video src="https://cdn.example.com/video.mp4" controls></video>
</div>
</body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: &testTransport{target: srv.URL},
		},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]
	assert.Equal(t, "javmenu", r.SiteName)
	assert.Equal(t, domain.StatusSuccess, r.Status)
	assert.Equal(t, domain.VersionOriginal, r.Version)
	assert.Len(t, r.VideoSources, 1)
	assert.Equal(t, "https://cdn.example.com/video.mp4", r.VideoSources[0].URL)
	assert.Equal(t, "video/mp4", r.VideoSources[0].Type)
}

func TestSearchSeoMainVideo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
<div id="seo-main-video" src="https://cdn.example.com/seo.mp4"></div>
</body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:      &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled:     true,
		proxyConfig: domain.ProxyConfig{},
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "https://cdn.example.com/seo.mp4", results[0].VideoSources[0].URL)
	assert.Equal(t, "video/mp4", results[0].VideoSources[0].Type)
}

func TestSearchDeduplicatesDuplicateSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
<div id="primary-player">
	<video src="https://cdn.example.com/same.mp4"></video>
</div>
<div id="seo-main-video" src="https://cdn.example.com/same.mp4"></div>
</body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].VideoSources, 1, "duplicate sources should be deduplicated")
}

func TestSearchMultiVersionPlayerTabs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
<div id="primary-player">
	<video src="https://cdn.example.com/original.mp4"></video>
</div>
<div id="player-tab">
	<a class="nav-link" data-m3u8="https://cdn.example.com/cnsub.m3u8">中文字幕</a>
	<a class="nav-link" data-m3u8="https://cdn.example.com/uncensored.m3u8">無碼 破解</a>
	<a class="nav-link" data-m3u8="https://cdn.example.com/hd.m3u8">HD</a>
</div>
</body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 4)

	versions := make(map[domain.VideoVersion]int)
	for _, r := range results {
		versions[r.Version]++
	}

	assert.GreaterOrEqual(t, versions[domain.VersionOriginal], 1, "should have at least one original version")
	assert.Equal(t, 1, versions[domain.VersionCNSub], "should have one cnsub version")
	assert.Equal(t, 1, versions[domain.VersionMosaicReduce], "should have one mosaic reduce version")
}

func TestSearchTabsOnlyNoPrimaryPlayer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
<div id="player-tab">
	<a class="nav-link" data-m3u8="https://cdn.example.com/chinese.m3u8">中文字幕</a>
</div>
</body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.VersionCNSub, results[0].Version)
	assert.True(t, results[0].Subtitle)
	assert.Equal(t, "application/x-mpegURL", results[0].VideoSources[0].Type)
}

func TestSearchM3U8VideoType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
<div id="primary-player">
	<video src="https://cdn.example.com/stream.m3u8"></video>
</div>
</body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "application/x-mpegURL", results[0].VideoSources[0].Type)
}

func TestSearchNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "NONEXIST")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
	assert.Equal(t, domain.VersionOriginal, results[0].Version)
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
	assert.Contains(t, results[0].Error, "500")
}

func TestSearchCFBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html>Just a moment... Checking your browser</html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusBlocked, results[0].Status)
	assert.Contains(t, results[0].Error, "Cloudflare")
}

func TestSearchNoVideoFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><p>No video here</p></body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results, err := s.Search(ctx, "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
}

func TestSearchDisabledScraper(t *testing.T) {
	s := New(false, domain.ProxyConfig{})
	assert.False(t, s.IsEnabled())
}

func TestSearchEmptyDataM3U8Skipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
<div id="player-tab">
	<a class="nav-link" data-m3u8="">Empty</a>
	<a class="nav-link">No data attr</a>
</div>
</body></html>`))
	}))
	defer srv.Close()

	s := &Scraper{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: &testTransport{target: srv.URL}},
		enabled: true,
	}

	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

// ---------------------------------------------------------------------------
// Init registration test
// ---------------------------------------------------------------------------

func TestInitRegistersScraper(t *testing.T) {
	s := New(true, domain.ProxyConfig{})
	assert.Equal(t, "javmenu", s.Name())
	assert.True(t, s.IsEnabled())
	assert.True(t, s.RequiresCFBypass())
}

// ---------------------------------------------------------------------------
// testTransport rewrites all requests to a target server, used to
// bypass the hardcoded javmenu.com URL during tests.
type testTransport struct {
	target string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	targetURL := t.target
	if len(targetURL) > 7 && targetURL[:7] == "http://" {
		targetURL = targetURL[7:]
	}

	req.URL.Scheme = "http"
	req.URL.Host = targetURL
	req.URL.Path = "/"

	return http.DefaultTransport.RoundTrip(req)
}


