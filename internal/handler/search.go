package handler

import (
	"net/http"
	"regexp"

	"github.com/henry/javapi/internal/aggregator"
)

var validCodeRE = regexp.MustCompile(`^[a-zA-Z0-9\-_ ]{3,20}$`)

type SearchHandler struct {
	svc *aggregator.Service
}

func NewSearchHandler(svc *aggregator.Service) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" || !validCodeRE.MatchString(code) {
		writeError(w, http.StatusBadRequest, "code parameter is required (3-20 alphanumeric characters)")
		return
	}
	resp, err := h.svc.Aggregate(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// StreamSearch streams search results as NDJSON lines as they complete.
func (h *SearchHandler) StreamSearch(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" || !validCodeRE.MatchString(code) {
		writeError(w, http.StatusBadRequest, "code parameter is required")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_ = h.svc.AggregateStream(r.Context(), code, w, flusher)
}
