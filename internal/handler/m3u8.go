package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/henry/javapi/internal/scraper"
)

func ProxyM3U8(w http.ResponseWriter, r *http.Request) {
	encoded := r.URL.Query().Get("url")
	if encoded == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required")
		return
	}
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil || (!strings.HasPrefix(string(decoded), "https://") && !strings.HasPrefix(string(decoded), "http://")) {
		writeError(w, http.StatusBadRequest, "invalid URL")
		return
	}
	targetURL := string(decoded)

	content, err := fetchM3U8(r, targetURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	if strings.Contains(content, "#EXT-X-STREAM-INF") {
		bestVariant := resolveBestVariant(content, targetURL)
		if bestVariant != "" {
			varContent, err := fetchM3U8(r, bestVariant)
			if err == nil {
				content = rewriteSegments(varContent, bestVariant)
				w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Cache-Control", "public, max-age=60")
				w.Write([]byte(content))
				return
			}
		}
	}

	content = rewriteSegments(content, targetURL)
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write([]byte(content))
}

func fetchM3U8(r *http.Request, targetURL string) (string, error) {
	var client *http.Client
	if strings.Contains(targetURL, "acek-cdn") || strings.Contains(targetURL, "?t=") {
		client = scraper.NewCFClient(os.Getenv("SCRAPER_MISSAV_PROXY_URL"))
	} else {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, _ := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0")
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,application/x-mpegURL,text/plain,*/*")
	if strings.Contains(targetURL, "surrit.com") {
		req.Header.Set("Referer", "https://missav.ws/")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	return string(body), nil
}

func resolveBestVariant(content, baseURL string) string {
	var bestURL string
	var bestBW int
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			continue
		}
		bw := 0
		for _, part := range strings.Split(line, ",") {
			if strings.HasPrefix(strings.TrimSpace(part), "BANDWIDTH=") {
				bw, _ = strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(part), "BANDWIDTH="))
			}
		}
		if i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next != "" && !strings.HasPrefix(next, "#") {
				if !strings.HasPrefix(next, "http") {
					u, _ := url.Parse(baseURL)
					ref, _ := url.Parse(next)
					if u != nil && ref != nil {
						next = u.ResolveReference(ref).String()
					}
				}
				if bw > bestBW {
					bestBW = bw
					bestURL = next
				}
			}
		}
	}
	return bestURL
}

func rewriteSegments(content, baseURL string) string {
	base, _ := url.Parse(baseURL)
	var out strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out.WriteString(line + "\n")
			continue
		}
		segURL := trimmed
		if !strings.HasPrefix(segURL, "http") && base != nil {
			ref, _ := url.Parse(segURL)
			if ref != nil {
				segURL = base.ResolveReference(ref).String()
			}
		}
		out.WriteString(segURL + "\n")
	}
	return out.String()
}
