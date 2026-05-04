package aggregator

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/embed"
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

type timingAccum struct {
	mu         sync.Mutex
	javdbMs    int64
	scraperTim []domain.ScraperTiming
}

func (t *timingAccum) addJavDB(ms int64) {
	t.mu.Lock()
	t.javdbMs = ms
	t.mu.Unlock()
}

func (t *timingAccum) addScraper(st domain.ScraperTiming) {
	t.mu.Lock()
	t.scraperTim = append(t.scraperTim, st)
	t.mu.Unlock()
}

func (s *Service) Aggregate(ctx context.Context, code string) (*domain.SearchResponse, error) {
	start := time.Now()
	cacheKey := scraper.NormalizeCode(code)

	cacheStart := time.Now()
	if os.Getenv("CACHE_DISABLED") == "" {
		if resp, ok := s.cache.Get(ctx, cacheKey); ok {
			cacheMs := time.Since(cacheStart).Milliseconds()
			resp.Cache.Hit = true
			resp.TookMs = time.Since(start).Milliseconds()
			if resp.Timing == nil {
				resp.Timing = &domain.TimingInfo{CacheMs: cacheMs, TotalMs: resp.TookMs}
			}
			return resp, nil
		}
	}

	sem := semaphore.NewWeighted(s.maxConcurrent)
	var wg sync.WaitGroup
	ta := &timingAccum{}

	var movie *domain.Movie
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sem.Acquire(ctx, 1); err != nil {
			return
		}
		defer sem.Release(1)
		javdbStart := time.Now()
		m, err := s.javdb.Search(ctx, code)
		if err != nil {
			ta.addJavDB(time.Since(javdbStart).Milliseconds())
			log.Printf("aggregator: JavDB search error: %v", err)
			return
		}
		if m != nil && m.ID != "" {
			if detail, err := s.javdb.GetMovie(ctx, m.ID); err == nil && detail != nil {
				movie = detail
				ta.addJavDB(time.Since(javdbStart).Milliseconds())
				return
			}
		}
		movie = m
		ta.addJavDB(time.Since(javdbStart).Milliseconds())
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
			scraperStart := time.Now()
			results, err := scraper.SafeSearch(ctx, scr, scr.FormatCode(code))
			elapsed := time.Since(scraperStart).Milliseconds()
			status := string(domain.StatusSuccess)
			if err != nil {
				status = string(domain.StatusError)
			} else if len(results) > 0 {
				status = string(results[0].Status)
			}
			ta.addScraper(domain.ScraperTiming{
				Name:    scr.Name(),
				FetchMs: elapsed,
				Status:  status,
			})
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

	embedStart := time.Now()
	for i := range videos {
		for j := range videos[i].VideoSources {
			resolved := embed.ResolveEmbed(ctx, videos[i].VideoSources[j])
			if resolved.URL != videos[i].VideoSources[j].URL {
				videos[i].VideoSources[j] = resolved
			}
		}
	}
	embedMs := time.Since(embedStart).Milliseconds()

	if movie == nil && len(videos) == 0 {
		return nil, fmt.Errorf("all sources failed for code %q", code)
	}

	cacheMs := time.Since(cacheStart).Milliseconds()
	totalMs := time.Since(start).Milliseconds()

	resp := &domain.SearchResponse{
		Code:   code,
		Movie:  movie,
		Videos: videos,
		Cache:  domain.CacheInfo{Hit: false},
		TookMs: totalMs,
		Timing: &domain.TimingInfo{
			TotalMs:  totalMs,
			JavDBMs:  ta.javdbMs,
			Scrapers: ta.scraperTim,
			EmbedMs:  embedMs,
			CacheMs:  cacheMs,
		},
	}
	if os.Getenv("CACHE_DISABLED") == "" {
		_ = s.cache.Set(ctx, cacheKey, resp, 5*time.Minute)
	}
	return resp, nil
}
