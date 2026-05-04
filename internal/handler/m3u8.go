package handler

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/henry/javapi/internal/scraper"
)

// ProxyM3U8 fetches a remote M3U8 playlist, rewrites segment URLs to proxy through this API,
// and returns the modified playlist to the client.
func ProxyM3U8(w http.ResponseWriter, r *http.Request) {
	encoded := r.URL.Query().Get("url")
	if encoded == "" {
		writeError(w, http.StatusBadRequest, "url parameter is required (base64-encoded)")
		return
	}

	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid base64 encoding")
		return
	}

	targetURL := string(decoded)
	if !strings.HasPrefix(targetURL, "https://") && !strings.HasPrefix(targetURL, "http://") {
		writeError(w, http.StatusBadRequest, "only HTTP/HTTPS URLs are allowed")
		return
	}

	req, _ := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,application/x-mpegURL,*/*")

	client := scraper.NewCFClient("")
	resp, err := client.Do(req)
	if err != nil {
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
			rewritten.WriteString(line)
			rewritten.WriteString("\n")
			continue
		}

		segmentURL := trimmed
		if !strings.HasPrefix(segmentURL, "http") {
			ref, _ := url.Parse(segmentURL)
			if baseURL != nil && ref != nil {
				segmentURL = baseURL.ResolveReference(ref).String()
			}
		}

		// Write absolute URL directly (no proxy for .ts segments)
		rewritten.WriteString(segmentURL)
		rewritten.WriteString("\n")
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rewritten.String()))
}

func getScheme(r *http.Request) string {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}
