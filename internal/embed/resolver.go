package embed

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

type Extractor interface {
	MatchHost(host string) bool
	Extract(ctx context.Context, client *http.Client, pageURL string) ([]string, error)
}

var extractors []Extractor

func Register(e Extractor) { extractors = append(extractors, e) }

func ResolveEmbed(ctx context.Context, src domain.VideoSource) domain.VideoSource {
	if src.Type != "text/html" && src.Type != "" {
		return src
	}
	u, err := url.Parse(src.URL)
	if err != nil {
		return src
	}
	host := strings.ToLower(u.Hostname())
	for _, e := range extractors {
		if !e.MatchHost(host) {
			continue
		}
		var client *http.Client
		if proxyURL := os.Getenv("EMBED_PROXY_URL"); proxyURL != "" {
			client = scraper.NewCFClient(proxyURL)
		} else if proxyURL := os.Getenv("SCRAPER_MISSAV_PROXY_URL"); proxyURL != "" {
			client = scraper.NewCFClient(proxyURL)
		} else {
			client = &http.Client{Timeout: 15 * time.Second}
		}
		urls, err := e.Extract(ctx, client, src.URL)
		if err == nil && len(urls) > 0 {
			return domain.VideoSource{URL: urls[0], Type: "application/x-mpegURL"}
		}
	}
	return src
}


