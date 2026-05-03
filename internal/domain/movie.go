package domain

// Movie represents a JAV movie with full metadata from JavDB.
type Movie struct {
	ID               string   `json:"id"`
	Number           string   `json:"number"`
	Title            string   `json:"title"`
	OriginTitle      string   `json:"origin_title,omitempty"`
	ThumbURL         string   `json:"thumb_url,omitempty"`
	CoverURL         string   `json:"cover_url,omitempty"`
	Duration         int      `json:"duration,omitempty"`
	Score            float64  `json:"score,omitempty"`
	ReleaseDate      string   `json:"release_date,omitempty"`
	MagnetsCount     int      `json:"magnets_count,omitempty"`
	CanPlay          bool     `json:"can_play"`
	HasPreviewVideo  bool     `json:"has_preview_video"`
	HasPreviewImages bool     `json:"has_preview_images"`
	PreviewImages    []string `json:"preview_images,omitempty"`
	PreviewVideoURL  string   `json:"preview_video_url,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Actors           []string `json:"actors,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	DirectorName     string   `json:"director_name,omitempty"`
	MakerName        string   `json:"maker_name,omitempty"`
	PublisherName    string   `json:"publisher_name,omitempty"`
	SeriesName       string   `json:"series_name,omitempty"`
}
