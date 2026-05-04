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

type ScraperTiming struct {
	Name     string `json:"name"`
	CFTestMs int64  `json:"cf_test_ms"`
	FetchMs  int64  `json:"fetch_ms"`
	EmbedMs  int64  `json:"embed_ms,omitempty"`
	Status   string `json:"status"`
}

type TimingInfo struct {
	TotalMs  int64           `json:"total_ms"`
	JavDBMs  int64           `json:"javdb_ms"`
	Scrapers []ScraperTiming `json:"scrapers"`
	EmbedMs  int64           `json:"embed_ms"`
	CacheMs  int64           `json:"cache_ms"`
}

// SearchResponse is the full API response for a movie search.
type SearchResponse struct {
	Code   string        `json:"code"`
	Movie  *Movie        `json:"movie,omitempty"`
	Videos []VideoResult `json:"videos"`
	Cache  CacheInfo     `json:"cache"`
	TookMs int64         `json:"took_ms"`
	Timing *TimingInfo   `json:"timing,omitempty"`
}
