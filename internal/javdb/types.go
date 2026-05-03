package javdb

import (
	"encoding/json"

	"github.com/henry/javapi/internal/domain"
)

// apiEnvelope is the common response wrapper for all JavDB API endpoints.
type apiEnvelope struct {
	Success int         `json:"success"`
	Action  string      `json:"action"`
	Message string      `json:"message"`
	Data    apiData     `json:"data"`
}

// apiData holds the data payload for both search and detail endpoints.
// Search fills Movies; detail fills Movie.
type apiData struct {
	Movies []searchMovie `json:"movies,omitempty"`
	Movie  detailMovie   `json:"movie,omitempty"`
}

// searchMovie represents a movie returned by /api/v2/search.
type searchMovie struct {
	ID               string        `json:"id"`
	Number           string        `json:"number"`
	Title            string        `json:"title"`
	OriginTitle      string        `json:"origin_title"`
	ThumbURL         string        `json:"thumb_url"`
	CoverURL         string        `json:"cover_url"`
	Duration         int           `json:"duration"`
	MagnetsCount     int           `json:"magnets_count"`
	CanPlay          bool          `json:"can_play"`
	HasPreviewVideo  bool          `json:"has_preview_video"`
	HasPreviewImages bool          `json:"has_preview_images"`
	ReleaseDate      string        `json:"release_date"`
	PreviewImages    []string      `json:"preview_images"`
}

// ToMovie converts a searchMovie to the domain Movie type.
func (sm searchMovie) ToMovie() *domain.Movie {
	return &domain.Movie{
		ID:               sm.ID,
		Number:           sm.Number,
		Title:            sm.Title,
		OriginTitle:      sm.OriginTitle,
		ThumbURL:         sm.ThumbURL,
		CoverURL:         sm.CoverURL,
		Duration:         sm.Duration,
		MagnetsCount:     sm.MagnetsCount,
		CanPlay:          sm.CanPlay,
		HasPreviewVideo:  sm.HasPreviewVideo,
		HasPreviewImages: sm.HasPreviewImages,
		ReleaseDate:      sm.ReleaseDate,
		PreviewImages:    sm.PreviewImages,
	}
}

// detailMovie represents a movie returned by /api/v4/movies/{id}.
type detailMovie struct {
	ID               string         `json:"id"`
	Number           string         `json:"number"`
	Title            string         `json:"title"`
	OriginTitle      string         `json:"origin_title"`
	ThumbURL         string         `json:"thumb_url"`
	CoverURL         string         `json:"cover_url"`
	Duration         int            `json:"duration"`
	Score            float64        `json:"score"`
	ReleaseDate      string         `json:"release_date"`
	MagnetsCount     int            `json:"magnets_count"`
	CanPlay          bool           `json:"can_play"`
	HasPreviewVideo  bool           `json:"has_preview_video"`
	HasPreviewImages bool           `json:"has_preview_images"`
	PreviewImages    []string       `json:"preview_images"`
	PreviewVideoURL  string         `json:"preview_video_url"`
	Summary          string         `json:"summary"`
	Actors           jsonStringSlice `json:"actors"`
	Tags             jsonStringSlice `json:"tags"`
	DirectorName     string         `json:"director_name"`
	MakerName        string         `json:"maker_name"`
	PublisherName    string         `json:"publisher_name"`
	SeriesName       string         `json:"series_name"`
}

// ToMovie converts a detailMovie to the domain Movie type.
func (dm detailMovie) ToMovie() *domain.Movie {
	return &domain.Movie{
		ID:               dm.ID,
		Number:           dm.Number,
		Title:            dm.Title,
		OriginTitle:      dm.OriginTitle,
		ThumbURL:         dm.ThumbURL,
		CoverURL:         dm.CoverURL,
		Duration:         dm.Duration,
		Score:            dm.Score,
		ReleaseDate:      dm.ReleaseDate,
		MagnetsCount:     dm.MagnetsCount,
		CanPlay:          dm.CanPlay,
		HasPreviewVideo:  dm.HasPreviewVideo,
		HasPreviewImages: dm.HasPreviewImages,
		PreviewImages:    dm.PreviewImages,
		PreviewVideoURL:  dm.PreviewVideoURL,
		Summary:          dm.Summary,
		Actors:           []string(dm.Actors),
		Tags:             []string(dm.Tags),
		DirectorName:     dm.DirectorName,
		MakerName:        dm.MakerName,
		PublisherName:    dm.PublisherName,
		SeriesName:       dm.SeriesName,
	}
}

// jsonStringSlice handles JSON arrays that may contain strings directly
// or objects with a "name" field (like actors/tags from JavDB API).
type jsonStringSlice []string

// UnmarshalJSON implements json.Unmarshaler for jsonStringSlice.
// It tries []string first, then falls back to extracting "name" from []object.
func (s *jsonStringSlice) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as []string first.
	var strs []string
	if err := json.Unmarshal(data, &strs); err == nil {
		*s = strs
		return nil
	}

	// Fall back: try []object with "name" field.
	var objs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &objs); err != nil {
		return err
	}
	result := make([]string, len(objs))
	for i, o := range objs {
		result[i] = o.Name
	}
	*s = jsonStringSlice(result)
	return nil
}
