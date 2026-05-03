package domain

// SearchRequest represents a movie search request by JAV code.
type SearchRequest struct {
	Code string `json:"code" form:"code"`
}

// CacheInfo describes the cache state for a search response.
type CacheInfo struct {
	Hit bool  `json:"hit"`
	Age int64 `json:"age"` // milliseconds since cached
}

// SearchResponse is the full API response for a movie search.
type SearchResponse struct {
	Code   string        `json:"code"`
	Movie  *Movie        `json:"movie,omitempty"`
	Videos []VideoResult `json:"videos"`
	Cache  CacheInfo     `json:"cache"`
	TookMs int64         `json:"took_ms"`
}
