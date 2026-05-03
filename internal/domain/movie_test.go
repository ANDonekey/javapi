package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmoke(t *testing.T) {
	assert.True(t, true)
}

func TestMovieJSONRoundTrip(t *testing.T) {
	movie := &Movie{
		ID:               "abc123",
		Number:           "ABC-123",
		Title:            "Test Movie Title",
		OriginTitle:      "テストムービー",
		ThumbURL:         "https://example.com/thumb.jpg",
		CoverURL:         "https://example.com/cover.jpg",
		Duration:         120,
		Score:            8.5,
		ReleaseDate:      "2025-01-15",
		MagnetsCount:     42,
		CanPlay:          true,
		HasPreviewVideo:  true,
		HasPreviewImages: false,
		PreviewImages:    nil,
		PreviewVideoURL:  "https://example.com/preview.mp4",
		Summary:          "A test movie for round-trip verification.",
		Actors:           []string{"Actor A", "Actor B"},
		Tags:             []string{"tag1", "tag2"},
		DirectorName:     "Director Name",
		MakerName:        "Maker Name",
		PublisherName:    "Publisher Name",
		SeriesName:       "Series Name",
	}

	data, err := json.Marshal(movie)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var decoded Movie
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, movie.ID, decoded.ID)
	assert.Equal(t, movie.Number, decoded.Number)
	assert.Equal(t, movie.Title, decoded.Title)
	assert.Equal(t, movie.OriginTitle, decoded.OriginTitle)
	assert.Equal(t, movie.ThumbURL, decoded.ThumbURL)
	assert.Equal(t, movie.CoverURL, decoded.CoverURL)
	assert.Equal(t, movie.Duration, decoded.Duration)
	assert.Equal(t, movie.Score, decoded.Score)
	assert.Equal(t, movie.ReleaseDate, decoded.ReleaseDate)
	assert.Equal(t, movie.MagnetsCount, decoded.MagnetsCount)
	assert.Equal(t, movie.CanPlay, decoded.CanPlay)
	assert.Equal(t, movie.HasPreviewVideo, decoded.HasPreviewVideo)
	assert.Equal(t, movie.HasPreviewImages, decoded.HasPreviewImages)
	assert.Equal(t, movie.PreviewImages, decoded.PreviewImages)
	assert.Equal(t, movie.PreviewVideoURL, decoded.PreviewVideoURL)
	assert.Equal(t, movie.Summary, decoded.Summary)
	assert.Equal(t, movie.Actors, decoded.Actors)
	assert.Equal(t, movie.Tags, decoded.Tags)
	assert.Equal(t, movie.DirectorName, decoded.DirectorName)
	assert.Equal(t, movie.MakerName, decoded.MakerName)
	assert.Equal(t, movie.PublisherName, decoded.PublisherName)
	assert.Equal(t, movie.SeriesName, decoded.SeriesName)

	assert.Equal(t, movie, &decoded)
}

func TestVideoResultRoundTrip(t *testing.T) {
	result := &VideoResult{
		SiteName: "javdb",
		Status:   StatusSuccess,
		Version:  VersionOriginal,
		Label:    "Original",
		PageURL:  "https://example.com/video/123",
		VideoSources: []VideoSource{
			{
				URL:      "https://example.com/video.mp4",
				Type:     "video/mp4",
				Quality:  "1080p",
				Location: "japan",
			},
		},
		Subtitle: false,
		Leak:     false,
		Error:    "",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var decoded VideoResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.SiteName, decoded.SiteName)
	assert.Equal(t, result.Status, decoded.Status)
	assert.Equal(t, result.Version, decoded.Version)
	assert.Equal(t, len(result.VideoSources), len(decoded.VideoSources))
	assert.Equal(t, result.VideoSources[0].URL, decoded.VideoSources[0].URL)
	assert.Equal(t, result.Subtitle, decoded.Subtitle)
	assert.Equal(t, result.Leak, decoded.Leak)
}

func TestSearchResponseRoundTrip(t *testing.T) {
	resp := &SearchResponse{
		Code: "ABC-123",
		Movie: &Movie{
			Number: "ABC-123",
			Title:  "Test Movie",
		},
		Videos: []VideoResult{
			{
				SiteName: "javdb",
				Status:   StatusSuccess,
				Version:  VersionCNSub,
			},
		},
		Cache: CacheInfo{
			Hit: true,
			Age: 5000,
		},
		TookMs: 1234,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var decoded SearchResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Code, decoded.Code)
	assert.Equal(t, resp.Movie.Number, decoded.Movie.Number)
	assert.Equal(t, resp.Cache.Hit, decoded.Cache.Hit)
	assert.Equal(t, resp.TookMs, decoded.TookMs)
	assert.Equal(t, len(resp.Videos), len(decoded.Videos))
}

func TestVideoVersionConstants(t *testing.T) {
	assert.Equal(t, VideoVersion("original"), VersionOriginal)
	assert.Equal(t, VideoVersion("cnsub"), VersionCNSub)
	assert.Equal(t, VideoVersion("mosaic_reduce"), VersionMosaicReduce)
}

func TestScrapeStatusConstants(t *testing.T) {
	assert.Equal(t, ScrapeStatus("success"), StatusSuccess)
	assert.Equal(t, ScrapeStatus("error"), StatusError)
	assert.Equal(t, ScrapeStatus("timeout"), StatusTimeout)
	assert.Equal(t, ScrapeStatus("not_found"), StatusNotFound)
	assert.Equal(t, ScrapeStatus("blocked"), StatusBlocked)
	assert.Equal(t, ScrapeStatus("skipped"), StatusSkipped)
}
