package handler

import (
	"encoding/base64"
	"io"
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

	proxyURL := os.Getenv("SCRAPER_MISSAV_PROXY_URL")
	client := scraper.NewCFClient(proxyURL)
	req, _ := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	req.Header.Set("User-Agent", scraper.FirefoxUA)
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,*/*")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, "failed to fetch M3U8")
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
		segmentURL := trimmed
		if !strings.HasPrefix(segmentURL, "http") {
			ref, _ := url.Parse(segmentURL)
			if baseURL != nil && ref != nil {
				segmentURL = baseURL.ResolveReference(ref).String()
			}
		}
		rewritten.WriteString(segmentURL + "\n")
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write([]byte(rewritten.String()))
}
