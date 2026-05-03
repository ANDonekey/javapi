package domain

import "context"

// ProxyConfig holds the HTTP proxy configuration for a scraper.
type ProxyConfig struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// Scraper defines the interface that each video site scraper must implement.
type Scraper interface {
	// Name returns a unique identifier for this scraper (e.g. "javdb", "mgstage").
	Name() string

	// Search scrapes video results for the given JAV code.
	Search(ctx context.Context, code string) ([]VideoResult, error)

	// FormatCode normalizes the input code into the format expected by the site.
	FormatCode(code string) string

	// IsEnabled indicates whether this scraper is currently enabled.
	IsEnabled() bool

	// RequiresCFBypass returns true if this site needs Cloudflare bypass (cloudscraper_go).
	RequiresCFBypass() bool

	// GetProxyConfig returns the proxy configuration for this scraper.
	GetProxyConfig() ProxyConfig
}
