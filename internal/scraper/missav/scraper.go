// Package missav implements a scraper for missav.ws (MISSAV).
// MISSAV uses raw video codes (no formatting) and is often behind Cloudflare.
package missav

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

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
	cfTested    bool
	cfPassed    bool
}

// NewMISSAVScraper creates a MISSAV scraper with the given proxy configuration.
// The Cloudflare bypass test is deferred to the first Search() call so that
// proxy configuration (set via ApplyConfig after construction) is respected.
func NewMISSAVScraper(config domain.ProxyConfig) *MISSAVScraper {
	proxyURL := ""
	if config.Enabled {
		proxyURL = config.URL
	}

	return &MISSAVScraper{
		enabled:     true,
		proxyConfig: config,
		httpClient:  scraper.NewCFClient(proxyURL),
		baseURL:     "https://missav.ws",
	}
}

func (s *MISSAVScraper) Name() string             { return siteName }
func (s *MISSAVScraper) IsEnabled() bool           { return s.enabled }
func (s *MISSAVScraper) RequiresCFBypass() bool    { return true }
func (s *MISSAVScraper) GetProxyConfig() domain.ProxyConfig { return s.proxyConfig }
func (s *MISSAVScraper) FormatCode(code string) string      { return code }

// SetProxyConfig updates the proxy configuration and recreates the CF client.
func (s *MISSAVScraper) SetProxyConfig(pc domain.ProxyConfig) {
	s.proxyConfig = pc
	proxyURL := ""
	if pc.Enabled {
		proxyURL = pc.URL
	}
	s.httpClient = scraper.NewCFClient(proxyURL)
}

// Search scrapes missav.ws for the given code and returns video results.
// It detects subtitle (cnsub) and mosaic_reduce versions via CSS selectors.
func (s *MISSAVScraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("missav: scraper is disabled")
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
			log.Printf("missav: Cloudflare bypass test failed, scraper disabled: %s", errMsg)
			return []domain.VideoResult{{
				SiteName: siteName,
				Status:   domain.StatusBlocked,
				Error:    "CF bypass failed: " + errMsg,
			}}, nil
		}
		s.cfPassed = true
	}
	if !s.cfPassed {
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

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("missav: read body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
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

	if strings.Contains(bodyText, "大量垃圾") || strings.Contains(bodyText, "massive garbage") {
		return []domain.VideoResult{{
			SiteName: siteName, Status: domain.StatusNotFound,
			Version: domain.VersionOriginal, PageURL: pageURL,
		}}, nil
	}

	sources := extractVideoSources(doc)

	seen := make(map[string]bool)
	for _, s := range sources {
		seen[s.URL] = true
	}
	for _, url := range scraper.ExtractM3U8FromRawHTML(string(bodyBytes)) {
		if !seen[url] {
			seen[url] = true
			sources = append(sources, domain.VideoSource{URL: url, Type: "application/x-mpegURL"})
		}
	}

	normalizedCode := scraper.NormalizeCode(code)
	titleOk := verifyTitle(doc, normalizedCode)

	if !titleOk && len(sources) == 0 {
		return []domain.VideoResult{{
			SiteName: siteName, Status: domain.StatusNotFound,
			Version: domain.VersionOriginal, PageURL: pageURL,
			Error: "title does not match video code",
		}}, nil
	}

	return []domain.VideoResult{{
		SiteName:     siteName,
		Status:       domain.StatusSuccess,
		Version:      domain.VersionOriginal,
		PageURL:      pageURL,
		VideoSources: sources,
		Subtitle:     false,
		Leak:         false,
	}}, nil
}

func extractVideoSources(doc *goquery.Document) []domain.VideoSource {
	var sources []domain.VideoSource
	seen := make(map[string]bool)

	doc.Find("video source").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" || seen[src] || !isValidVideoURL(src) {
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
		if !exists || src == "" || seen[src] || !isValidVideoURL(src) {
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
		if !isVideoDomain(src) {
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

func isVideoDomain(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, s := range []string{"bluetraffic", "smartpop", "labadena", "subid", "tracking", "adserver", "popup", "ads."} {
		if strings.Contains(host, s) {
			return false
		}
	}
	for _, s := range []string{"missav", "asg.to", "avmeet", "avplayer", "player", "m3u8", "embed", "stream", "cdn", "surrit"} {
		if strings.Contains(host, s) {
			return true
		}
	}
	return false
}

func isValidVideoURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	lower := strings.ToLower(rawURL)
	if strings.Contains(lower, ".m3u8") || strings.Contains(lower, ".mp4") || strings.Contains(lower, ".ts") ||
		strings.Contains(lower, "/hls/") || strings.Contains(lower, "surrit.com") || strings.Contains(lower, "cdn.") {
		return true
	}
	if strings.Contains(lower, "bluetraffic") || strings.Contains(lower, "smartpop") ||
		strings.Contains(lower, "labadena") || strings.Contains(lower, "subid") {
		return false
	}
	return false
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
