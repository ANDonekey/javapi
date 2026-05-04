package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/RomainMichau/CycleTLS/cycletls"
	"github.com/RomainMichau/cloudscraper_go/cloudscraper"
	"github.com/henry/javapi/internal/domain"
)

// ProxyFromEnv reads SCRAPER_PROXY_URL from the environment.
func ProxyFromEnv() domain.ProxyConfig {
	url := os.Getenv("SCRAPER_PROXY_URL")
	if url != "" {
		return domain.ProxyConfig{URL: url, Enabled: true}
	}
	return domain.ProxyConfig{}
}

const FirefoxUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0"

type CFTestResult struct {
	URL    string `json:"url"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

var cfBlockMarkers = []string{
	"Just a moment",
	"Checking your browser",
	"cf-browser-verification",
	"Attention Required!",
}

func getScraper() (*cloudscraper.CloudScrapper, error) {
	return cloudscraper.Init(false, false)
}

func CFBypassTest(ctx context.Context, url string, proxyURL string) (*CFTestResult, error) {
	if ctx.Err() != nil {
		return &CFTestResult{URL: url, Passed: false, Error: ctx.Err().Error()}, nil
	}

	forceHTTP1 := false
	var lastErr error
	for i := 1; i <= 3; i++ {
		scraper, err := getScraper()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i) * time.Second)
			continue
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
			ForceHTTP1: forceHTTP1,
		}

		if proxyURL != "" {
			options.Proxy = proxyURL
		}

		res, err := scraper.Do(url, options, "GET")
		if err != nil {
			if strings.Contains(err.Error(), "421") {
				forceHTTP1 = true
			}
			lastErr = err
			time.Sleep(time.Duration(i) * time.Second)
			continue
		}

		if res.Status == 421 {
			forceHTTP1 = true
			time.Sleep(30 * time.Second)
			continue
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

		return &CFTestResult{URL: url, Passed: true}, nil
	}

	errMsg := "retries exhausted"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	return &CFTestResult{URL: url, Passed: false, Error: errMsg}, nil
}

func NewCFClient(proxyURL string) *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &cfRoundTripper{
			proxyURL:  proxyURL,
			userAgent: FirefoxUA,
		},
	}
}

type cfRoundTripper struct {
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

	forceHTTP1 := false
	var lastErr error
	for i := 1; i <= 3; i++ {
		scraper, err := getScraper()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i) * time.Second)
			continue
		}

		options := cycletls.Options{
			UserAgent:  rt.userAgent,
			Timeout:    15,
			Headers:    headers,
			ForceHTTP1: forceHTTP1,
		}

		if rt.proxyURL != "" {
			options.Proxy = rt.proxyURL
		}

		res, err := scraper.Do(req.URL.String(), options, req.Method)
		if err != nil {
			lastErr = err
			if strings.Contains(err.Error(), "421") {
				forceHTTP1 = true
			}
			time.Sleep(time.Duration(i) * time.Second)
			continue
		}

		if res.Status == 421 {
			forceHTTP1 = true
			time.Sleep(30 * time.Second)
			continue
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

	if lastErr != nil {
		return nil, lastErr
	}
	log.Printf("cf: retries exhausted for %s without specific error", req.URL.String())
	return nil, fmt.Errorf("cf: retries exhausted for %s", req.URL.String())
}
