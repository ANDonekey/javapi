package handler

import (
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/henry/javapi/internal/scraper"
)

func ProxyURL(w http.ResponseWriter, r *http.Request) {
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

	var client *http.Client
	if proxyURL != "" && (strings.Contains(targetURL, "jav-vids.xyz") || strings.Contains(targetURL, "pixibay.cc")) {
		client = scraper.NewCFClient(proxyURL)
	} else {
		client = &http.Client{}
	}

	req, _ := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, "failed to fetch")
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=300")
	io.Copy(w, resp.Body)
}
