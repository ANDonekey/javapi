package handler

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"
)

var m3u8Client = &http.Client{Timeout: 10 * time.Second}

// ProxyM3U8 fetches a remote M3U8 playlist through the API.
// Accepts a base64-encoded URL via the ?url= query parameter.
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

	req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,application/x-mpegURL,*/*")

	resp, err := m3u8Client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch M3U8")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, io.LimitReader(resp.Body, 5<<20))
}
