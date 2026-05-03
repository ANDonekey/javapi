package embed

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/henry/javapi/internal/domain"
)

type Extractor interface {
	MatchHost(host string) bool
	Extract(ctx context.Context, client *http.Client, pageURL string) ([]string, error)
}

var extractors []Extractor

func Register(e Extractor) { extractors = append(extractors, e) }

func ResolveEmbed(ctx context.Context, src domain.VideoSource) domain.VideoSource {
	if src.Type != "text/html" && src.Type != "" {
		return src
	}
	u, err := url.Parse(src.URL)
	if err != nil {
		return src
	}
	host := strings.ToLower(u.Hostname())
	for _, e := range extractors {
		if !e.MatchHost(host) {
			continue
		}
		client := &http.Client{Timeout: 15 * time.Second}
		urls, err := e.Extract(ctx, client, src.URL)
		if err == nil && len(urls) > 0 {
			variants, err := resolveM3U8Playlist(ctx, client, urls[0])
			if err == nil && len(variants) > 1 {
				best := variants[0]
				for _, v := range variants[1:] {
					if v.Bandwidth > best.Bandwidth {
						best = v
					}
				}
				return domain.VideoSource{URL: best.URL, Type: "application/x-mpegURL", Quality: best.Quality}
			}
			return domain.VideoSource{URL: urls[0], Type: "application/x-mpegURL"}
		}
	}
	return src
}

type playlistVariant struct {
	URL       string
	Bandwidth int
	Quality   string
}

func resolveM3U8Playlist(ctx context.Context, client *http.Client, masterURL string) ([]playlistVariant, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", masterURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,*/*")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	content := string(body)
	lines := strings.Split(content, "\n")
	var variants []playlistVariant
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			continue
		}
		bandwidth := 0
		resolution := ""
		for _, part := range strings.Split(line, ",") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "BANDWIDTH=") {
				bandwidth, _ = strconv.Atoi(strings.TrimPrefix(part, "BANDWIDTH="))
			}
			if strings.HasPrefix(part, "RESOLUTION=") {
				resolution = strings.TrimPrefix(part, "RESOLUTION=")
			}
		}
		if i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine != "" && !strings.HasPrefix(nextLine, "#") {
				if !strings.HasPrefix(nextLine, "http") {
					u, _ := url.Parse(masterURL)
					ref, _ := url.Parse(nextLine)
					if u != nil && ref != nil {
						nextLine = u.ResolveReference(ref).String()
					}
				}
				quality := resolutionToQuality(resolution)
				variants = append(variants, playlistVariant{
					URL: nextLine, Bandwidth: bandwidth, Quality: quality,
				})
			}
		}
	}
	if len(variants) == 0 {
		return []playlistVariant{{URL: masterURL}}, nil
	}
	return variants, nil
}

func resolutionToQuality(res string) string {
	switch {
	case strings.Contains(res, "1080"):
		return "1080p"
	case strings.Contains(res, "720"):
		return "720p"
	case strings.Contains(res, "480"):
		return "480p"
	case strings.Contains(res, "360"):
		return "360p"
	default:
		return res
	}
}
