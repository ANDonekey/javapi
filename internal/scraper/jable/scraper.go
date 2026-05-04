// Package jable implements a scraper for jable.tv video search.
//
// Jable uses a custom code format: whitespace and underscores are replaced
// with hyphens, and the entire code is uppercased (abc_123 → ABC-123).
//
// The search flow:
//  1. GET https://jable.tv/search/{formatted_code}/ with custom headers
//  2. Parse search results for a matching video link
//  3. Follow the link to the video page
//  4. Extract player sources (m3u8, mp4) from the video page
//  5. Detect multi-version variants (cnsub, original)
//
// 404 detection uses HTTP status code, final URL, and response body length.
package jable

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

const (
	siteName = "jable"
	baseURL  = "https://jable.tv"
)

// formatRegex matches runs of whitespace and/or underscores to be replaced with hyphens.
var formatRegex = regexp.MustCompile(`[\s_]+`)

var _ domain.Scraper = (*Scraper)(nil)

// Scraper scrapes jable.tv for video embed links.
type Scraper struct {
	client      *http.Client
	proxyConfig domain.ProxyConfig
	enabled     bool
}

func init() {
	s := &Scraper{
		client:  scraper.NewCFClient(""),
		enabled: true,
	}

	s.SetProxyConfig(scraper.ProxyFromEnv())
	scraper.Register(s)
}

// Name returns the unique scraper identifier.
func (s *Scraper) Name() string { return siteName }

// IsEnabled reports whether this scraper is active.
func (s *Scraper) IsEnabled() bool { return s.enabled }

// RequiresCFBypass reports that jable.tv may need Cloudflare bypass.
func (s *Scraper) RequiresCFBypass() bool { return true }

// GetProxyConfig returns per-site proxy settings.
func (s *Scraper) GetProxyConfig() domain.ProxyConfig { return s.proxyConfig }

// SetProxyConfig applies proxy configuration for this scraper.
func (s *Scraper) SetProxyConfig(pc domain.ProxyConfig) {
	s.proxyConfig = pc
	if pc.Enabled && pc.URL != "" {
		s.client = scraper.NewCFClient(pc.URL)
	}
}

// SetClient replaces the HTTP client. Used for testing.
func (s *Scraper) SetClient(c *http.Client) {
	s.client = c
}

// FormatCode converts a JAV code into jable.tv search format:
// replace whitespace/underscores with hyphens, then uppercase.
//
//	"abc_123"  → "ABC-123"
//	"abc 123"  → "ABC-123"
//	"ABC-123"  → "ABC-123"
//	"ABC 123"  → "ABC-123"
func (s *Scraper) FormatCode(code string) string {
	code = scraper.ExtractCode(code)
	code = formatRegex.ReplaceAllString(code, "-")
	return strings.ToUpper(code)
}

// Search scrapes jable.tv for video results matching the given JAV code.
func (s *Scraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	formatted := s.FormatCode(code)

	// 1. Search page
	searchURL := fmt.Sprintf("%s/search/%s/", baseURL, formatted)
	doc, finalURL, _, err := s.fetchPage(ctx, searchURL)
	if err != nil {
		return errorResult(formatted, fmt.Sprintf("search request: %v", err)), nil
	}

	// 2. 404 detection
	if len(finalURL) > 0 && isNotFound(finalURL, doc) {
		return notFoundResult(formatted, searchURL), nil
	}

	// 3. Parse search results for matching video link
	videoURL := findVideoLink(doc, formatted)

	// Also check if search page IS the video page (site may redirect directly)
	if videoURL == "" && isVideoPage(finalURL) {
		videoURL = finalURL
	}

	if videoURL == "" {
		// Check for cnsub variant in search results
		cnsubURL := findCNSubLink(doc, formatted)
		if cnsubURL == "" {
			return notFoundResult(formatted, searchURL), nil
		}
		// Got a cnsub link but no default — return the cnsub version
		results := []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusSuccess,
			Version:  domain.VersionCNSub,
			Label:    "Chinese Sub",
			PageURL:  cnsubURL,
		}}
		return results, nil
	}

	// 4. Fetch video page and extract player
	videoDoc, _, rawVideoHTML, fetchErr := s.fetchPage(ctx, videoURL)
	if fetchErr != nil {
		return errorResult(formatted, fmt.Sprintf("video page: %v", fetchErr)), nil
	}

	// Priority 1: Search raw HTML for M3U8 URLs (mushroomtrack.com etc)
	playerURL, cnsubPlayerURL := "", ""
	m3u8URLs := scraper.ExtractM3U8FromRawHTML(rawVideoHTML)
	for _, url := range m3u8URLs {
		if !strings.Contains(url, "jable.tv/search") && !strings.Contains(url, "assets-cdn") {
			playerURL = url
			break
		}
	}

	// Priority 2: Fall back to DOM-based extraction
	if playerURL == "" {
		playerURL, cnsubPlayerURL = extractPlayers(videoDoc, rawVideoHTML)
	} else {
		_, cnsubPlayerURL = extractPlayers(videoDoc, rawVideoHTML)
	}

	// 5. Build results — multi-version support
	var results []domain.VideoResult

	// Original version
	origResult := domain.VideoResult{
		SiteName: siteName,
		Status:   domain.StatusSuccess,
		Version:  domain.VersionOriginal,
		PageURL:  videoURL,
	}
	if playerURL != "" {
		origResult.VideoSources = []domain.VideoSource{{
			URL:  playerURL,
			Type: detectVideoType(playerURL),
		}}
	}
	results = append(results, origResult)

	// CNSub version (if detected)
	if cnsubPlayerURL != "" && cnsubPlayerURL != playerURL {
		cnsubResult := domain.VideoResult{
			SiteName: siteName,
			Status:   domain.StatusSuccess,
			Version:  domain.VersionCNSub,
			Label:    "Chinese Sub",
			PageURL:  videoURL,
			VideoSources: []domain.VideoSource{{
				URL:  cnsubPlayerURL,
				Type: detectVideoType(cnsubPlayerURL),
			}},
		}
		results = append(results, cnsubResult)
	}

	return results, nil
}

// fetchPage performs a GET request with jable-specific headers and returns
// the parsed HTML document and the final URL after any redirects.
func (s *Scraper) fetchPage(ctx context.Context, pageURL string) (*goquery.Document, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil, pageURL, "", err
	}

	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, pageURL, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, resp.Request.URL.String(), "", err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, resp.Request.URL.String(), "", err
	}

	return doc, resp.Request.URL.String(), string(body), nil
}

// isNotFound checks whether the response indicates no results were found.
// Detection: final URL differs from expected, or body contains too few bytes (< 5000).
func isNotFound(finalURL string, doc *goquery.Document) bool {
	// Very short responses indicate error/empty results
	html, _ := doc.Html()
	if len(html) < 5000 {
		return true
	}
	// If redirected to home page or different path, treat as not found
	if strings.Contains(finalURL, "/search/") && !strings.Contains(finalURL, "/search/") {
		return false
	}
	return false
}

// isVideoPage returns true when the URL looks like a video detail page.
func isVideoPage(pageURL string) bool {
	// jable.tv video pages are like: https://jable.tv/videos/xxx-xxx/
	return strings.Contains(pageURL, "/videos/")
}

// findVideoLink searches the parsed HTML for a video link matching the
// formatted code. Returns empty string if no match is found.
func findVideoLink(doc *goquery.Document, code string) string {
	var link string
	upperCode := strings.ToUpper(code)

	// Search in video cards — typical structure: .video-item, .card, a[href*="/videos/"]
	doc.Find("a[href*='/videos/']").Each(func(i int, s *goquery.Selection) {
		if link != "" {
			return
		}
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		hrefUpper := strings.ToUpper(href)
		if strings.Contains(hrefUpper, upperCode) {
			// Resolve relative URL
			link = resolveURL(baseURL, href)
		}
	})

	// Also try h6.title a pattern (common Jable layout)
	if link == "" {
		doc.Find("h6.title a[href*='/videos/']").Each(func(i int, s *goquery.Selection) {
			if link != "" {
				return
			}
			href, _ := s.Attr("href")
			hrefUpper := strings.ToUpper(href)
			if strings.Contains(hrefUpper, upperCode) {
				link = resolveURL(baseURL, href)
			}
		})
	}

	// Fallback: .video-img-box a with matching href
	if link == "" {
		doc.Find(".video-img-box a[href*='/videos/']").Each(func(i int, s *goquery.Selection) {
			if link != "" {
				return
			}
			href, _ := s.Attr("href")
			hrefUpper := strings.ToUpper(href)
			if strings.Contains(hrefUpper, upperCode) {
				link = resolveURL(baseURL, href)
			}
		})
	}

	return link
}

// findCNSubLink locates a Chinese-subtitle version of the video from search results.
func findCNSubLink(doc *goquery.Document, code string) string {
	upperCode := strings.ToUpper(code)

	// CN-subbed videos often have "中文" or "字幕" in the title/card
	var link string
	doc.Find("a[href*='/videos/']").Each(func(i int, s *goquery.Selection) {
		if link != "" {
			return
		}
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		hrefUpper := strings.ToUpper(href)
		text := strings.ToUpper(s.Text())
		if strings.Contains(hrefUpper, upperCode) &&
			(strings.Contains(text, "中文") || strings.Contains(text, "字幕")) {
			link = resolveURL(baseURL, href)
		}
	})

	return link
}

// extractPlayers scans the video page for player source URLs.
// Returns (original_player_url, cnsub_player_url).
func extractPlayers(doc *goquery.Document, rawHTML string) (string, string) {
	var playerURL, cnsubURL string

	// Look for video source elements
	doc.Find("video source").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}
		if playerURL == "" && isValidPlayerURL(src) {
			playerURL = src
		}
	})

	// Look for iframe player
	if playerURL == "" {
		doc.Find("iframe[src]").Each(func(i int, s *goquery.Selection) {
			src, _ := s.Attr("src")
			if src == "" || playerURL != "" {
				return
			}
			if isValidPlayerURL(src) {
				playerURL = resolveURL(baseURL, src)
			}
		})
	}

	// Check for data attributes with player URLs
	if playerURL == "" {
		doc.Find("[data-src], [data-video], [data-url]").Each(func(i int, s *goquery.Selection) {
			if playerURL != "" {
				return
			}
			for _, attr := range []string{"data-src", "data-video", "data-url"} {
				if val, exists := s.Attr(attr); exists && val != "" {
					if isValidPlayerURL(val) {
						playerURL = val
					}
					return
				}
			}
		})
	}

	// Look for JP player .h-player-wrapper or similar
	if playerURL == "" {
		doc.Find("#video-player script, .h-player script, script").EachWithBreak(func(i int, s *goquery.Selection) bool {
			script := s.Text()
			for _, pattern := range []string{`source:'`, `source: "`, `src: "`, `src:'`} {
				idx := strings.Index(script, pattern)
				if idx >= 0 {
					start := idx + len(pattern)
					end := strings.IndexAny(script[start:], `"'`)
					if end >= 0 {
						candidate := script[start : start+end]
						if strings.HasPrefix(candidate, "http") && playerURL == "" && isValidPlayerURL(candidate) {
							playerURL = candidate
						}
					}
				}
			}
			return playerURL == "" // continue if not found
		})
	}

	// Raw HTML fallback: search for M3U8 URLs that DOM selectors missed
	if playerURL == "" {
		for _, url := range scraper.ExtractM3U8FromRawHTML(rawHTML) {
			if isValidPlayerURL(url) {
				playerURL = url
				break
			}
		}
	}

	// CNSub detection: .nav-tabs or tab content with "字幕"/"中文" label
	doc.Find(".nav-tabs a, .tab-nav a, .video-version-item").Each(func(i int, s *goquery.Selection) {
		if cnsubURL != "" {
			return
		}
		text := strings.ToUpper(s.Text())
		if strings.Contains(text, "中文") || strings.Contains(text, "字幕") {
			if href, exists := s.Attr("href"); exists && href != "" {
				cnsubURL = resolveURL(baseURL, href)
			}
			if src, exists := s.Attr("data-src"); exists && src != "" {
				cnsubURL = src
			}
		}
	})

	return playerURL, cnsubURL
}

// detectVideoType infers the MIME type from the URL extension.
func detectVideoType(rawURL string) string {
	lower := strings.ToLower(rawURL)
	if strings.HasSuffix(lower, ".m3u8") {
		return "application/x-mpegURL"
	}
	if strings.HasSuffix(lower, ".mp4") {
		return "video/mp4"
	}
	if strings.HasSuffix(lower, ".ts") {
		return "video/mp2t"
	}
	if strings.Contains(lower, "m3u8") {
		return "application/x-mpegURL"
	}
	return "video/mp4"
}

// resolveURL resolves a relative URL against the base URL.
func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "http") {
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

// errorResult builds a single-error VideoResult slice.
func errorResult(code string, msg string) []domain.VideoResult {
	return []domain.VideoResult{{
		SiteName: siteName,
		Status:   domain.StatusError,
		Version:  domain.VersionOriginal,
		PageURL:  fmt.Sprintf("%s/search/%s/", baseURL, code),
		Error:    msg,
	}}
}

// isValidPlayerURL checks whether a raw URL looks like a real video player URL.
func isValidPlayerURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	lower := strings.ToLower(rawURL)
	if strings.HasPrefix(lower, "#") || strings.HasPrefix(lower, "javascript:") {
		return false
	}
	if strings.Contains(lower, "labadena") || strings.Contains(lower, "bluetraffic") ||
		strings.Contains(lower, "smartpop") || strings.Contains(lower, "subid") ||
		strings.Contains(lower, "/api/spots") || strings.Contains(lower, "/api/ads") ||
		strings.Contains(lower, "tracking") || strings.Contains(lower, "analytics") {
		return false
	}
	if strings.Contains(lower, "%query%") || strings.Contains(lower, "/search/") {
		return false
	}
	return true
}

// notFoundResult builds a not-found VideoResult slice.
func notFoundResult(code string, searchURL string) []domain.VideoResult {
	return []domain.VideoResult{{
		SiteName: siteName,
		Status:   domain.StatusNotFound,
		Version:  domain.VersionOriginal,
		PageURL:  searchURL,
	}}
}
