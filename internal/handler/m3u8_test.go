package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyM3U8_MissingURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/m3u8", nil)
	w := httptest.NewRecorder()
	ProxyM3U8(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyM3U8_InvalidBase64(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/m3u8?url=!!!invalid!!!", nil)
	w := httptest.NewRecorder()
	ProxyM3U8(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyM3U8_NonHTTP(t *testing.T) {
	encoded := base64.URLEncoding.EncodeToString([]byte("ftp://bad.url/test.m3u8"))
	req := httptest.NewRequest("GET", "/api/v1/m3u8?url="+encoded, nil)
	w := httptest.NewRecorder()
	ProxyM3U8(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
