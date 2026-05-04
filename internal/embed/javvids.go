package embed

import (
	"context"
	"fmt"
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

	for _, u := range allURLs {
		if strings.Contains(u, "jav-vids.xyz/stream/") {
			return []string{u}, nil
		}
	}

	fileID := ""
	for _, u := range allURLs {
		if idx := strings.Index(u, "&f="); idx >= 0 {
			fileID = u[idx+3:]
			if end := strings.IndexAny(fileID, "& "); end > 0 {
				fileID = fileID[:end]
			}
			break
		}
	}

	dictStart := strings.LastIndex(html, ".split('|')")
	if dictStart >= 0 {
		quoteStart := strings.LastIndex(html[:dictStart], "'")
		if quoteStart >= 0 {
			dictStr := html[quoteStart+1 : dictStart]
			words := strings.Split(dictStr, "|")

			var streamID, token, timestamp string
			for _, w := range words {
				if len(w) >= 20 && hasUpperAndLower(w) {
					streamID = w
				} else if len(w) == 15 && isAlpha(w) {
					token = w
				} else if len(w) == 10 && isNumeric(w) && strings.HasPrefix(w, "177") {
					timestamp = w
				}
			}

			if streamID != "" && fileID != "" && token != "" && timestamp != "" {
				streamURL := fmt.Sprintf("https://jav-vids.xyz/stream/%s/%s/%s/%s/master.m3u8",
					streamID, token, timestamp, fileID)
				return []string{streamURL}, nil
			}
		}
	}

	if len(allURLs) > 0 {
		return allURLs, nil
	}
	return nil, nil
}

func isAlpha(s string) bool {
	for _, c := range s {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func hasUpperAndLower(s string) bool {
	hasUpper, hasLower := false, false
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		}
		if c >= 'a' && c <= 'z' {
			hasLower = true
		}
	}
	return hasUpper && hasLower
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
