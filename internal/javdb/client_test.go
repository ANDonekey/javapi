package javdb

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSignature(t *testing.T) {
	client := NewClient("https://example.com", "lpw6vgqzsp", "71cf27bb3c0bcdf207b64abecddc970098c7421ee7203b9cdae54478478a199e7d5a6e1a57691123c1a931c057842fb73ba3b3c83bcd69c17ccf174081e3d8aa")

	sig := client.generateSignature()

	parts := strings.Split(sig, ".")
	require.Len(t, parts, 3, "signature should have 3 dot-separated parts")

	ts := parts[0]
	middle := parts[1]
	hash := parts[2]

	assert.Equal(t, "lpw6vgqzsp", middle)

	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(ts+client.suffix)))
	assert.Equal(t, expectedHash, hash)
}

func TestNewClient(t *testing.T) {
	client := NewClient("https://jdforrepam.com", "mid", "suf")

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "https://jdforrepam.com", client.baseURL)
	assert.Equal(t, "mid", client.middle)
	assert.Equal(t, "suf", client.suffix)
}

func TestSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/search", r.URL.Path)
		assert.Equal(t, "IPZZ-001", r.URL.Query().Get("q"))
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.NotEmpty(t, r.Header.Get("jdsignature"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 1,
			"action": null,
			"message": null,
			"data": {
				"movies": [{
					"id": "abc123",
					"number": "IPZZ-001",
					"title": "Test Movie",
					"origin_title": "テスト映画",
					"thumb_url": "https://example.com/thumb.jpg",
					"cover_url": "https://example.com/cover.jpg",
					"duration": 120,
					"magnets_count": 5,
					"can_play": true,
					"has_preview_video": true,
					"has_preview_images": true,
					"release_date": "2024-01-01",
					"preview_images": ["img1.jpg", "img2.jpg"]
				}]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.Search(ctx, "IPZZ-001")

	require.NoError(t, err)
	require.NotNil(t, movie)
	assert.Equal(t, "abc123", movie.ID)
	assert.Equal(t, "IPZZ-001", movie.Number)
	assert.Equal(t, "Test Movie", movie.Title)
	assert.Equal(t, "テスト映画", movie.OriginTitle)
	assert.Equal(t, "https://example.com/thumb.jpg", movie.ThumbURL)
	assert.Equal(t, "https://example.com/cover.jpg", movie.CoverURL)
	assert.Equal(t, 120, movie.Duration)
	assert.Equal(t, 5, movie.MagnetsCount)
	assert.True(t, movie.CanPlay)
	assert.True(t, movie.HasPreviewVideo)
	assert.True(t, movie.HasPreviewImages)
	assert.Equal(t, "2024-01-01", movie.ReleaseDate)
	assert.Equal(t, []string{"img1.jpg", "img2.jpg"}, movie.PreviewImages)
}

func TestSearch_EmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 1,
			"action": null,
			"message": null,
			"data": {
				"movies": []
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.Search(ctx, "NONEXIST")

	assert.NoError(t, err)
	assert.Nil(t, movie, "empty movies should return nil movie, not an error")
}

func TestSearch_InvalidSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 0,
			"action": "InvalidSignature",
			"message": "签名无效",
			"data": null
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.Search(ctx, "ABC-123")

	assert.Error(t, err)
	assert.Nil(t, movie)
	assert.Contains(t, err.Error(), "signature")
}

func TestSearch_ParameterInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 0,
			"action": "ParameterInvalid",
			"message": "参数不能为空: q",
			"data": null
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.Search(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, movie)
	assert.Contains(t, err.Error(), "parameter invalid")
}

func TestSearch_JWTVerificationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 0,
			"action": "JWTVerificationError",
			"message": "需要登录",
			"data": null
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.Search(ctx, "ABC-123")

	assert.Error(t, err)
	assert.Nil(t, movie)
	assert.Contains(t, err.Error(), "authentication required")
}

func TestSearch_HTTP404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.Search(ctx, "ABC-123")

	assert.Error(t, err)
	assert.Nil(t, movie)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestSearch_HTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.Search(ctx, "ABC-123")

	assert.Error(t, err)
	assert.Nil(t, movie)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestSearch_NetworkTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	movie, err := client.Search(ctx, "ABC-123")

	assert.Error(t, err)
	assert.Nil(t, movie)
}

func TestGetMovie_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v4/movies/abc123", r.URL.Path)
		assert.NotEmpty(t, r.Header.Get("jdsignature"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 1,
			"action": null,
			"message": null,
			"data": {
				"movie": {
					"id": "abc123",
					"number": "IPZZ-001",
					"title": "Full Detail Movie",
					"origin_title": "詳細映画",
					"thumb_url": "https://example.com/thumb.jpg",
					"cover_url": "https://example.com/cover.jpg",
					"duration": 150,
					"score": 4.5,
					"release_date": "2024-06-15",
					"magnets_count": 10,
					"can_play": true,
					"has_preview_video": true,
					"has_preview_images": false,
					"preview_images": ["prev1.jpg"],
					"preview_video_url": "https://example.com/preview.mp4",
					"summary": "A great movie summary",
					"actors": ["Actor One", "Actor Two"],
					"tags": ["tag1", "tag2"],
					"director_name": "Director X",
					"maker_name": "Maker Y",
					"publisher_name": "Publisher Z",
					"series_name": "Series W"
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.GetMovie(ctx, "abc123")

	require.NoError(t, err)
	require.NotNil(t, movie)
	assert.Equal(t, "abc123", movie.ID)
	assert.Equal(t, "IPZZ-001", movie.Number)
	assert.Equal(t, "Full Detail Movie", movie.Title)
	assert.Equal(t, "詳細映画", movie.OriginTitle)
	assert.Equal(t, 150, movie.Duration)
	assert.Equal(t, 4.5, movie.Score)
	assert.Equal(t, "2024-06-15", movie.ReleaseDate)
	assert.Equal(t, 10, movie.MagnetsCount)
	assert.True(t, movie.CanPlay)
	assert.True(t, movie.HasPreviewVideo)
	assert.False(t, movie.HasPreviewImages)
	assert.Equal(t, []string{"prev1.jpg"}, movie.PreviewImages)
	assert.Equal(t, "https://example.com/preview.mp4", movie.PreviewVideoURL)
	assert.Equal(t, "A great movie summary", movie.Summary)
	assert.Equal(t, []string{"Actor One", "Actor Two"}, movie.Actors)
	assert.Equal(t, []string{"tag1", "tag2"}, movie.Tags)
	assert.Equal(t, "Director X", movie.DirectorName)
	assert.Equal(t, "Maker Y", movie.MakerName)
	assert.Equal(t, "Publisher Z", movie.PublisherName)
	assert.Equal(t, "Series W", movie.SeriesName)
}

func TestGetMovie_ActorsAsObjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 1,
			"action": null,
			"message": null,
			"data": {
				"movie": {
					"id": "xyz789",
					"number": "ABC-123",
					"title": "Actor Objects Test",
					"origin_title": "",
					"thumb_url": "",
					"cover_url": "",
					"duration": 90,
					"score": 3.2,
					"release_date": "2023-01-01",
					"magnets_count": 0,
					"can_play": false,
					"has_preview_video": false,
					"has_preview_images": false,
					"preview_images": [],
					"preview_video_url": "",
					"summary": "",
					"actors": [
						{"id": "a1", "name": "Actress A"},
						{"id": "a2", "name": "Actress B"}
					],
					"tags": [
						{"id": "t1", "name": "Tag X"},
						{"id": "t2", "name": "Tag Y"}
					],
					"director_name": "",
					"maker_name": "",
					"publisher_name": "",
					"series_name": ""
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.GetMovie(ctx, "xyz789")

	require.NoError(t, err)
	require.NotNil(t, movie)
	assert.Equal(t, []string{"Actress A", "Actress B"}, movie.Actors)
	assert.Equal(t, []string{"Tag X", "Tag Y"}, movie.Tags)
}

func TestGetMovie_HTTP404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.GetMovie(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, movie)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestGetMovie_InvalidSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": 0,
			"action": "InvalidSignature",
			"message": "签名无效",
			"data": null
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "mid", "suf")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	movie, err := client.GetMovie(ctx, "abc123")

	assert.Error(t, err)
	assert.Nil(t, movie)
	assert.Contains(t, err.Error(), "signature")
}
