package embed

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/henry/javapi/internal/scraper"
)

type javvidsExtractor struct{}

func init() { Register(&javvidsExtractor{}) }

func (e *javvidsExtractor) MatchHost(host string) bool {
	return host == "jav-vids.xyz"
}

var m3u8RE = regexp.MustCompile(`https?://[^\s'"<>]+\.(?:m3u8|txt)[^\s'"<>]*`)

func (e *javvidsExtractor) Extract(ctx context.Context, client *http.Client, pageURL string) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html,*/*")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	html := string(body)

	var allURLs []string

	allURLs = append(allURLs, scraper.ExtractM3U8FromRawHTML(html)...)

	for _, unpacked := range scraper.UnpackJavascriptStrings(html) {
		allURLs = append(allURLs, m3u8RE.FindAllString(unpacked, -1)...)
	}

	seen := make(map[string]bool)
	var result []string
	for _, u := range allURLs {
		if !seen[u] {
			seen[u] = true
			result = append(result, u)
		}
	}

	for _, u := range result {
		if strings.Contains(u, "jav-vids.xyz/stream/") || strings.Contains(u, "/stream/") {
			return []string{u}, nil
		}
	}

	if len(result) > 0 {
		return result, nil
	}
	return nil, nil
}
