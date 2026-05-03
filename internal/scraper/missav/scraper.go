// Package missav implements a scraper for missav.ws (MISSAV).
// MISSAV uses raw video codes (no formatting) and is often behind Cloudflare.
package missav

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
	siteName  = "missav"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0"
	cfTestURL = "https://missav.ws/"
)

var _ domain.Scraper = (*MISSAVScraper)(nil)

// MISSAVScraper scrapes video sources from missav.ws.
type MISSAVScraper struct {
	enabled     bool
	proxyConfig domain.ProxyConfig
	httpClient  *http.Client
	baseURL     string
}

// NewMISSAVScraper creates a MISSAV scraper with the given proxy configuration.
// It runs a Cloudflare bypass pre-test; if the test fails, the scraper is
// created with IsEnabled() returning false and a warning is logged.
func NewMISSAVScraper(config domain.ProxyConfig) *MISSAVScraper {
	proxyURL := ""
	if config.Enabled {
		proxyURL = config.URL
	}

	s := &MISSAVScraper{
		enabled:     true,
		proxyConfig: config,
		httpClient:  scraper.NewCFClient(proxyURL),
		baseURL:     "https://missav.ws",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	result, err := scraper.CFBypassTest(ctx, cfTestURL, proxyURL)
	if err != nil || !result.Passed {
		s.enabled = false
		errMsg := "unknown error"
		if result != nil {
			errMsg = result.Error
		}
		log.Printf("missav: Cloudflare bypass test failed, scraper disabled: %s", errMsg)
	}

	return s
}

func (s *MISSAVScraper) Name() string             { return siteName }
func (s *MISSAVScraper) IsEnabled() bool           { return s.enabled }
func (s *MISSAVScraper) RequiresCFBypass() bool    { return true }
func (s *MISSAVScraper) GetProxyConfig() domain.ProxyConfig { return s.proxyConfig }
func (s *MISSAVScraper) FormatCode(code string) string      { return code }

// Search scrapes missav.ws for the given code and returns video results.
// It detects subtitle (cnsub) and mosaic_reduce versions via CSS selectors.
func (s *MISSAVScraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("missav: scraper is disabled")
	}

	pageURL := fmt.Sprintf("%s/%s/", s.baseURL, code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("missav: create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("missav: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			Version:  domain.VersionOriginal,
			PageURL:  pageURL,
		}}, nil
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("missav: HTTP %d: %s", resp.StatusCode, string(body))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("missav: parse HTML: %w", err)
	}

	title := strings.ToLower(doc.Find("title").First().Text())
	bodyText := strings.ToLower(doc.Find("body").Text())
	if strings.Contains(title, "not found") || strings.Contains(bodyText, "not found") {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			Version:  domain.VersionOriginal,
			PageURL:  pageURL,
		}}, nil
	}

	sources := extractVideoSources(doc)

	normalizedCode := scraper.NormalizeCode(code)
	titleOk := verifyTitle(doc, normalizedCode)

	if !titleOk && len(sources) == 0 {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			Version:  domain.VersionOriginal,
			PageURL:  pageURL,
			Error:    "title does not match code and no video sources found",
		}}, nil
	}

	hasSub := doc.Find(".space-y-2 a.text-nord13[href*='chinese-subtitle']").Length() > 0
	hasLeak := doc.Find(".order-first div.rounded-md a[href]").Last().Length() > 0

	var results []domain.VideoResult

	original := domain.VideoResult{
		SiteName:     siteName,
		Status:       domain.StatusSuccess,
		Version:      domain.VersionOriginal,
		PageURL:      pageURL,
		VideoSources: sources,
		Subtitle:     false,
		Leak:         false,
	}
	if !titleOk {
		original.Status = domain.StatusError
		original.Error = "page title does not match video code"
	}
	results = append(results, original)

	if hasSub {
		results = append(results, domain.VideoResult{
			SiteName:     siteName,
			Status:       domain.StatusSuccess,
			Version:      domain.VersionCNSub,
			PageURL:      pageURL,
			VideoSources: sources,
			Subtitle:     true,
			Leak:         false,
		})
	}

	if hasLeak {
		results = append(results, domain.VideoResult{
			SiteName:     siteName,
			Status:       domain.StatusSuccess,
			Version:      domain.VersionMosaicReduce,
			PageURL:      pageURL,
			VideoSources: sources,
			Subtitle:     false,
			Leak:         true,
		})
	}

	return results, nil
}

func extractVideoSources(doc *goquery.Document) []domain.VideoSource {
	var sources []domain.VideoSource
	seen := make(map[string]bool)

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

func verifyTitle(doc *goquery.Document, normalizedCode string) bool {
	title := scraper.NormalizeCode(doc.Find("title").First().Text())
	if strings.Contains(title, normalizedCode) {
		return true
	}

	ogTitle, exists := doc.Find("meta[property='og:title']").First().Attr("content")
	if exists && strings.Contains(scraper.NormalizeCode(ogTitle), normalizedCode) {
		return true
	}

	foundInH1 := false
	doc.Find("h1").Each(func(_ int, s *goquery.Selection) {
		if strings.Contains(scraper.NormalizeCode(s.Text()), normalizedCode) {
			foundInH1 = true
		}
	})
	return foundInH1
}

func init() {
	scraper.Register(NewMISSAVScraper(domain.ProxyConfig{}))
}
