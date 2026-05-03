// Package javgg implements a scraper for javgg.net video search.
//
// javgg.net uses a search-based flow:
//  1. GET https://javgg.net/?s={code} with custom headers
//  2. Parse search results for article .details .title a[href*='/jav/']
//  3. Follow the matched link to the video page
//  4. Extract video sources from the player on the video page
//
// 404 detection uses HTTP status code, search result count, and body content.
package javgg

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

const (
	siteName  = "javgg"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	cfTestURL = "https://javgg.net/"
	baseURL   = "https://javgg.net"
)

var _ domain.Scraper = (*Scraper)(nil)

// Scraper scrapes video sources from javgg.net.
type Scraper struct {
	enabled     bool
	proxyConfig domain.ProxyConfig
	httpClient  *http.Client
	baseURL     string
}

// New creates a javgg scraper with the given proxy configuration.
// It runs a Cloudflare bypass pre-test; if the test fails, the scraper is
// created with IsEnabled() returning false and a warning is logged.
func New(config domain.ProxyConfig) *Scraper {
	s := &Scraper{
		enabled:     true,
		proxyConfig: config,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		baseURL:     baseURL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	proxyURL := ""
	if config.Enabled {
		proxyURL = config.URL
	}

	result, err := scraper.CFBypassTest(ctx, cfTestURL, proxyURL)
	if err != nil || !result.Passed {
		s.enabled = false
		errMsg := "unknown error"
		if result != nil {
			errMsg = result.Error
		}
		log.Printf("javgg: Cloudflare bypass test failed, scraper disabled: %s", errMsg)
	}

	return s
}

func (s *Scraper) Name() string                         { return siteName }
func (s *Scraper) IsEnabled() bool                       { return s.enabled }
func (s *Scraper) RequiresCFBypass() bool                { return true }
func (s *Scraper) GetProxyConfig() domain.ProxyConfig    { return s.proxyConfig }
func (s *Scraper) FormatCode(code string) string         { return code }

// Search scrapes javgg.net for the given code and returns video results.
func (s *Scraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("javgg: scraper is disabled")
	}

	// 1. Search page
	searchURL := fmt.Sprintf("%s/?s=%s", s.baseURL, code)
	searchDoc, err := s.fetchSearchPage(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("javgg: search request: %w", err)
	}

	// 2. Find matching video link in search results
	normalizedCode := scraper.NormalizeCode(code)
	videoURL := findVideoLink(searchDoc, normalizedCode)
	if videoURL == "" {
		return notFoundResult(code, searchURL), nil
	}

	// 3. Fetch video page and extract sources
	videoDoc, err := s.fetchVideoPage(ctx, videoURL)
	if err != nil {
		return errorResult(code, videoURL, fmt.Sprintf("video page: %v", err)), nil
	}

	// 4. Extract video sources
	sources := extractVideoSources(videoDoc)

	if len(sources) == 0 {
		return notFoundResult(code, searchURL), nil
	}

	return []domain.VideoResult{{
		SiteName:     siteName,
		Status:       domain.StatusSuccess,
		Version:      domain.VersionOriginal,
		PageURL:      videoURL,
		VideoSources: sources,
		Subtitle:     false,
		Leak:         false,
	}}, nil
}

func (s *Scraper) fetchSearchPage(ctx context.Context, pageURL string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Referer", baseURL+"/")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("HTTP 404")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func (s *Scraper) fetchVideoPage(ctx context.Context, pageURL string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Referer", baseURL+"/")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("HTTP 404")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

// findVideoLink searches the search results page for a video link matching
// the given normalized code. It looks for article .details .title a[href*='/jav/'].
func findVideoLink(doc *goquery.Document, normalizedCode string) string {
	var link string

	doc.Find("article").Each(func(_ int, article *goquery.Selection) {
		if link != "" {
			return
		}
		article.Find(".details .title a[href*='/jav/']").Each(func(_ int, a *goquery.Selection) {
			if link != "" {
				return
			}
			href, exists := a.Attr("href")
			if !exists {
				return
			}
			hrefNormalized := scraper.NormalizeCode(href)
			text := scraper.NormalizeCode(a.Text())
			if strings.Contains(hrefNormalized, normalizedCode) || strings.Contains(text, normalizedCode) {
				link = href
			}
		})
	})

	// Fallback: search all a[href*='/jav/'] outside article structure
	if link == "" {
		doc.Find("a[href*='/jav/']").Each(func(_ int, a *goquery.Selection) {
			if link != "" {
				return
			}
			href, exists := a.Attr("href")
			if !exists {
				return
			}
			hrefNormalized := scraper.NormalizeCode(href)
			text := scraper.NormalizeCode(a.Text())
			if strings.Contains(hrefNormalized, normalizedCode) || strings.Contains(text, normalizedCode) {
				link = href
			}
		})
	}

	return link
}

// extractVideoSources extracts playable video URLs from the video page.
func extractVideoSources(doc *goquery.Document) []domain.VideoSource {
	var sources []domain.VideoSource
	seen := make(map[string]bool)

	// video source elements
	doc.Find("video source").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" || seen[src] {
			return
		}
		seen[src] = true
		mimeType, _ := s.Attr("type")
		if mimeType == "" {
			mimeType = "video/mp4"
		}
		sources = append(sources, domain.VideoSource{
			URL:  src,
			Type: mimeType,
		})
	})

	// video[src] elements
	doc.Find("video[src]").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" || seen[src] {
			return
		}
		seen[src] = true
		sources = append(sources, domain.VideoSource{
			URL:  src,
			Type: "video/mp4",
		})
	})

	// iframe[src] elements
	doc.Find("iframe[src]").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" || seen[src] {
			return
		}
		seen[src] = true
		sources = append(sources, domain.VideoSource{
			URL:  src,
			Type: "text/html",
		})
	})

	return sources
}

func notFoundResult(code string, pageURL string) []domain.VideoResult {
	return []domain.VideoResult{{
		SiteName: siteName,
		Status:   domain.StatusNotFound,
		Version:  domain.VersionOriginal,
		PageURL:  pageURL,
	}}
}

func errorResult(code string, pageURL string, msg string) []domain.VideoResult {
	return []domain.VideoResult{{
		SiteName: siteName,
		Status:   domain.StatusError,
		Version:  domain.VersionOriginal,
		PageURL:  pageURL,
		Error:    msg,
	}}
}

func init() {
	scraper.Register(New(domain.ProxyConfig{}))
}
