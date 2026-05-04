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

func TestPickBestVariant_SingleVariant(t *testing.T) {
	content := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360
variant_360p.m3u8
`
	bestURL, bestBW := pickBestVariant(content, "https://example.com/master.m3u8")
	assert.Equal(t, "https://example.com/variant_360p.m3u8", bestURL)
	assert.Equal(t, 500000, bestBW)
}

func TestPickBestVariant_MultipleVariants(t *testing.T) {
	content := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=500000,RESOLUTION=640x360
variant_360p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720
variant_720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=5000000,RESOLUTION=1920x1080
variant_1080p.m3u8
`
	bestURL, bestBW := pickBestVariant(content, "https://example.com/master.m3u8")
	assert.Equal(t, "https://example.com/variant_1080p.m3u8", bestURL)
	assert.Equal(t, 5000000, bestBW)
}

func TestPickBestVariant_IFramesIgnored(t *testing.T) {
	content := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=500000
variant_low.m3u8
#EXT-X-I-FRAME-STREAM-INF:BANDWIDTH=99999999
keyframes.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1000000
variant_high.m3u8
`
	bestURL, bestBW := pickBestVariant(content, "https://example.com/master.m3u8")
	assert.Equal(t, "https://example.com/variant_high.m3u8", bestURL)
	assert.Equal(t, 1000000, bestBW)
}

func TestPickBestVariant_AbsoluteURL(t *testing.T) {
	content := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=1000000
https://othercdn.example.com/variant.m3u8
`
	bestURL, _ := pickBestVariant(content, "https://example.com/master.m3u8")
	assert.Equal(t, "https://othercdn.example.com/variant.m3u8", bestURL)
}

func TestPickBestVariant_NoVariants(t *testing.T) {
	content := `#EXTM3U
#EXTINF:10.0,
segment_001.ts
#EXTINF:10.0,
segment_002.ts
`
	bestURL, bestBW := pickBestVariant(content, "https://example.com/variant.m3u8")
	assert.Equal(t, "", bestURL)
	assert.Equal(t, 0, bestBW)
}

func TestRewriteSegments_RelativeSegments(t *testing.T) {
	content := `#EXTM3U
#EXTINF:10.0,
segment_001.ts
#EXTINF:10.0,
segment_002.ts
#EXT-X-ENDLIST
`
	result := rewriteSegments(content, "https://cdn.example.com/path/variant.m3u8")
	assert.Contains(t, result, "https://cdn.example.com/path/segment_001.ts")
	assert.Contains(t, result, "https://cdn.example.com/path/segment_002.ts")
	assert.Contains(t, result, "#EXTINF:10.0,")
	assert.Contains(t, result, "#EXT-X-ENDLIST")
}

func TestRewriteSegments_AbsoluteSegmentsPreserved(t *testing.T) {
	content := `#EXTM3U
#EXTINF:10.0,
https://othercdn.example.com/segment_001.ts
`
	result := rewriteSegments(content, "https://cdn.example.com/variant.m3u8")
	assert.Contains(t, result, "https://othercdn.example.com/segment_001.ts")
}

func TestRewriteSegments_EmptyContent(t *testing.T) {
	result := rewriteSegments("", "https://example.com/variant.m3u8")
	assert.Equal(t, "", result)
}

func TestRewriteSegments_MasterPlaylistPassThrough(t *testing.T) {
	content := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=500000
variant.m3u8
`
	result := rewriteSegments(content, "https://example.com/master.m3u8")
	assert.Contains(t, result, "#EXT-X-STREAM-INF:BANDWIDTH=500000")
	assert.Contains(t, result, "https://example.com/variant.m3u8")
}
