package embed

import (
	"context"
	"io"
	"net/http"
	"net/url"
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

	for _, unpacked := range scraper.UnpackJavascriptStrings(html) {
		for _, key := range []string{`"hls4":"/stream/`, `"hls3":"/stream/`, `"/stream/`} {
			idx := strings.Index(unpacked, key)
			if idx < 0 {
				continue
			}
			start := idx
			if key == `"/stream/` {
				start = idx + 1
			} else {
				start = idx + len(key) - len(`/stream/`)
			}
			remaining := unpacked[start:]
			end := strings.IndexAny(remaining, `"'`)
			if end > 0 {
				relPath := remaining[:end]
				relPath = strings.ReplaceAll(relPath, `\/`, `/`)
				if strings.Contains(relPath, "master.m3u8") || strings.Contains(relPath, "index") {
					base, _ := url.Parse(pageURL)
					ref, _ := url.Parse(relPath)
					if base != nil && ref != nil {
						fullURL := base.ResolveReference(ref).String()
						return []string{fullURL}, nil
					}
				}
			}
		}
	}

	allURLs := scraper.ExtractM3U8FromRawHTML(html)
	for _, u := range allURLs {
		if strings.Contains(u, "jav-vids.xyz/stream/") || strings.Contains(u, "/stream/") {
			return []string{u}, nil
		}
	}
	if len(allURLs) > 0 {
		return allURLs, nil
	}
	return nil, nil
}
