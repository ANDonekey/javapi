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

	allURLs := scraper.ExtractM3U8FromRawHTML(html)
	var streamURLs, otherURLs []string
	for _, u := range allURLs {
		if strings.Contains(u, "jav-vids.xyz/stream/") {
			streamURLs = append(streamURLs, u)
		} else {
			otherURLs = append(otherURLs, u)
		}
	}
	if len(streamURLs) > 0 {
		return streamURLs, nil
	}
	if len(otherURLs) > 0 {
		return otherURLs, nil
	}

	for _, line := range strings.Split(html, "\n") {
		for _, key := range []string{`"file":"`, `"src":"`, `"source":"`, `file:"`, `src:`} {
			if idx := strings.Index(line, key); idx >= 0 {
				start := idx + len(key)
				end := strings.IndexAny(line[start:], `"'`)
				if end > 0 {
					candidate := line[start : start+end]
					if strings.Contains(candidate, ".m3u8") && strings.Contains(candidate, "jav-vids.xyz/stream/") {
						return []string{candidate}, nil
					}
				}
			}
		}
	}
	for _, unpacked := range scraper.UnpackJavascriptStrings(html) {
		start := 0
		for {
			idx := strings.Index(unpacked[start:], "/stream/")
			if idx < 0 {
				break
			}
			absIdx := start + idx
			end := strings.Index(unpacked[absIdx:], ".m3u8")
			if end > 0 {
				relPath := unpacked[absIdx : absIdx+end+5]
				relPath = strings.ReplaceAll(relPath, `\/`, `/`)
				fullURL := "https://jav-vids.xyz" + relPath
				return []string{fullURL}, nil
			}
			start = absIdx + 8
		}
	}
	return nil, nil
}
