package embed

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/henry/javapi/internal/scraper"
)

type javvidsExtractor struct{}

func init() { Register(&javvidsExtractor{}) }

func (e *javvidsExtractor) MatchHost(host string) bool {
	return host == "jav-vids.xyz"
}

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

	if urls := scraper.ExtractM3U8FromRawHTML(html); len(urls) > 0 {
		return urls, nil
	}

	for _, line := range strings.Split(html, "\n") {
		for _, key := range []string{`"file":"`, `"src":"`, `"source":"`, `file:"`, `src:`} {
			if idx := strings.Index(line, key); idx >= 0 {
				start := idx + len(key)
				end := strings.IndexAny(line[start:], `"'`)
				if end > 0 {
					candidate := line[start : start+end]
					if strings.Contains(candidate, ".m3u8") {
						return []string{candidate}, nil
					}
				}
			}
		}
	}
	return nil, nil
}
