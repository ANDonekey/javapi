package domain

// VideoVersion represents the version/type of a video source.
type VideoVersion string

const (
	VersionOriginal     VideoVersion = "original"
	VersionCNSub        VideoVersion = "cnsub"
	VersionMosaicReduce VideoVersion = "mosaic_reduce"
)

// ScrapeStatus represents the outcome of a scrape attempt.
type ScrapeStatus string

const (
	StatusSuccess   ScrapeStatus = "success"
	StatusError     ScrapeStatus = "error"
	StatusTimeout   ScrapeStatus = "timeout"
	StatusNotFound  ScrapeStatus = "not_found"
	StatusBlocked   ScrapeStatus = "blocked"
	StatusSkipped   ScrapeStatus = "skipped"
)

// VideoSource describes a single playable video URL.
type VideoSource struct {
	URL      string `json:"url"`
	Type     string `json:"type"`               // "video/mp4", "application/x-mpegURL", "text/html"
	Quality  string `json:"quality,omitempty"`  // "720p", "1080p"
	Location string `json:"location,omitempty"` // server location hint
}

// VideoResult holds the scrape result from a single scraper for a specific version.
type VideoResult struct {
	SiteName     string        `json:"siteName"`
	Status       ScrapeStatus  `json:"status"`
	Version      VideoVersion  `json:"version"`
	Label        string        `json:"label,omitempty"`
	PageURL      string        `json:"pageUrl,omitempty"`
	VideoSources []VideoSource `json:"videoSources,omitempty"`
	Subtitle     bool          `json:"subtitle"`
	Leak         bool          `json:"leak"`
	Error        string        `json:"error,omitempty"`
}
