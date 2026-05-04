package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

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

	proxyURL := os.Getenv("EMBED_PROXY_URL")
	if proxyURL == "" {
		proxyURL = os.Getenv("SCRAPER_MISSAV_PROXY_URL")
	}
	log.Printf("m3u8 proxy: url=%.60s proxy=%.30s", targetURL, proxyURL)

	content, err := fetchM3U8(r, targetURL, proxyURL)
	if err != nil {
		log.Printf("m3u8 proxy: FAILED err=%v", err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to fetch M3U8: %v", err))
		return
	}

	if strings.Contains(content, "#EXT-X-STREAM-INF") {
		bestURL, bestBW := pickBestVariant(content, targetURL)
		if bestURL != "" {
			log.Printf("m3u8 proxy: resolved master→variant bw=%d url=%.60s", bestBW, bestURL)
			varContent, ferr := fetchM3U8(r, bestURL, proxyURL)
			if ferr == nil {
				w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Cache-Control", "public, max-age=60")
				w.Write([]byte(varContent))
				return
			}
			log.Printf("m3u8 proxy: variant fetch failed err=%v", ferr)
		}
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write([]byte(content))
}

func fetchM3U8(r *http.Request, targetURL, proxyURL string) (string, error) {
	client := scraper.NewCFClient(proxyURL)
	req, _ := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	req.Header.Set("User-Agent", scraper.FirefoxUA)
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,application/x-mpegURL,text/plain,*/*")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(body), nil
}

func pickBestVariant(content, baseURL string) (string, int) {
	var bestURL string
	bestBW := 0
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF") || strings.Contains(line, "I-FRAME") {
			continue
		}
		bw := 0
		for _, part := range strings.Split(line, ",") {
			part = strings.TrimSpace(part)
			if idx := strings.Index(part, "BANDWIDTH="); idx >= 0 {
				bw, _ = strconv.Atoi(part[idx+len("BANDWIDTH="):])
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
	return bestURL, bestBW
}

func rewriteSegments(content, baseURLStr string) string {
	if content == "" {
		return ""
	}
	baseURL, _ := url.Parse(baseURLStr)
	var rewritten strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			rewritten.WriteString(line + "\n")
			continue
		}
		segURL := trimmed
		if !strings.HasPrefix(segURL, "http") && baseURL != nil {
			ref, _ := url.Parse(segURL)
			if ref != nil {
				segURL = baseURL.ResolveReference(ref).String()
			}
		}
		rewritten.WriteString(segURL + "\n")
	}
	return rewritten.String()
}
