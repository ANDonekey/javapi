package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

	proxyURL := os.Getenv("EMBED_PROXY_URL")
	if proxyURL == "" {
		proxyURL = os.Getenv("SCRAPER_MISSAV_PROXY_URL")
	}

	content, err := fetchM3U8(r, targetURL, proxyURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch M3U8: "+err.Error())
		return
	}

	// Resolve relative variant playlist URLs to absolute (needed for hls.js)
	// Leave all other lines (.ts segments, tags) unchanged
	content = resolveRelativeURLs(content, targetURL)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write([]byte(content))
}

func resolveRelativeURLs(content, baseURL string) string {
	base, _ := url.Parse(baseURL)
	if base == nil {
		return content
	}
	var out strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// Only resolve lines that are variant playlist URLs (.m3u8 or tokens)
		// Leave tags (#), comments, empty lines, and .ts segments unchanged
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out.WriteString(line + "\n")
		} else if strings.Contains(trimmed, ".m3u8") {
			if !strings.HasPrefix(trimmed, "http") {
				ref, _ := url.Parse(trimmed)
				if ref != nil {
					trimmed = base.ResolveReference(ref).String()
				}
			}
			out.WriteString(trimmed + "\n")
		} else {
			out.WriteString(line + "\n")
		}
	}
	return out.String()
}

func fetchM3U8(r *http.Request, targetURL, proxyURL string) (string, error) {
	var client *http.Client
	if proxyURL != "" && (strings.Contains(targetURL, "acek-cdn") ||
		strings.Contains(targetURL, "dramiyos-cdn") ||
		strings.Contains(targetURL, "?t=")) {
		client = scraper.NewCFClient(proxyURL)
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
