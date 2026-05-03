package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/RomainMichau/CycleTLS/cycletls"
	"github.com/RomainMichau/cloudscraper_go/cloudscraper"
)

// FirefoxUA is a browser-like Firefox user agent matching CycleTLS TLS fingerprint.
const FirefoxUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0"

// CFTestResult holds the result of a Cloudflare bypass test.
type CFTestResult struct {
	URL    string `json:"url"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// cfBlockMarkers are strings that appear in Cloudflare challenge pages
// when a request has been blocked.
var cfBlockMarkers = []string{
	"Just a moment",
	"Checking your browser",
	"cf-browser-verification",
	"Attention Required!",
}

var (
	scraperInstance *cloudscraper.CloudScrapper
	scraperErr      error
	scraperOnce     sync.Once
)

func getScraper() (*cloudscraper.CloudScrapper, error) {
	scraperOnce.Do(func() {
		scraperInstance, scraperErr = cloudscraper.Init(false, false)
	})
	return scraperInstance, scraperErr
}

// CFBypassTest tests if Cloudflare can be bypassed on a URL using cloudscraper_go + CycleTLS.
func CFBypassTest(ctx context.Context, url string, proxyURL string) (*CFTestResult, error) {
	if ctx.Err() != nil {
		return &CFTestResult{URL: url, Passed: false, Error: ctx.Err().Error()}, nil
	}

	scraper, err := getScraper()
	if err != nil {
		return &CFTestResult{URL: url, Passed: false, Error: "cloudscraper init failed: " + err.Error()}, nil
	}

	options := cycletls.Options{
		UserAgent: FirefoxUA,
		Timeout:   15,
		Headers: map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
			"Sec-Fetch-User":            "?1",
			"Priority":                  "u=0, i",
		},
	}

	if proxyURL != "" {
		options.Proxy = proxyURL
	}

	res, err := scraper.Do(url, options, "GET")
	if err != nil {
		if strings.Contains(err.Error(), "421") {
			options.ForceHTTP1 = true
			res, err = scraper.Do(url, options, "GET")
		}
		if err != nil {
			return &CFTestResult{URL: url, Passed: false, Error: err.Error()}, nil
		}
	}

	if res.Status >= 400 && res.Status != 421 {
		return &CFTestResult{URL: url, Passed: false, Error: fmt.Sprintf("HTTP %d", res.Status)}, nil
	}

	bodyStr := res.Body
	for _, marker := range cfBlockMarkers {
		if strings.Contains(bodyStr, marker) {
			return &CFTestResult{URL: url, Passed: false, Error: "Cloudflare challenge detected"}, nil
		}
	}

	if hasRealPageContent(bodyStr) {
		return &CFTestResult{URL: url, Passed: true}, nil
	}

	return &CFTestResult{URL: url, Passed: true}, nil
}

func hasRealPageContent(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<body") ||
		strings.Contains(lower, "<title")
}

// NewCFClient creates an http.Client that uses cloudscraper_go for Cloudflare bypass.
// This should be used in scrapers instead of raw net/http client.
func NewCFClient(proxyURL string) *http.Client {
	scraper, err := getScraper()
	if err != nil {
		log.Printf("cloudscraper init failed: %v, falling back to standard client", err)
		return &http.Client{Timeout: 15 * time.Second}
	}

	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &cfRoundTripper{
			scraper:   scraper,
			proxyURL:  proxyURL,
			userAgent: FirefoxUA,
		},
	}
}

// cfRoundTripper implements http.RoundTripper using cloudscraper_go.
type cfRoundTripper struct {
	scraper   *cloudscraper.CloudScrapper
	proxyURL  string
	userAgent string
}

func (rt *cfRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	headers := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	options := cycletls.Options{
		UserAgent: rt.userAgent,
		Timeout:   15,
		Headers:   headers,
	}

	if rt.proxyURL != "" {
		options.Proxy = rt.proxyURL
	}

	res, err := rt.scraper.Do(req.URL.String(), options, req.Method)
	if err != nil {
		if strings.Contains(err.Error(), "421") {
			options.ForceHTTP1 = true
			res, err = rt.scraper.Do(req.URL.String(), options, req.Method)
		}
		if err != nil {
			return nil, err
		}
	}

	httpResp := &http.Response{
		Status:        http.StatusText(res.Status),
		StatusCode:    res.Status,
		Body:          io.NopCloser(strings.NewReader(res.Body)),
		ContentLength: int64(len(res.Body)),
		Request:       req,
	}

	return httpResp, nil
}
