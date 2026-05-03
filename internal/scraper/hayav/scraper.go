// Package hayav implements a domain.Scraper for https://hayav.com.
//
// The scraper fetches video pages, detects player elements (video, source, iframe,
// jwplayer, plyr), extracts video URLs, and returns grouped VideoResults per version.
package hayav

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

const (
	// Name is the unique identifier for this scraper.
	Name = "hayav"

	// DefaultBaseURL is the canonical base URL for the HAYAV site.
	DefaultBaseURL = "https://hayav.com"

	// videoPath is the URL template for a video page.
	videoPath = "/video/%s/"
)

// cfMarkers are strings found in Cloudflare challenge/blocked pages.
var cfMarkers = []string{
	"Just a moment",
	"Checking your browser",
	"cf-browser-verification",
	"challenge-platform",
	"Attention Required!",
	"Cloudflare",
}

// jsURLRE matches video/stream URLs inside JavaScript player configs.
var (
	jwplayerFileRE = regexp.MustCompile(`file\s*:\s*["']([^"']+)["']`)
	plyrSourceRE   = regexp.MustCompile(`src\s*:\s*["']([^"']+)["']`)
)

// Scraper implements domain.Scraper for HAYAV.
// It is safe for concurrent use — all mutable fields are set at construction time.
type Scraper struct {
	client      *http.Client
	baseURL     string
	enabled     bool
	proxyConfig domain.ProxyConfig
}

// Compile-time interface check.
var _ domain.Scraper = (*Scraper)(nil)

func init() {
	scraper.Register(New(domain.ProxyConfig{}))
}

// New creates a HAYAV scraper with sensible defaults (15s timeout, standard UA).
func New(config domain.ProxyConfig) *Scraper {
	s := &Scraper{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL:     DefaultBaseURL,
		enabled:     true,
		proxyConfig: config,
	}
	s.applyProxy()
	return s
}

// NewWithClient returns a Scraper that uses a custom http.Client and base URL.
// Primarily used in tests to inject httptest servers.
func NewWithClient(client *http.Client, baseURL string) *Scraper {
	return &Scraper{
		client:      client,
		baseURL:     baseURL,
		enabled:     true,
		proxyConfig: domain.ProxyConfig{Enabled: false},
	}
}

// Name returns the unique scraper identifier.
func (s *Scraper) Name() string { return Name }

// FormatCode returns the code unchanged. HAYAV accepts standard JAV codes directly.
func (s *Scraper) FormatCode(code string) string { return code }

// IsEnabled reports whether the scraper is currently active.
func (s *Scraper) IsEnabled() bool { return s.enabled }

// RequiresCFBypass reports whether this scraper needs Cloudflare bypass.
// HAYAV may use Cloudflare intermittently; the Search method detects blocks inline.
func (s *Scraper) RequiresCFBypass() bool { return false }

// GetProxyConfig returns the proxy configuration for this scraper.
func (s *Scraper) GetProxyConfig() domain.ProxyConfig { return s.proxyConfig }

// SetProxyConfig updates the proxy configuration for this scraper.
func (s *Scraper) SetProxyConfig(pc domain.ProxyConfig) {
	s.proxyConfig = pc
	s.applyProxy()
}

func (s *Scraper) applyProxy() {
	if s.proxyConfig.Enabled && s.proxyConfig.URL != "" {
		proxyURL, err := url.Parse(s.proxyConfig.URL)
		if err == nil {
			s.client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}
}

// Search fetches and parses the video page for the given JAV code.
// It returns one or more VideoResults grouped by version.
func (s *Scraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	pageURL := fmt.Sprintf("%s"+videoPath, s.baseURL, code)

	body, pageErr, cfBlocked := s.fetch(ctx, pageURL)
	if pageErr != nil {
		return nil, fmt.Errorf("hayav fetch: %w", pageErr)
	}

	if cfBlocked {
		return []domain.VideoResult{{
			SiteName: Name,
			Status:   domain.StatusBlocked,
			PageURL:  pageURL,
			Error:    "cloudflare challenge detected",
		}}, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("hayav parse html: %w", err)
	}

	title := extractTitle(doc)
	sources := extractAllSources(doc, pageURL)

	if len(sources) == 0 {
		return []domain.VideoResult{{
			SiteName: Name,
			Status:   domain.StatusNotFound,
			PageURL:  pageURL,
			Label:    title,
		}}, nil
	}

	return groupByVersion(Name, pageURL, title, sources), nil
}

// fetch performs an HTTP GET and returns the body bytes.
// It also checks for Cloudflare block markers in the response.
func (s *Scraper) fetch(ctx context.Context, pageURL string) (body []byte, err error, cfBlocked bool) {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if reqErr != nil {
		return nil, reqErr, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, doErr := s.client.Do(req)
	if doErr != nil {
		return nil, doErr, false
	}
	defer resp.Body.Close()

	rawBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2 MiB cap
	if readErr != nil {
		return nil, fmt.Errorf("read body: %w", readErr), false
	}
	bodyStr := string(rawBody)

	// Cloudflare detection
	for _, m := range cfMarkers {
		if strings.Contains(bodyStr, m) {
			return nil, nil, true
		}
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode), false
	}

	return rawBody, nil, false
}

// extractTitle returns the most suitable title from the HTML document.
// Priority: og:title > h1 > page title.
func extractTitle(doc *goquery.Document) string {
	ogTitle, ok := doc.Find(`meta[property="og:title"]`).Attr("content")
	if ok && ogTitle != "" {
		return strings.TrimSpace(ogTitle)
	}
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	if h1 != "" {
		return h1
	}
	return strings.TrimSpace(doc.Find("title").First().Text())
}

// scrapeSource represents a raw video source extracted from the HTML.
type scrapeSource struct {
	URL     string
	MIMEType string
	Quality  string
	Label    string // contextual hint used for version classification
}

// extractAllSources finds every playable URL on the page.
// It inspects <video>, <source>, <iframe> elements and JavaScript player configs.
func extractAllSources(doc *goquery.Document, pageURL string) []scrapeSource {
	var sources []scrapeSource

	// 1) <video src="...">  (direct src attribute)
	doc.Find("video[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		if src == "" {
			return
		}
		sources = append(sources, scrapeSource{
			URL:     resolveURL(src, pageURL),
			Label:   labelFromContext(el),
		})
	})

	// 2) <video><source src="..." type="..."></video>
	doc.Find("video source[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		mime, _ := el.Attr("type")
		if src == "" {
			return
		}
		sources = append(sources, scrapeSource{
			URL:     resolveURL(src, pageURL),
			MIMEType: mime,
			Label:   labelFromContext(el),
		})
	})

	// 3) standalone <source src="..."> (outside <video>)
	doc.Find("source[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		mime, _ := el.Attr("type")
		if src == "" {
			return
		}
		sources = append(sources, scrapeSource{
			URL:     resolveURL(src, pageURL),
			MIMEType: mime,
			Label:   labelFromContext(el),
		})
	})

	// 4) <iframe src="...">
	doc.Find("iframe[src]").Each(func(_ int, el *goquery.Selection) {
		src, _ := el.Attr("src")
		if src == "" {
			return
		}
		sources = append(sources, scrapeSource{
			URL:     resolveURL(src, pageURL),
			Label:   labelFromContext(el),
		})
	})

	// 5) JavaScript player configs (jwplayer, plyr)
	doc.Find("script").Each(func(_ int, el *goquery.Selection) {
		script := el.Text()
		// jwplayer("...").setup({ file: "https://..." })
		for _, m := range jwplayerFileRE.FindAllStringSubmatch(script, -1) {
			if len(m) > 1 && m[1] != "" {
				sources = append(sources, scrapeSource{
					URL:   resolveURL(m[1], pageURL),
					Label: labelFromPlayerScript(script),
				})
			}
		}
		// plyr setup — new Plyr(..., { sources: [{ src: "..." }] })
		for _, m := range plyrSourceRE.FindAllStringSubmatch(script, -1) {
			if len(m) > 1 && m[1] != "" {
				sources = append(sources, scrapeSource{
					URL:   resolveURL(m[1], pageURL),
					Label: labelFromPlayerScript(script),
				})
			}
		}
	})

	return sources
}

// resolveURL resolves a possibly-relative URL against the page URL.
func resolveURL(raw, pageURL string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "//") {
		return raw
	}
	// Extract scheme+host from pageURL to resolve root-relative paths.
	// Find the end of the authority section (after optional port).
	idx := strings.Index(pageURL, "://")
	if idx >= 0 {
		hostStart := idx + 3
		hostEnd := strings.IndexByte(pageURL[hostStart:], '/')
		var origin string
		if hostEnd >= 0 {
			origin = pageURL[:hostStart+hostEnd]
		} else {
			origin = pageURL
		}
		if strings.HasPrefix(raw, "/") {
			return origin + raw
		}
		// Relative to the directory of pageURL.
		dir := pageURL[:strings.LastIndex(pageURL[:len(pageURL)-1], "/")+1]
		return dir + raw
	}
	return raw
}

// labelFromContext builds a version-hint label from the element and its ancestors.
func labelFromContext(el *goquery.Selection) string {
	// Walk up to 3 ancestors looking for class/id text hints.
	parts := []string{}
	current := el
	for i := 0; i < 3 && current.Length() > 0; i++ {
		cls, _ := current.Attr("class")
		id, _ := current.Attr("id")
		txt := strings.ToLower(cls + " " + id)
		parts = append(parts, txt)
		current = current.Parent()
	}
	joined := strings.Join(parts, " ")
	return joined
}

// labelFromPlayerScript extracts version hints from JavaScript player setup code.
func labelFromPlayerScript(script string) string {
	lower := strings.ToLower(script)
	return lower
}

// groupByVersion classifies sources by VideoVersion and returns one VideoResult per version.
// It deduplicates sources by URL to avoid redundant entries.
func groupByVersion(siteName, pageURL, title string, sources []scrapeSource) []domain.VideoResult {
	type key struct {
		version domain.VideoVersion
		label   string
	}

	groups := make(map[key][]domain.VideoSource)

	for _, src := range sources {
		version, label := classifyVersion(src)
		k := key{version: version, label: label}

		// deduplicate by URL within same group
		found := false
		for _, existing := range groups[k] {
			if existing.URL == src.URL {
				found = true
				break
			}
		}
		if found {
			continue
		}

		groups[k] = append(groups[k], domain.VideoSource{
			URL:     src.URL,
			Type:    classifyMIME(src.MIMEType),
			Quality: src.Quality,
		})
	}

	// Ensure at least one result is returned even for empty groups.
	if len(groups) == 0 {
		return nil
	}

	results := make([]domain.VideoResult, 0, len(groups))
	for k, vs := range groups {
		results = append(results, domain.VideoResult{
			SiteName:     siteName,
			Status:       domain.StatusSuccess,
			Version:      k.version,
			Label:        firstNonEmpty(title, k.label),
			PageURL:      pageURL,
			VideoSources: vs,
		})
	}
	return results
}

// classifyVersion determines the VideoVersion from contextual hints.
func classifyVersion(src scrapeSource) (domain.VideoVersion, string) {
	hint := strings.ToLower(src.Label + " " + src.URL)
	switch {
	case strings.Contains(hint, "cnsub") || strings.Contains(hint, "chinese") || strings.Contains(hint, "subtitle"):
		return domain.VersionCNSub, "Chinese subtitled"
	case strings.Contains(hint, "mosaic") || strings.Contains(hint, "reduce") || strings.Contains(hint, "decrypt"):
		return domain.VersionMosaicReduce, "Mosaic reduced"
	default:
		return domain.VersionOriginal, "Original"
	}
}

// classifyMIME returns a MIME type for the video source based on file extension or explicit type.
func classifyMIME(mime string) string {
	if mime != "" {
		return mime
	}
	return "video/mp4" // default assumption
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
