// Package sevenmmtv implements a domain.Scraper for https://7mmtv.sx.
//
// 7mmtv uses POST form-urlencoded search (not direct URL or GET search):
//
//	POST https://7mmtv.sx/zh/searchform_search/all/index.html
//	Content-Type: application/x-www-form-urlencoded; charset=UTF-8
//	Body: search_keyword=CODE&search_type=searchall&op=search
//
// The search returns an HTML results page. Results are filtered via CSS selector
// a[target='_top'][href$='.html'] excluding /searchall_search/ links, deduplicated
// by href, and matched against the input code via href slug + image alt text.
//
// Once a matching detail page is found, the scraper performs a second GET to
// extract video sources (iframe, video, source elements).
//
// The scraper uses a Cloudflare bypass pre-test on construction; if CF blocks,
// the scraper is created with IsEnabled() returning false.
package sevenmmtv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

const (
	// siteName is the unique scraper identifier registered in the global registry.
	siteName = "7mmtv"

	// baseURL is the canonical origin for all requests.
	baseURL = "https://7mmtv.sx"

	// searchPath is the POST endpoint for code search.
	searchPath = "/zh/searchform_search/all/index.html"

	// userAgent mimics Chrome on Windows for compatibility.
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// cfBlockMarkers are strings that appear in Cloudflare challenge pages.
var cfBlockMarkers = []string{
	"Just a moment",
	"Checking your browser",
	"cf-browser-verification",
	"challenge-platform",
	"Attention Required!",
	"Cloudflare",
}

// Scraper implements domain.Scraper for 7mmtv.sx.
// It is safe for concurrent use — all mutable fields are set at construction time.
type Scraper struct {
	client      *http.Client
	baseURL     string
	enabled     bool
	proxyConfig domain.ProxyConfig
	cfTested    bool
	cfPassed    bool
}

// Compile-time interface check.
var _ domain.Scraper = (*Scraper)(nil)

func init() {
	scraper.Register(New(domain.ProxyConfig{}))
}

// New creates a 7mmtv scraper with the given proxy configuration.
// The Cloudflare bypass test is deferred to the first Search() call so that
// proxy configuration (set via ApplyConfig after construction) is respected.
func New(config domain.ProxyConfig) *Scraper {
	transport := &http.Transport{}
	if config.Enabled && config.URL != "" {
		if pu, err := url.Parse(config.URL); err == nil {
			transport.Proxy = http.ProxyURL(pu)
		}
	}

	return &Scraper{
		client: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		baseURL:     baseURL,
		enabled:     true,
		proxyConfig: config,
	}
}

// newTestScraper creates a scraper with a custom HTTP client and base URL.
// Used by tests to inject httptest servers. Skips the CF pre-test.
func newTestScraper(client *http.Client, baseURLOverride string) *Scraper {
	return &Scraper{
		client:      client,
		baseURL:     baseURLOverride,
		enabled:     true,
		proxyConfig: domain.ProxyConfig{Enabled: false},
		cfTested:    true,
		cfPassed:    true,
	}
}

// Name returns the unique scraper identifier.
func (s *Scraper) Name() string { return siteName }

// FormatCode returns the code unchanged. 7mmtv accepts standard JAV codes.
func (s *Scraper) FormatCode(code string) string { return code }

// IsEnabled reports whether the scraper is currently active.
func (s *Scraper) IsEnabled() bool { return s.enabled }

// RequiresCFBypass reports that 7mmtv needs Cloudflare bypass.
func (s *Scraper) RequiresCFBypass() bool { return true }

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

// Search performs a two-phase scrape:
//  1. POST search form → parse result links → find matching detail page
//  2. GET detail page → extract video sources (iframe/video)
//
// Returns one VideoResult if found, StatusNotFound if no match, or an error.
func (s *Scraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("7mmtv: scraper is disabled")
	}

	if !s.cfTested {
		s.cfTested = true
		proxyURL := ""
		if s.proxyConfig.Enabled {
			proxyURL = s.proxyConfig.URL
		}
		result, err := scraper.CFBypassTest(ctx, baseURL+"/", proxyURL)
		if err != nil || !result.Passed {
			s.cfPassed = false
			errMsg := "CF bypass failed"
			if result != nil {
				errMsg = result.Error
			}
			log.Printf("7mmtv: Cloudflare bypass test failed, scraper disabled: %s", errMsg)
			return []domain.VideoResult{{
				SiteName: siteName,
				Status:   domain.StatusBlocked,
				Error:    "CF bypass failed: " + errMsg,
			}}, nil
		}
		s.cfPassed = true
	}
	if !s.cfPassed {
		return nil, fmt.Errorf("7mmtv: scraper is disabled")
	}

	searchURL := s.baseURL + searchPath

	// Phase 1: POST search form
	body := url.Values{
		"search_keyword": {code},
		"search_type":    {"searchall"},
		"op":             {"search"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("7mmtv: create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Origin", s.baseURL)
	req.Header.Set("Referer", s.baseURL+"/zh/")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("7mmtv: search request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2 MiB cap
	if err != nil {
		return nil, fmt.Errorf("7mmtv: read search body: %w", err)
	}

	// Cloudflare detection
	if isCFBlocked(string(rawBody)) {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusBlocked,
			PageURL:  searchURL,
			Version:  domain.VersionOriginal,
			Error:    "Cloudflare challenge detected",
		}}, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("7mmtv: search HTTP %d", resp.StatusCode)
	}

	// Phase 2: Parse search results for matching detail link
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("7mmtv: parse search HTML: %w", err)
	}

	detailURL := findMatchingResult(doc, code, s.baseURL)
	if detailURL == "" {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			PageURL:  searchURL,
			Version:  domain.VersionOriginal,
		}}, nil
	}

	// Phase 3: Fetch detail page and extract video sources
	return s.fetchDetail(ctx, detailURL, code)
}

// fetchDetail performs a GET to the detail page and extracts video sources.
func (s *Scraper) fetchDetail(ctx context.Context, detailURL, code string) ([]domain.VideoResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return nil, fmt.Errorf("7mmtv: create detail request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Referer", s.baseURL+searchPath)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("7mmtv: detail request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("7mmtv: read detail body: %w", err)
	}

	// Cloudflare detection on detail page too
	if isCFBlocked(string(rawBody)) {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusBlocked,
			PageURL:  detailURL,
			Version:  domain.VersionOriginal,
			Error:    "Cloudflare challenge detected on detail page",
		}}, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			PageURL:  detailURL,
			Version:  domain.VersionOriginal,
		}}, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("7mmtv: detail HTTP %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("7mmtv: parse detail HTML: %w", err)
	}

	sources := extractVideoSources(doc)
	if len(sources) == 0 {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			Version:  domain.VersionOriginal,
			PageURL:  detailURL,
			Error:    "no video sources found on detail page",
		}}, nil
	}

	return []domain.VideoResult{{
		SiteName:     siteName,
		Status:       domain.StatusSuccess,
		Version:      domain.VersionOriginal,
		PageURL:      detailURL,
		VideoSources: sources,
	}}, nil
}

// findMatchingResult parses the search results HTML and returns the URL of
// the first matching video detail page, or empty string if none found.
//
// Algorithm:
//  1. Select all <a target="_top"> with href ending in ".html"
//  2. Exclude links containing "/searchall_search/" (these are search pages, not videos)
//  3. Deduplicate by href
//  4. For each candidate, extract the URL slug and img alt text
//  5. Match: slug contains normalized code AND (if img alt exists) alt contains normalized code
func findMatchingResult(doc *goquery.Document, code, baseURL string) string {
	normalizedCode := scraper.NormalizeCode(code)
	seen := make(map[string]bool)

	var bestMatch string

	doc.Find(`a[target="_top"][href$=".html"]`).Each(func(_ int, sel *goquery.Selection) {
		if bestMatch != "" {
			return
		}

		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		// Exclude search page links
		if strings.Contains(href, "/searchall_search/") {
			return
		}

		// Deduplicate by resolved href
		resolved := resolveURL(baseURL, href)
		if seen[resolved] {
			return
		}
		seen[resolved] = true

		// Match code via href slug
		slug := extractSlug(href)
		slugNormalized := scraper.NormalizeCode(slug)
		if !strings.Contains(slugNormalized, normalizedCode) {
			return
		}

		// Confirm via image alt text (if available)
		imgAlt := strings.TrimSpace(sel.Find("img").AttrOr("alt", ""))
		if imgAlt != "" {
			altNormalized := scraper.NormalizeCode(imgAlt)
			if !strings.Contains(altNormalized, normalizedCode) {
				return // alt exists but doesn't match — skip
			}
		}

		bestMatch = resolved
	})

	return bestMatch
}

// extractSlug extracts the last path segment (minus .html extension) from a URL path.
//
// Examples:
//
//	"/zh/video/abc-123.html"    → "abc-123"
//	"/zh/video/mide-999.html"   → "mide-999"
//	"/video/ssis-001.html"      → "ssis-001"
//	"/abc-123.html?q=1"         → "abc-123"
func extractSlug(href string) string {
	// Remove query string
	if idx := strings.Index(href, "?"); idx != -1 {
		href = href[:idx]
	}

	// Get the last path segment
	lastSlash := strings.LastIndex(href, "/")
	if lastSlash >= 0 {
		href = href[lastSlash+1:]
	}

	// Strip .html extension
	href = strings.TrimSuffix(href, ".html")

	return href
}

// extractVideoSources finds all playable video sources in a parsed HTML document.
// It inspects <iframe>, <video>, and <source> elements, deduplicating by URL.
func extractVideoSources(doc *goquery.Document) []domain.VideoSource {
	var sources []domain.VideoSource
	seen := make(map[string]bool)

	// 1) <iframe src="...">
	doc.Find("iframe[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		if src == "" || seen[src] {
			return
		}
		seen[src] = true
		sources = append(sources, domain.VideoSource{
			URL:  src,
			Type: "text/html",
		})
	})

	// 2) <video src="...">
	doc.Find("video[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		if src == "" || seen[src] {
			return
		}
		seen[src] = true
		sources = append(sources, domain.VideoSource{
			URL:  src,
			Type: detectVideoType(src),
		})
	})

	// 3) <video><source src="..." type="..."></video>
	doc.Find("video source[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		mime, _ := el.Attr("type")
		if src == "" || seen[src] {
			return
		}
		seen[src] = true
		if mime == "" {
			mime = detectVideoType(src)
		}
		sources = append(sources, domain.VideoSource{
			URL:  src,
			Type: mime,
		})
	})

	// 4) Standalone <source src="...">
	doc.Find("source[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		mime, _ := el.Attr("type")
		if src == "" || seen[src] {
			return
		}
		seen[src] = true
		if mime == "" {
			mime = detectVideoType(src)
		}
		sources = append(sources, domain.VideoSource{
			URL:  src,
			Type: mime,
		})
	})

	return sources
}

// detectVideoType infers the MIME type from a URL's file extension.
func detectVideoType(rawURL string) string {
	lower := strings.ToLower(rawURL)
	switch {
	case strings.Contains(lower, ".m3u8"):
		return "application/x-mpegURL"
	case strings.HasSuffix(lower, ".mp4"):
		return "video/mp4"
	case strings.HasSuffix(lower, ".webm"):
		return "video/webm"
	case strings.HasSuffix(lower, ".ts"):
		return "video/mp2t"
	default:
		return "video/mp4"
	}
}

// resolveURL resolves a possibly-relative URL against a base URL.
// If the href is already absolute, it is returned unchanged.
func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") || strings.HasPrefix(href, "//") {
		return href
	}

	u, err := url.Parse(base)
	if err != nil {
		return base + href
	}

	ref, err := url.Parse(href)
	if err != nil {
		return base + href
	}

	return u.ResolveReference(ref).String()
}

// isCFBlocked checks whether the response body contains Cloudflare challenge markers.
func isCFBlocked(body string) bool {
	for _, marker := range cfBlockMarkers {
		if strings.Contains(body, marker) {
			return true
		}
	}
	return false
}
