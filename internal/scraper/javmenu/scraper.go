// Package javmenu implements a domain.Scraper for the javmenu.com video site.
// It parses the video page directly and extracts video source URLs.
package javmenu

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

// defaultTimeout is the default HTTP client timeout for scraping javmenu.
const defaultTimeout = 8 * time.Second

var _ domain.Scraper = (*Scraper)(nil)

// Scraper implements domain.Scraper for javmenu.com.
type Scraper struct {
	client      *http.Client
	enabled     bool
	proxyConfig domain.ProxyConfig
}

// New creates a new javmenu Scraper with the given configuration.
func New(enabled bool, config domain.ProxyConfig) *Scraper {
	s := &Scraper{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		enabled:     enabled,
		proxyConfig: config,
	}
	s.applyProxy()
	return s
}

// NewWithProxy creates an enabled scraper with the given proxy configuration.
func NewWithProxy(config domain.ProxyConfig) *Scraper {
	return New(true, config)
}

// Name returns the unique scraper identifier.
func (s *Scraper) Name() string { return "javmenu" }

// FormatCode returns the code unchanged — javmenu expects the raw code format.
func (s *Scraper) FormatCode(code string) string { return code }

// IsEnabled reports whether this scraper is currently enabled.
func (s *Scraper) IsEnabled() bool { return s.enabled }

// RequiresCFBypass reports that javmenu may need Cloudflare bypass.
func (s *Scraper) RequiresCFBypass() bool { return true }

// GetProxyConfig returns the proxy configuration for this scraper.
func (s *Scraper) GetProxyConfig() domain.ProxyConfig { return s.proxyConfig }

// SetProxyConfig updates the proxy configuration for this scraper.
func (s *Scraper) SetProxyConfig(pc domain.ProxyConfig) {
	s.proxyConfig = pc
	s.applyProxy()
}

func (s *Scraper) applyProxy() {
	if s.proxyConfig.Enabled && s.proxyConfig.URL != "" {
		proxyURL, err := url.Parse(s.proxyConfig.URL)
		if err == nil {
			s.client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}
}

// Search fetches the javmenu video page and extracts video source URLs.
//
// It parses the HTML for video sources using the following selectors:
//   - #primary-player video[src] — primary player video element
//   - #seo-main-video[src] — SEO fallback element with src attribute
//   - #player-tab .nav-link[data-m3u8] — alternate video tabs (multi-version)
//
// Returns one or more VideoResult entries depending on the number of
// versions found on the page.
func (s *Scraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	pageURL := fmt.Sprintf("https://javmenu.com/%s", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("javmenu create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return []domain.VideoResult{{
			SiteName: s.Name(),
			Status:   domain.StatusError,
			PageURL:  pageURL,
			Version:  domain.VersionOriginal,
			Error:    err.Error(),
		}}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []domain.VideoResult{{
			SiteName: s.Name(),
			Status:   domain.StatusNotFound,
			PageURL:  pageURL,
			Version:  domain.VersionOriginal,
		}}, nil
	}

	if resp.StatusCode >= 400 {
		return []domain.VideoResult{{
			SiteName: s.Name(),
			Status:   domain.StatusError,
			PageURL:  pageURL,
			Version:  domain.VersionOriginal,
			Error:    fmt.Sprintf("HTTP %d", resp.StatusCode),
		}}, nil
	}

	// Check for Cloudflare challenge before parsing
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, fmt.Errorf("javmenu read body: %w", err)
	}
	bodyStr := string(bodyBytes)
	if isCFBlocked(bodyStr) {
		return []domain.VideoResult{{
			SiteName: s.Name(),
			Status:   domain.StatusBlocked,
			PageURL:  pageURL,
			Version:  domain.VersionOriginal,
			Error:    "Cloudflare challenge detected",
		}}, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return nil, fmt.Errorf("javmenu parse html: %w", err)
	}

	results := collectVideoResults(doc, s.Name(), pageURL)
	if len(results) == 0 {
		return []domain.VideoResult{{
			SiteName: s.Name(),
			Status:   domain.StatusNotFound,
			PageURL:  pageURL,
			Version:  domain.VersionOriginal,
		}}, nil
	}

	return results, nil
}

// collectVideoResults extracts video sources from the parsed HTML document.
func collectVideoResults(doc *goquery.Document, siteName, pageURL string) []domain.VideoResult {
	var results []domain.VideoResult

	// Collect primary video sources from #primary-player
	var primarySources []domain.VideoSource
	doc.Find("#primary-player video[src]").Each(func(_ int, sel *goquery.Selection) {
		if src, ok := sel.Attr("src"); ok && src != "" {
			primarySources = append(primarySources, domain.VideoSource{
				URL:  src,
				Type: detectVideoType(src),
			})
		}
	})

	// Collect SEO fallback video
	doc.Find("#seo-main-video[src]").Each(func(_ int, sel *goquery.Selection) {
		if src, ok := sel.Attr("src"); ok && src != "" {
			// Only add if not already found in primary player
			if !sourceExists(primarySources, src) {
				primarySources = append(primarySources, domain.VideoSource{
					URL:  src,
					Type: detectVideoType(src),
				})
			}
		}
	})

	// Add main (original) result if primary sources found
	if len(primarySources) > 0 {
		results = append(results, domain.VideoResult{
			SiteName:     siteName,
			Status:       domain.StatusSuccess,
			Version:      domain.VersionOriginal,
			PageURL:      pageURL,
			VideoSources: primarySources,
		})
	}

	// Check player-tab for multi-version links
	doc.Find("#player-tab .nav-link[data-m3u8]").Each(func(_ int, sel *goquery.Selection) {
		m3u8, _ := sel.Attr("data-m3u8")
		if m3u8 == "" {
			return
		}
		tabText := strings.TrimSpace(sel.Text())

		result := domain.VideoResult{
			SiteName: siteName,
			Status:   domain.StatusSuccess,
			PageURL:  pageURL,
			VideoSources: []domain.VideoSource{{
				URL:  m3u8,
				Type: "application/x-mpegURL",
			}},
		}

		switch {
		case isCNSubVariant(tabText):
			result.Version = domain.VersionCNSub
			result.Label = tabText
			result.Subtitle = true
		case isMosaicReduceVariant(tabText):
			result.Version = domain.VersionMosaicReduce
			result.Label = tabText
			result.Leak = true
		default:
			// Other tabs are treated as original variants (e.g., different quality/resolution)
			result.Version = domain.VersionOriginal
			result.Label = tabText
			// If we already have primary sources, skip adding duplicate original
			if len(primarySources) > 0 && !hasM3U8InSources(primarySources, m3u8) {
				// Different quality variant — add as additional original source
			}
		}

		results = append(results, result)
	})

	return results
}

// detectVideoType returns the MIME type based on URL extension.
func detectVideoType(src string) string {
	lower := strings.ToLower(src)
	switch {
	case strings.HasSuffix(lower, ".m3u8") || strings.Contains(lower, ".m3u8"):
		return "application/x-mpegURL"
	case strings.HasSuffix(lower, ".mp4"):
		return "video/mp4"
	case strings.HasSuffix(lower, ".webm"):
		return "video/webm"
	default:
		return "video/mp4"
	}
}

// sourceExists checks whether a source URL is already in the slice.
func sourceExists(sources []domain.VideoSource, url string) bool {
	for _, s := range sources {
		if s.URL == url {
			return true
		}
	}
	return false
}

// hasM3U8InSources checks whether any source in the slice has the given M3U8 URL.
func hasM3U8InSources(sources []domain.VideoSource, m3u8 string) bool {
	for _, s := range sources {
		if s.URL == m3u8 {
			return true
		}
	}
	return false
}

// cnSubKeywords are Chinese/Japanese keywords indicating a Chinese subtitle version.
var cnSubKeywords = []string{
	"中文字幕", "中文", "字幕", "Chinese", "cnsub", "CN-SUB",
	"漢化", "汉化", "chinese subtitle",
}

// mosaicKeywords are keywords indicating a mosaic-reduced/uncensored version.
var mosaicKeywords = []string{
	"去码", "無碼", "无码", "破码", "破解", "uncensored",
	"leak", "流出", "mosaic", "reducing", "reduction",
	"破坏", "破坏版", "無修正",
}

// isCNSubVariant checks whether the tab text indicates a Chinese subtitle variant.
func isCNSubVariant(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range cnSubKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// isMosaicReduceVariant checks whether the tab text indicates a mosaic-reduced variant.
func isMosaicReduceVariant(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range mosaicKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// cfBlockMarkers are strings that appear in Cloudflare challenge pages.
var cfBlockMarkers = []string{
	"Just a moment",
	"Checking your browser",
	"cf-browser-verification",
	"challenge-platform",
	"Attention Required!",
	"Cloudflare",
}

// isCFBlocked checks if the response body contains Cloudflare challenge markers.
func isCFBlocked(body string) bool {
	for _, marker := range cfBlockMarkers {
		if strings.Contains(body, marker) {
			return true
		}
	}
	return false
}

func init() {
	scraper.Register(New(true, domain.ProxyConfig{}))
}
