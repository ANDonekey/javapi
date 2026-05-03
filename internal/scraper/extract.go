package scraper

import "regexp"

var m3u8RawRE = regexp.MustCompile(`https?://[^'"\\\s<>]+\.m3u8[^'"\\\s<>]*`)

// ExtractM3U8FromRawHTML finds all M3U8 URLs in raw HTML text.
// This catches URLs in script tags, JS variables, and data attributes
// that DOM-based selectors (goquery) cannot find.
func ExtractM3U8FromRawHTML(html string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, match := range m3u8RawRE.FindAllString(html, -1) {
		if !seen[match] {
			seen[match] = true
			result = append(result, match)
		}
	}
	return result
}
