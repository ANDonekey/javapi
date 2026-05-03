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
