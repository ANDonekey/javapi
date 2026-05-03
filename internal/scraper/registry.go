package scraper

import (
	"sync"

	"github.com/henry/javapi/internal/config"
	"github.com/henry/javapi/internal/domain"
)

var (
	mu       sync.RWMutex
	scrapers = make(map[string]domain.Scraper)
)

// Register adds a scraper to the global registry keyed by its Name.
// It is safe for concurrent use.
func Register(s domain.Scraper) {
	mu.Lock()
	defer mu.Unlock()
	scrapers[s.Name()] = s
}

// GetAll returns all registered scrapers regardless of enabled state.
func GetAll() []domain.Scraper {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]domain.Scraper, 0, len(scrapers))
	for _, s := range scrapers {
		result = append(result, s)
	}
	return result
}

// GetEnabled returns only scrapers that are currently enabled.
func GetEnabled() []domain.Scraper {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]domain.Scraper, 0, len(scrapers))
	for _, s := range scrapers {
		if s.IsEnabled() {
			result = append(result, s)
		}
	}
	return result
}

// ApplyConfig applies per-site proxy configuration to all registered scrapers.
func ApplyConfig(sites []config.ScraperSiteConfig) {
	mu.Lock()
	defer mu.Unlock()
	for _, site := range sites {
		if s, ok := scrapers[site.Name]; ok {
			if pc, hasSetter := s.(interface{ SetProxyConfig(domain.ProxyConfig) }); hasSetter {
				pc.SetProxyConfig(domain.ProxyConfig{URL: site.ProxyURL, Enabled: site.ProxyEnabled})
			}
		}
	}
}
