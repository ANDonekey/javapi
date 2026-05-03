package aggregator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
)

type JavDBClient interface {
	Search(ctx context.Context, code string) (*domain.Movie, error)
	GetMovie(ctx context.Context, movieID string) (*domain.Movie, error)
}

type Service struct {
	javdb         JavDBClient
	cache         domain.Cache
	maxConcurrent int64
}

func NewService(javdb JavDBClient, cache domain.Cache, maxConcurrent int) *Service {
	if maxConcurrent <= 0 {
		maxConcurrent = 6
	}
	return &Service{javdb: javdb, cache: cache, maxConcurrent: int64(maxConcurrent)}
}

func (s *Service) Aggregate(ctx context.Context, code string) (*domain.SearchResponse, error) {
	start := time.Now()
	cacheKey := scraper.NormalizeCode(code)

	if resp, ok := s.cache.Get(ctx, cacheKey); ok {
		resp.Cache.Hit = true
		resp.TookMs = time.Since(start).Milliseconds()
		return resp, nil
	}

	sem := semaphore.NewWeighted(s.maxConcurrent)
	var wg sync.WaitGroup

	var movie *domain.Movie
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sem.Acquire(ctx, 1); err != nil {
			return
		}
		defer sem.Release(1)
		m, err := s.javdb.Search(ctx, code)
		if err != nil {
			log.Printf("aggregator: JavDB search error: %v", err)
			return
		}
		if m != nil && m.ID != "" {
			if detail, err := s.javdb.GetMovie(ctx, m.ID); err == nil && detail != nil {
				movie = detail
				return
			}
		}
		movie = m
	}()

	scrapersList := scraper.GetEnabled()
	var mu sync.Mutex
	videos := make([]domain.VideoResult, 0)

	for _, scr := range scrapersList {
		scr := scr
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			defer sem.Release(1)
			results, err := scraper.SafeSearch(ctx, scr, scr.FormatCode(code))
			if err != nil {
				mu.Lock()
				videos = append(videos, domain.VideoResult{SiteName: scr.Name(), Status: domain.StatusError, Version: domain.VersionOriginal, Error: err.Error()})
				mu.Unlock()
				return
			}
			mu.Lock()
			videos = append(videos, results...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	if movie == nil && len(videos) == 0 {
		return nil, fmt.Errorf("all sources failed for code %q", code)
	}

	resp := &domain.SearchResponse{
		Code: code, Movie: movie, Videos: videos,
		Cache:  domain.CacheInfo{Hit: false},
		TookMs: time.Since(start).Milliseconds(),
	}
	_ = s.cache.Set(ctx, cacheKey, resp, 5*time.Minute)
	return resp, nil
}
