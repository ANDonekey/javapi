package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/henry/javapi/internal/aggregator"
	"github.com/henry/javapi/internal/cache"
	"github.com/henry/javapi/internal/config"
	"github.com/henry/javapi/internal/domain"
	"github.com/henry/javapi/internal/scraper"
	"github.com/henry/javapi/internal/handler"
	"github.com/henry/javapi/internal/javdb"
	myMiddleware "github.com/henry/javapi/internal/middleware"

	// Register all scrapers via init()
	_ "github.com/henry/javapi/internal/scraper/av01"
	_ "github.com/henry/javapi/internal/scraper/jable"
	_ "github.com/henry/javapi/internal/scraper/javgg"
	_ "github.com/henry/javapi/internal/scraper/missav"
	_ "github.com/henry/javapi/internal/scraper/sevenmmtv"

	// Register embed extractors via init()
	_ "github.com/henry/javapi/internal/embed"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	scraper.ApplyConfig(cfg.Scrapers.Sites)

	javdbClient := javdb.NewClient(cfg.JavDB.BaseURL, cfg.JavDB.Middle, cfg.JavDB.Suffix)

	var cacheImpl domain.Cache = cache.NewMemoryCache(time.Duration(cfg.Cache.MemoryTTLSeconds) * time.Second)

	if cfg.Cache.PostgresEnabled && cfg.Cache.PostgresURL != "" {
		pgCache, err := cache.NewPostgresCache(context.Background(), cfg.Cache.PostgresURL)
		if err != nil {
			log.Printf("PostgreSQL cache unavailable: %v (falling back to memory-only)", err)
		} else {
			defer pgCache.Close()
			cacheImpl = pgCache
			log.Println("PostgreSQL cache enabled")
		}
	}

	// Aggregator
	svc := aggregator.NewService(javdbClient, cacheImpl, cfg.Scrapers.MaxConcurrent)

	searchH := handler.NewSearchHandler(svc)

	r := chi.NewRouter()
	r.Use(myMiddleware.Recovery)
	r.Use(myMiddleware.Logging)
	r.Use(myMiddleware.CORS)
	r.Use(middleware.RequestID)

	r.Get("/api/health", handler.Health)

	r.Group(func(r chi.Router) {
		r.Use(myMiddleware.Auth(cfg.Auth.APIKeys, "/api/health"))
		r.Get("/api/v1/search", searchH.Search)
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
	}

	go func() {
		log.Printf("javapi server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeoutSec)*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("stopped")
}
