package scraper

import "regexp"

// videoCodeRegex matches JAV video codes: 2-10 letters followed by 2-6 digits,
// with optional whitespace, underscores, or hyphens between them.
var videoCodeRegex = regexp.MustCompile(`[a-zA-Z]{2,10}[\s_-]*\d{2,6}`)

// NormalizeCode lowercases the input and strips hyphens, underscores, and spaces.
// Examples:
//
//	"ABC-123"  → "abc123"
//	"ABC_123"  → "abc123"
//	"ABC 123"  → "abc123"
//	"abc123"   → "abc123"
func NormalizeCode(code string) string {
	result := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if c == '-' || c == '_' || c == ' ' {
			continue
		}
		result = append(result, c)
	}
	return string(result)
}

// ExtractCode finds the first JAV video code pattern in the raw string.
// If no pattern is found, the original string is returned unchanged.
func ExtractCode(raw string) string {
	match := videoCodeRegex.FindString(raw)
	if match == "" {
		return raw
	}
	return match
}
