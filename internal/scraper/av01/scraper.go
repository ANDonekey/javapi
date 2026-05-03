// Package av01 implements a domain.Scraper for https://www.av01.media.
//
// Search flow:
//  1. POST https://www.av01.media/cn/search?q=CODE (form-urlencoded body: q=CODE)
//  2. Parse HTML results with goquery → extract video ID from result links
//  3. Build video page URL: https://www.av01.media/cn/video/{id}/{code_lower}
//  4. Fetch video page → extract M3U8 URL from page source
//  5. Fall back to constructed URL: /api/v1/videos/{id}/manifest/master.m3u8
package av01

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

const (
	siteName      = "av01"
	baseURL       = "https://www.av01.media"
	searchPath    = "/cn/search"
	videoPathTmpl = "/cn/video/%s/%s"
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	cfTestURL     = "https://www.av01.media/"
)

var (
	m3u8Pattern    = regexp.MustCompile(`/api/v1/videos/\d+/manifest/[^"'\s]+\.m3u8`)
	videoIDPattern = regexp.MustCompile(`/cn/video/(\d+)/`)
)

// Scraper implements domain.Scraper for av01.media.
type Scraper struct {
	client      *http.Client
	enabled     bool
	proxyConfig domain.ProxyConfig
	baseURL     string
	cfTested    bool
	cfPassed    bool
}

// Compile-time interface check.
var _ domain.Scraper = (*Scraper)(nil)

func init() {
	scraper.Register(New(domain.ProxyConfig{}))
}

// New creates an AV01 scraper with the given proxy configuration.
func New(config domain.ProxyConfig) *Scraper {
	client := &http.Client{Timeout: 15 * time.Second}
	if config.Enabled && config.URL != "" {
		proxyURL, err := url.Parse(config.URL)
		if err == nil {
			client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}
	return &Scraper{
		client:      client,
		enabled:     true,
		proxyConfig: config,
		baseURL:     baseURL,
	}
}

// NewWithClient returns a Scraper using a custom http.Client and base URL.
// Primarily used in tests to inject httptest servers.
func NewWithClient(client *http.Client, baseURL string) *Scraper {
	return &Scraper{
		client:      client,
		enabled:     true,
		proxyConfig: domain.ProxyConfig{Enabled: false},
		baseURL:     baseURL,
		cfTested:    true,
		cfPassed:    true,
	}
}

// Name returns the unique scraper identifier.
func (s *Scraper) Name() string { return siteName }

// FormatCode returns the code unchanged. AV01 accepts standard JAV codes directly.
func (s *Scraper) FormatCode(code string) string { return code }

// IsEnabled reports whether this scraper is currently active.
func (s *Scraper) IsEnabled() bool { return s.enabled }

// RequiresCFBypass reports whether this scraper needs Cloudflare bypass.
func (s *Scraper) RequiresCFBypass() bool { return false }

// GetProxyConfig returns the proxy configuration for this scraper.
func (s *Scraper) GetProxyConfig() domain.ProxyConfig { return s.proxyConfig }

// SetProxyConfig updates the proxy configuration for this scraper.
func (s *Scraper) SetProxyConfig(pc domain.ProxyConfig) {
	s.proxyConfig = pc
	if pc.Enabled && pc.URL != "" {
		proxyURL, err := url.Parse(pc.URL)
		if err == nil {
			s.client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}
}

// Search performs an HTML-based search on AV01 and returns video results.
//
// Flow:
//  1. POST form-urlencoded search to /cn/search?q=CODE
//  2. Parse HTML with goquery to find matching video IDs
//  3. Fetch the video page and extract M3U8 sources
//  4. Fall back to a constructed M3U8 URL if extraction yields nothing
func (s *Scraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("av01: scraper is disabled")
	}

	if !s.cfTested {
		s.cfTested = true
		proxyURL := ""
		if s.proxyConfig.Enabled {
			proxyURL = s.proxyConfig.URL
		}
		result, err := scraper.CFBypassTest(ctx, cfTestURL, proxyURL)
		if err != nil || !result.Passed {
			s.cfPassed = false
			errMsg := "CF bypass failed"
			if result != nil {
				errMsg = result.Error
			}
			log.Printf("av01: Cloudflare bypass test failed, scraper disabled: %s", errMsg)
			return []domain.VideoResult{{
				SiteName: siteName,
				Status:   domain.StatusBlocked,
				Error:    "CF bypass failed: " + errMsg,
			}}, nil
		}
		s.cfPassed = true
	}
	if !s.cfPassed {
		return nil, fmt.Errorf("av01: scraper is disabled")
	}

	searchURL := s.baseURL + searchPath + "?q=" + url.QueryEscape(code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL,
		strings.NewReader("q="+url.QueryEscape(code)))
	if err != nil {
		return nil, fmt.Errorf("av01: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusError,
			Version:  domain.VersionOriginal,
			PageURL:  searchURL,
			Error:    fmt.Sprintf("request failed: %v", err),
		}}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusError,
			Version:  domain.VersionOriginal,
			PageURL:  searchURL,
			Error:    fmt.Sprintf("HTTP %d", resp.StatusCode),
		}}, nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("av01: parse search HTML: %w", err)
	}

	normalized := scraper.NormalizeCode(code)
	var videoID string

	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		if videoID != "" {
			return
		}
		href, exists := sel.Attr("href")
		if !exists {
			return
		}
		matches := videoIDPattern.FindStringSubmatch(href)
		if len(matches) < 2 {
			return
		}
		linkText := scraper.NormalizeCode(strings.TrimSpace(sel.Text()))
		if strings.Contains(linkText, normalized) || strings.Contains(scraper.NormalizeCode(href), normalized) {
			videoID = matches[1]
		}
	})

	if videoID == "" {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			Version:  domain.VersionOriginal,
			PageURL:  searchURL,
		}}, nil
	}

	codeLower := strings.ToLower(code)
	pageURL := fmt.Sprintf(s.baseURL+videoPathTmpl, videoID, codeLower)

	videoSources := s.fetchVideoSources(ctx, pageURL, videoID)
	if len(videoSources) == 0 {
		constructed := fmt.Sprintf(s.baseURL+"/api/v1/videos/%s/manifest/master.m3u8", videoID)
		videoSources = []domain.VideoSource{
			{URL: constructed, Type: "application/x-mpegURL"},
		}
	}

	return []domain.VideoResult{{
		SiteName:     siteName,
		Status:       domain.StatusSuccess,
		Version:      domain.VersionOriginal,
		PageURL:      pageURL,
		VideoSources: videoSources,
	}}, nil
}

// fetchVideoSources fetches the video page and extracts M3U8 URLs.
func (s *Scraper) fetchVideoSources(ctx context.Context, pageURL, videoID string) []domain.VideoSource {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil
	}

	bodyStr := string(bodyBytes)

	matches := m3u8Pattern.FindAllString(bodyStr, -1)
	if len(matches) > 0 {
		seen := make(map[string]bool, len(matches))
		sources := make([]domain.VideoSource, 0, len(matches))
		for _, m := range matches {
			if seen[m] {
				continue
			}
			seen[m] = true
			fullURL := m
			if !strings.HasPrefix(m, "http") {
				fullURL = s.baseURL + m
			}
			sources = append(sources, domain.VideoSource{
				URL:  fullURL,
				Type: "application/x-mpegURL",
			})
		}
		return sources
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return nil
	}

	var sources []domain.VideoSource
	doc.Find("source, video").Each(func(_ int, sel *goquery.Selection) {
		for _, attr := range []string{"src", "data-src"} {
			src, exists := sel.Attr(attr)
			if exists && strings.Contains(src, ".m3u8") {
				fullURL := src
				if !strings.HasPrefix(src, "http") {
					fullURL = s.baseURL + src
				}
				sources = append(sources, domain.VideoSource{
					URL:  fullURL,
					Type: "application/x-mpegURL",
				})
				return
			}
		}
	})

	return sources
}
