// Package av01 implements a domain.Scraper for https://www.av01.media.
//
// Unlike other scrapers that parse HTML, AV01 uses a POST JSON API for search.
// The scraper sends a JSON search request, parses the JSON response, and matches
// results by dvd_id or dmm_id against the target code.
//
// Search endpoint:
//
//	POST https://www.av01.media/api/v1/videos/search?lang=ja
//	Body: {"query":"CODE","pagination":{"page":1,"limit":24}}
//
// Result URL template:
//
//	https://www.av01.media/jp/video/{id}/{code}
package av01

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

const (
	siteName      = "av01"
	baseURL       = "https://www.av01.media"
	searchPath    = "/api/v1/videos/search?lang=ja"
	videoPathTmpl = "/jp/video/%s/%s" // {id}/{code}
	userAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	cfTestURL     = "https://www.av01.media/"
)

// searchRequest is the JSON body sent to the AV01 search API.
type searchRequest struct {
	Query      string           `json:"query"`
	Pagination searchPagination `json:"pagination"`
}

// searchPagination holds pagination parameters for the search API.
type searchPagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// searchResponse is the JSON response from the AV01 search API.
// Only the fields needed for code matching are extracted; the rest are ignored.
type searchResponse struct {
	Videos []videoEntry `json:"videos"`
}

// videoEntry represents a single video result from the AV01 search API.
type videoEntry struct {
	ID    string `json:"id"`
	DVDID string `json:"dvd_id"`
	DMMID string `json:"dmm_id"`
}

// Scraper implements domain.Scraper for av01.media.
// It is safe for concurrent use — all mutable fields are set at construction time.
type Scraper struct {
	client      *http.Client
	enabled     bool
	proxyConfig domain.ProxyConfig
	baseURL     string
}

// Compile-time interface check.
var _ domain.Scraper = (*Scraper)(nil)

func init() {
	scraper.Register(New())
}

// New creates an AV01 scraper with sensible defaults (15s timeout, standard UA).
// It runs a Cloudflare bypass pre-test; if the test fails, the scraper is
// created with IsEnabled() returning false and a warning is logged.
func New() *Scraper {
	s := &Scraper{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		enabled: true,
		proxyConfig: domain.ProxyConfig{
			Enabled: false,
		},
		baseURL: baseURL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	proxyURL := ""
	if s.proxyConfig.Enabled {
		proxyURL = s.proxyConfig.URL
	}

	result, err := scraper.CFBypassTest(ctx, cfTestURL, proxyURL)
	if err != nil || !result.Passed {
		s.enabled = false
		errMsg := "unknown error"
		if result != nil {
			errMsg = result.Error
		}
		log.Printf("av01: Cloudflare bypass test failed, scraper disabled: %s", errMsg)
	}

	return s
}

// NewWithClient returns a Scraper that uses a custom http.Client and base URL.
// Primarily used in tests to inject httptest servers.
func NewWithClient(client *http.Client, baseURL string) *Scraper {
	return &Scraper{
		client:      client,
		enabled:     true,
		proxyConfig: domain.ProxyConfig{Enabled: false},
		baseURL:     baseURL,
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

// Search queries the AV01 JSON API and returns video results matching the code.
//
// Flow:
//  1. Construct JSON POST body with the search code
//  2. POST to the search endpoint with JSON headers
//  3. Parse the JSON response
//  4. Match entries by dvd_id or dmm_id against the normalized search code
//  5. Return VideoResult with page URL, or StatusNotFound if no match
func (s *Scraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("av01: scraper is disabled")
	}

	searchURL := s.baseURL + searchPath
	normalized := scraper.NormalizeCode(code)

	// Build request body
	reqBody := searchRequest{
		Query: code,
		Pagination: searchPagination{
			Page:  1,
			Limit: 24,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("av01: marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("av01: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

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

	// Check for Cloudflare block in the response body
	rawBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if readErr != nil {
		return nil, fmt.Errorf("av01: read response body: %w", readErr)
	}

	var sr searchResponse
	if err := json.Unmarshal(rawBody, &sr); err != nil {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusError,
			Version:  domain.VersionOriginal,
			PageURL:  searchURL,
			Error:    fmt.Sprintf("invalid response JSON: %v", err),
		}}, nil
	}

	// Search for a matching entry by dvd_id or dmm_id
	var matched *videoEntry
	for i := range sr.Videos {
		entry := &sr.Videos[i]
		if entry.DVDID != "" && scraper.NormalizeCode(entry.DVDID) == normalized {
			matched = entry
			break
		}
		if entry.DMMID != "" && scraper.NormalizeCode(entry.DMMID) == normalized {
			matched = entry
			break
		}
	}

	if matched == nil {
		return []domain.VideoResult{{
			SiteName: siteName,
			Status:   domain.StatusNotFound,
			Version:  domain.VersionOriginal,
			PageURL:  searchURL,
		}}, nil
	}

	pageURL := fmt.Sprintf(s.baseURL+videoPathTmpl, matched.ID, code)

	return []domain.VideoResult{{
		SiteName: siteName,
		Status:   domain.StatusSuccess,
		Version:  domain.VersionOriginal,
		PageURL:  pageURL,
	}}, nil
}
