package scraper

import (
	"regexp"
	"strconv"
	"strings"
)

var m3u8RawRE = regexp.MustCompile(`https?://[^'"\\\s<>]+\.m3u8[^'"\\\s<>]*`)

// packedJSRE matches Dean Edwards JavaScript packer: eval(function(p,a,c,k,e,d){...}('...',r,n,'dict'.split('|'))
var packedJSRE = regexp.MustCompile(`(?s)eval\(function\(p,a,c,k,e,d\).*?\('((?:\\'|[^'])*)',(\d+),(\d+),'([^']*)'\.split\('\|'\)`)

// ExtractM3U8FromRawHTML finds all M3U8 URLs in raw HTML text.
func ExtractM3U8FromRawHTML(html string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, match := range m3u8RawRE.FindAllString(html, -1) {
		fixed := strings.ReplaceAll(match, `\/`, `/`)
		if !seen[fixed] {
			seen[fixed] = true
			result = append(result, fixed)
		}
	}
	for _, unpacked := range UnpackJavascriptStrings(html) {
		for _, match := range m3u8RawRE.FindAllString(unpacked, -1) {
			fixed := strings.ReplaceAll(match, `\/`, `/`)
			if !seen[fixed] {
				seen[fixed] = true
				result = append(result, fixed)
			}
		}
	}
	return result
}

// UnpackJavascriptStrings decodes Dean Edwards packed JavaScript found in HTML.
// Pattern: eval(function(p,a,c,k,e,d){...}('escaped_payload',radix,count,'word|word|word'.split('|'))
// Returns decoded strings that may contain M3U8 URLs.
func UnpackJavascriptStrings(body string) []string {
	matches := packedJSRE.FindAllStringSubmatch(body, -1)
	result := []string{}
	for _, match := range matches {
		if len(match) != 5 {
			continue
		}
		payload := strings.ReplaceAll(match[1], `\'`, `'`)
		radix, _ := strconv.Atoi(match[2])
		words := strings.Split(match[4], "|")

		tokenRe := regexp.MustCompile(`\b[0-9a-zA-Z]+\b`)
		unpacked := tokenRe.ReplaceAllStringFunc(payload, func(token string) string {
			index, err := strconv.ParseInt(token, radix, 64)
			if err != nil || index < 0 || int(index) >= len(words) || words[index] == "" {
				return token
			}
			return words[index]
		})
		result = append(result, unpacked)
	}
	return result
}
