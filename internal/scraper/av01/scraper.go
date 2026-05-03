// Package av01 implements a domain.Scraper for https://www.av01.media.
//
// Search flow:
//  1. POST /api/v1/videos/search?lang=cn&comp=true with JSON body
//     {"pagination":{"limit":20,"page":1},"query":"CODE"}
//  2. Parse JSON response → match by dvd_id or dmm_id (normalized)
//  3. Build M3U8 URL: /api/v1/videos/{id}/manifest/master.m3u8
package av01

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

const (
	siteName  = "av01"
	baseURL   = "https://www.av01.media"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	cfTestURL = "https://www.av01.media/"
)

// searchRequest is the JSON body for the search API.
type searchRequest struct {
	Pagination searchPagination `json:"pagination"`
	Query      string           `json:"query"`
}

type searchPagination struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
}

// searchVideo mirrors a single video entry in the API response.
type searchVideo struct {
	ID    int    `json:"id"`
	DvdID string `json:"dvd_id"`
	DmmID string `json:"dmm_id"`
}

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

// Search performs a JSON API search on AV01 and returns video results.
//
// Flow:
//  1. POST JSON to /api/v1/videos/search?lang=cn&comp=true
//  2. Parse JSON response → match by normalized dvd_id or dmm_id
//  3. Construct M3U8 URL from matched video ID
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

	searchURL := s.baseURL + "/api/v1/videos/search?lang=cn&comp=true"

	body := searchRequest{
		Pagination: searchPagination{Limit: 20, Page: 1},
		Query:      code,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("av01: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("av01: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

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

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("av01: read response: %w", err)
	}

	var sr struct {
		Videos []searchVideo `json:"videos"`
	}
	if err := json.Unmarshal(respBytes, &sr); err != nil {
		return nil, fmt.Errorf("av01: parse JSON: %w", err)
	}

	normalized := scraper.NormalizeCode(code)
	var matchedID int

	for _, v := range sr.Videos {
		if v.DvdID != "" && scraper.NormalizeCode(v.DvdID) == normalized {
			matchedID = v.ID
			break
		}
		if v.DmmID != "" && scraper.NormalizeCode(v.DmmID) == normalized {
			matchedID = v.ID
			break
		}
	}

	if matchedID == 0 {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			Version:  domain.VersionOriginal,
			PageURL:  searchURL,
		}}, nil
	}

	codeLower := strings.ToLower(code)
	m3u8URL := fmt.Sprintf("%s/api/v1/videos/%d/manifest/master.m3u8", s.baseURL, matchedID)

	return []domain.VideoResult{{
		SiteName: siteName,
		Status:   domain.StatusSuccess,
		Version:  domain.VersionOriginal,
		PageURL:  fmt.Sprintf("%s/cn/video/%d/%s", s.baseURL, matchedID, codeLower),
		VideoSources: []domain.VideoSource{
			{URL: m3u8URL, Type: "application/x-mpegURL"},
		},
	}}, nil
}
