package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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

	// Use fixed-session proxy for signed CDN URLs (jav-vids.xyz chain)
	// so embed page and M3U8 fetch use same IP
	proxyURL := os.Getenv("EMBED_PROXY_URL")
	if proxyURL == "" {
		proxyURL = os.Getenv("SCRAPER_MISSAV_PROXY_URL")
	}
	log.Printf("m3u8 proxy: url=%.60s proxy=%.30s", targetURL, proxyURL)

	client := scraper.NewCFClient(proxyURL)
	req, _ := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	req.Header.Set("User-Agent", scraper.FirefoxUA)
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,application/x-mpegURL,text/plain,*/*")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		status := 0
		if resp != nil { status = resp.StatusCode }
		log.Printf("m3u8 proxy: FAILED status=%d err=%v", status, err)
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to fetch M3U8 (HTTP %d)", status))
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	content := string(body)

	baseURL, _ := url.Parse(targetURL)
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
			if ref != nil { segURL = baseURL.ResolveReference(ref).String() }
		}
		rewritten.WriteString(segURL + "\n")
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write([]byte(rewritten.String()))
}
