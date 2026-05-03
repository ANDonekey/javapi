package jable

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rewriteTransport struct {
	base    string
	wrapped http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://jable.tv") {
		u := strings.Replace(req.URL.String(), "https://jable.tv", rt.base, 1)
		var err error
		req.URL, err = req.URL.Parse(u)
		if err != nil {
			return nil, err
		}
		req.Host = req.URL.Host
	}
	return rt.wrapped.RoundTrip(req)
}

func newTestScraper(srv *httptest.Server) *Scraper {
	return &Scraper{
		enabled: true,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &rewriteTransport{
				base:    srv.URL,
				wrapped: srv.Client().Transport,
			},
		},
	}
}

var padding5k = strings.Repeat("x", 5100)

func searchResultsHTML(codes ...string) string {
	var items string
	for _, c := range codes {
		items += fmt.Sprintf(`<div class="video-img-box"><a href="/videos/%s/">%s Title</a></div>`, c, c)
	}
	return fmt.Sprintf(`<html lang="en"><head><title>Search</title></head><body><div class="container"><h1>Search Results</h1>%s<p>Found %d results</p><p>%s</p></div></body></html>`, items, len(codes), padding5k)
}

func searchResultsWithTitlesHTML(code string) string {
	return fmt.Sprintf(`<html lang="en"><head><title>Search</title></head><body><div class="container"><h1>Results</h1><h6 class="title"><a href="/videos/%s/">%s Video Title</a></h6><p>%s</p></div></body></html>`, code, code, padding5k)
}

func videoPageHTML(playerURL string) string {
	return fmt.Sprintf(`<html lang="en"><head><title>Video Page</title></head><body><div class="container"><h1>Player</h1><video id="player"><source src="%s" type="application/x-mpegURL"></video><p>%s</p></div></body></html>`, playerURL, padding5k)
}

func videoPageWithTabbedPlayerHTML(playerURL, cnsubURL string) string {
	return fmt.Sprintf(`<html lang="en"><head><title>Video Page</title></head><body><div class="container"><h1>Player</h1><video id="player"><source src="%s" type="application/x-mpegURL"></video><div class="nav-tabs"><a href="#">Original</a><a href="#" data-src="%s">中文字幕</a></div><p>%s</p></div></body></html>`, playerURL, cnsubURL, padding5k)
}

func searchResultsWithCNSubHTML(code string) string {
	return fmt.Sprintf(`<html lang="en"><head><title>Search</title></head><body><div class="container"><h1>Results</h1><a href="/videos/%s-ch/">中文 %s 字幕</a><p>%s</p></div></body></html>`, code, code, padding5k)
}

func emptySearchHTML() string {
	return `<html><body>No results.</body></html>`
}

func videoPageWithIframeHTML(iframeSrc string) string {
	return fmt.Sprintf(`<html lang="en"><head><title>Video</title></head><body><div class="container"><h1>Player</h1><iframe src="%s"></iframe><p>%s</p></div></body></html>`, iframeSrc, padding5k)
}

func videoPageWithDataAttrHTML(dataSrc string) string {
	return fmt.Sprintf(`<html lang="en"><head><title>Video</title></head><body><div class="container"><h1>Player</h1><div id="player" data-src="%s"></div><p>%s</p></div></body></html>`, dataSrc, padding5k)
}

func TestFormatCode(t *testing.T) {
	s := &Scraper{}
	tests := []struct {
		input, expected string
	}{
		{"abc_123", "ABC-123"},
		{"abc 123", "ABC-123"},
		{"ABC-123", "ABC-123"},
		{"ABC 123", "ABC-123"},
		{"ABC_123", "ABC-123"},
		{"abc-123", "ABC-123"},
		{"  ABC_123  ", "ABC-123"},
		{"ssis-001", "SSIS-001"},
		{"IPX-811", "IPX-811"},
		{"ipx_811", "IPX-811"},
		{"abp_999", "ABP-999"},
		{"mide_777", "MIDE-777"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, s.FormatCode(tt.input))
		})
	}
}

func TestFormatCodeExtractsCode(t *testing.T) {
	s := &Scraper{}
	assert.Equal(t, "ABC-123", s.FormatCode("watch ABC_123 online free"))
}

func TestInterfaceCompliance(t *testing.T) {
	s := &Scraper{enabled: true}
	assert.Equal(t, "jable", s.Name())
	assert.True(t, s.IsEnabled())
	assert.True(t, s.RequiresCFBypass())
	assert.Equal(t, domain.ProxyConfig{}, s.GetProxyConfig())
}

func TestSetProxyConfig(t *testing.T) {
	s := &Scraper{enabled: true, client: &http.Client{}}
	pc := domain.ProxyConfig{URL: "http://proxy.example.com:8080", Enabled: true}
	s.SetProxyConfig(pc)
	assert.Equal(t, pc, s.GetProxyConfig())
}

func TestSearchFoundWithPlayer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.Contains(path, "/search/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, searchResultsHTML("ABC-123"))
			return
		}
		if strings.Contains(path, "/videos/ABC-123") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, videoPageHTML("https://cdn.jable.tv/abc-123.m3u8"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "abc_123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "jable", results[0].SiteName)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	require.Len(t, results[0].VideoSources, 1)
	assert.Equal(t, "application/x-mpegURL", results[0].VideoSources[0].Type)
}

func TestSearchFoundWithH6Title(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.Contains(path, "/search/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, searchResultsWithTitlesHTML("IPX-811"))
			return
		}
		if strings.Contains(path, "/videos/IPX-811") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, videoPageHTML("https://cdn.jable.tv/ipx-811.mp4"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ipx_811")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
	assert.Equal(t, "video/mp4", results[0].VideoSources[0].Type)
}

func TestSearchFoundWithCNSubMultiVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.Contains(path, "/search/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, searchResultsHTML("ABC-123"))
			return
		}
		if strings.Contains(path, "/videos/ABC-123") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, videoPageWithTabbedPlayerHTML(
				"https://cdn.jable.tv/abc-123.m3u8",
				"https://cdn.jable.tv/abc-123-cn.m3u8",
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "abc_123")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	assert.Equal(t, domain.VersionCNSub, results[1].Version)
	assert.Equal(t, "Chinese Sub", results[1].Label)
	assert.NotEqual(t, results[0].VideoSources[0].URL, results[1].VideoSources[0].URL)
}

func TestSearchCNSubOnly(t *testing.T) {
	// When only a cnsub-marked link exists in search results, findVideoLink
	// matches it as the primary video URL (since it contains the code).
	// The result is returned as original; cnsub detection happens on the
	// video page (tabs/versions), not from search results alone.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.Contains(path, "/search/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, searchResultsWithCNSubHTML("ABC-123"))
			return
		}
		if strings.Contains(path, "/videos/ABC-123-ch") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, videoPageHTML("https://cdn.jable.tv/abc-123-cn.m3u8"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "abc_123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.VersionOriginal, results[0].Version)
	assert.Equal(t, domain.StatusSuccess, results[0].Status)
}

func TestSearchNotFoundShortBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, emptySearchHTML())
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "XYZ-999")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchNotFoundNoLinkMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, searchResultsHTML("XYZ-001", "XYZ-002"))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchHTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `<html><body><h1>Not Found</h1><p>The page you are looking for does not exist on this server and we cannot find it anywhere in our database.</p></body></html>`)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusNotFound, results[0].Status)
}

func TestSearchNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	s := &Scraper{
		enabled: true,
		client: &http.Client{
			Timeout: 10 * time.Millisecond,
		},
	}
	results, err := s.Search(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, domain.StatusError, results[0].Status)
	assert.NotEmpty(t, results[0].Error)
}

func TestExtractPlayersFromIframe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.Contains(path, "/search/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, searchResultsHTML("ABC-123"))
			return
		}
		if strings.Contains(path, "/videos/ABC-123") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, videoPageWithIframeHTML("https://player.jable.tv/embed/abc-123"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "abc_123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].VideoSources, 1)
	assert.Contains(t, results[0].VideoSources[0].URL, "player.jable.tv")
}

func TestExtractPlayersFromDataAttr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if strings.Contains(path, "/search/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, searchResultsHTML("ABC-123"))
			return
		}
		if strings.Contains(path, "/videos/ABC-123") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, videoPageWithDataAttrHTML("https://cdn.jable.tv/v/abc-123.m3u8"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	results, err := s.Search(context.Background(), "abc_123")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].VideoSources, 1)
	assert.Equal(t, "application/x-mpegURL", results[0].VideoSources[0].Type)
}

func TestCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "https://jable.tv/", r.Header.Get("Referer"))
		assert.Equal(t, "https://jable.tv", r.Header.Get("Origin"))
		assert.True(t, strings.HasPrefix(r.Header.Get("Accept-Language"), "zh-CN"))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, searchResultsHTML("ABC-123"))
	}))
	defer srv.Close()

	s := newTestScraper(srv)
	_, err := s.Search(context.Background(), "abc_123")
	require.NoError(t, err)
}

func TestDetectVideoType(t *testing.T) {
	tests := []struct {
		url, expected string
	}{
		{"https://cdn.example.com/video.m3u8", "application/x-mpegURL"},
		{"https://cdn.example.com/video.mp4", "video/mp4"},
		{"https://cdn.example.com/video.ts", "video/mp2t"},
		{"https://cdn.example.com/video/index.m3u8?token=xxx", "application/x-mpegURL"},
		{"https://cdn.example.com/video", "video/mp4"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, detectVideoType(tt.url))
		})
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base, href, expected string
	}{
		{"https://jable.tv", "/videos/abc-123/", "https://jable.tv/videos/abc-123/"},
		{"https://jable.tv/", "https://other.com/video.mp4", "https://other.com/video.mp4"},
		{"https://jable.tv/search/ABC-123/", "../videos/ABC-123/", "https://jable.tv/search/videos/ABC-123/"},
	}
	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolveURL(tt.base, tt.href))
		})
	}
}


