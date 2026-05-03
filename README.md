# JAV Search API

Aggregated JAV movie search — metadata from JavDB API + embed video links from 8 video hosting sites.

[![Go](https://img.shields.io/badge/Go-1.26.2-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## Overview

Accepts a JAV code (e.g. `ABC-123`) and returns:
- **Movie metadata** — title, actors, release date, cover art, score, tags, previews — from [JavDB](https://jdforrepam.com) API
- **Video embed links** — playable video URLs scraped from 8 video sites concurrently, with multi-version support (original, Chinese subtitle, mosaic-reduced)
- **Cached responses** — in-memory cache (configurable TTL) + optional PostgreSQL persistence

Designed for the **Render.com free tier** (< 512 MB RAM, database-less by default).

## Quick Start

### Prerequisites
- Go 1.26+
- (Optional) PostgreSQL for persistent cache
- (Optional) Docker

### Install & Run

```bash
# Clone
cd JAVprovide

# Install dependencies
go mod download

# Build
go build -o javapi ./cmd/api

# Run (database-less mode — in-memory cache only)
GOMEMLIMIT=400MiB ./javapi

# Or run directly without building
GOMEMLIMIT=400MiB go run ./cmd/api
```

The server starts on `:8080`:

```
2025/05/03 12:00:00 javapi server starting on :8080
```

## API Usage

### Health Check

No authentication required.

```bash
curl http://localhost:8080/api/health
# {"status":"ok"}
```

### Authentication

All other endpoints require an `X-API-Key` header.

Start the server with an API key:

```bash
AUTH_API_KEYS=my-secret-key GOMEMLIMIT=400MiB go run ./cmd/api
```

### Search By Code

```bash
curl -H "X-API-Key: my-secret-key" \
  "http://localhost:8080/api/v1/search?code=ABC-123"
```

### Response Format

```json
{
  "code": "ABC-123",
  "movie": {
    "id": "abc123",
    "number": "ABC-123",
    "title": "Movie Title",
    "origin_title": "オリジナルタイトル",
    "thumb_url": "https://example.com/thumb.jpg",
    "cover_url": "https://example.com/cover.jpg",
    "duration": 120,
    "score": 8.5,
    "release_date": "2025-01-01",
    "magnets_count": 42,
    "can_play": true,
    "has_preview_video": true,
    "has_preview_images": true,
    "preview_images": ["https://example.com/pv1.jpg"],
    "preview_video_url": "https://example.com/preview.mp4",
    "summary": "A captivating story...",
    "actors": ["Actor A", "Actor B"],
    "tags": ["Tag1", "Tag2"],
    "director_name": "Director Name",
    "maker_name": "Maker Name",
    "publisher_name": "Publisher Name",
    "series_name": "Series Name"
  },
  "videos": [
    {
      "siteName": "MISSAV",
      "status": "success",
      "version": "original",
      "pageUrl": "https://missav.ws/ABC-123/",
      "videoSources": [
        {"url": "https://cdn.example.com/abc123.mp4", "type": "video/mp4"}
      ],
      "subtitle": false,
      "leak": false
    },
    {
      "siteName": "HAYAV",
      "status": "success",
      "version": "cnsub",
      "label": "Chinese Sub",
      "pageUrl": "https://hayav.com/video/ABC-123/",
      "videoSources": [
        {"url": "https://cdn.example.com/abc123_cn.m3u8", "type": "application/x-mpegURL"}
      ],
      "subtitle": true,
      "leak": false
    }
  ],
  "cache": {
    "hit": false,
    "age": 0
  },
  "took_ms": 8234
}
```

### Error Responses

```json
// 401 — Missing or invalid API key
{"error":"unauthorized","message":"invalid or missing API key"}

// 400 — Missing search code
{"error":"bad_request","message":"code parameter is required"}

// 502 — All upstream sources failed
{"error":"search_failed","message":"all search sources failed"}
```

## Configuration

Configuration is loaded from `configs/config.yaml` with environment variable overrides.

| Env Var | YAML Path | Default | Description |
|---------|-----------|---------|-------------|
| `AUTH_API_KEYS` | `auth.api_keys` | (required) | Comma-separated API keys |
| `JAVDB_MIDDLE` | `javdb.middle` | `lpw6vgqzsp` | jdsignature middle segment |
| `JAVDB_SUFFIX` | `javdb.suffix` | (long hex) | jdsignature suffix for MD5 hash |
| `CACHE_POSTGRES_URL` | `cache.postgres_url` | (empty) | PostgreSQL DSN (disabled if empty) |
| `CACHE_POSTGRES_ENABLED` | `cache.postgres_enabled` | `false` | Enable PostgreSQL persistent cache |
| `CACHE_MEMORY_TTL_SECONDS` | `cache.memory_ttl_seconds` | `300` | In-memory cache TTL in seconds |
| `SCRAPERS_MAX_CONCURRENT` | `scrapers.max_concurrent` | `6` | Max parallel scraper goroutines |
| — | `scrapers.timeout_sec` | `8` | Per-scraper HTTP timeout |
| — | `scrapers.rate_limit_delay_ms` | `500` | Delay between scraper requests (ms) |
| — | `scrapers.user_agent` | (Chrome 124) | HTTP User-Agent header |
| `GOMEMLIMIT` | `render.gomemlimit` | `400MiB` | Go GC memory limit (Render.com) |

### Database-Less Mode

By default, the server runs **without PostgreSQL** — only the in-memory cache is active. Results are cached for 300 seconds (configurable). To enable PostgreSQL:

```bash
CACHE_POSTGRES_ENABLED=true CACHE_POSTGRES_URL="postgres://user:pass@localhost:5432/javapi?sslmode=disable" go run ./cmd/api
```

### Per-Site Proxy Configuration

Each scraper can be configured with an individual BrightData-compatible proxy:

```yaml
scrapers:
  sites:
    - name: "MISSAV"
      enabled: true
      proxy_url: "http://user:pass@brd.superproxy.io:22225"
      proxy_enabled: true
      timeout_sec: 10
```

## Supported Video Sites

| Site | Type | Multi-Version | CF Bypass | Status |
|------|------|---------------|-----------|--------|
| [MISSAV](https://missav.ws) | GET — direct URL | ✅ original / cnsub / mosaic_reduce | ✅ | ✅ Implemented |
| [Jable](https://jable.tv) | GET — search + parse | ✅ original / cnsub | ✅ | ✅ Implemented |
| [JAVMENU](https://javmenu.com) | GET — direct URL | ✅ original / cnsub / mosaic_reduce | ✅ | ✅ Implemented |
| [HAYAV](https://hayav.com) | GET — direct URL | ✅ original / cnsub | ✅ | ✅ Implemented |
| [Supjav](https://supjav.com) | GET — search + parse | ✅ planned | ⏳ Planned | ⏳ Not yet |
| [javgg](https://javgg.net) | GET — search + parse | ✅ planned | ⏳ Planned | ⏳ Not yet |
| [AV01](https://www.av01.media) | POST — JSON API | ✅ planned | ⏳ Planned | ⏳ Not yet |
| [7mmtv](https://7mmtv.sx) | POST — form-urlencoded | ✅ planned | ⏳ Planned | ⏳ Not yet |

### Metadata Sources (JavDB)

| Source | Type | Description |
|--------|------|-------------|
| JavDB API | REST API | Movie metadata: title, actors, cover, score, tags, previews |

## Docker

> **Note**: Dockerfile and docker-compose.yml are planned but not yet created.

```bash
# Build image
docker build -t javapi .

# Run (database-less)
docker run -p 8080:8080 -e AUTH_API_KEYS=my-key -e GOMEMLIMIT=400MiB javapi

# With PostgreSQL (when docker-compose.yml is ready)
docker compose --profile pg up

# Stop
docker compose --profile pg down
```

## Development

```bash
# Run all tests
go test ./... -v -count=1

# Run with race detector
go test -race ./...

# Run specific package tests
go test ./internal/scraper/missav/... -v

# Using Makefile
make build      # Build binary
make test       # Run all tests
make run        # go run ./cmd/api
make clean      # Remove built binary
```

### Code Coverage

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Project Structure

```
├── cmd/api/main.go                 — Entry point, chi server with graceful shutdown
├── configs/config.yaml             — Default configuration template
├── internal/
│   ├── config/                     — YAML + env var configuration loading
│   ├── domain/                     — Core types: Movie, VideoResult, SearchResponse, Scraper interface
│   ├── javdb/                      — JavDB API client (jdsignature auth, search + movie detail)
│   ├── scraper/                    — Scraper registry + site implementations
│   │   ├── registry.go             — Self-registering plugin pattern
│   │   ├── code.go                 — Code normalization utilities
│   │   ├── safe.go                 — Panic-recovery wrapper
│   │   ├── cf.go                   — Cloudflare bypass pre-test
│   │   ├── missav/                 — MISSAV scraper
│   │   ├── jable/                  — Jable scraper
│   │   ├── javmenu/                — JAVMENU scraper
│   │   └── hayav/                  — HAYAV scraper
│   ├── aggregator/                 — Parallel search orchestration + cache (planned)
│   ├── cache/                      — In-memory (go-cache) + optional PostgreSQL
│   ├── middleware/                  — Auth, logging, recovery, CORS
│   └── handler/                    — HTTP handlers (health, search)
├── Makefile                        — Build, test, docker targets
└── go.mod                          — Go module (github.com/henry/javapi)
```

## Architecture

```
Client → X-API-Key → chi Router → Auth Middleware → Search Handler
                                                       │
                                                       ▼
                                              Aggregator Service
                                              ┌─────────────────┐
                                              │  1. Check Cache  │──→ (cache hit) → return
                                              │  2. JavDB API    │──→ movie metadata
                                              │  3. 8 Scrapers   │──→ video links (concurrent)
                                              │     (semaphore   │
                                              │      cap: 6)     │
                                              │  4. Cache Result │
                                              └─────────────────┘
```

- **Concurrency**: All scrapers run in parallel via goroutines, capped by `semaphore.Weighted` (default 6 for Render free tier)
- **Graceful degradation**: Failed scrapers return error status in the response; the API succeeds as long as at least one source responds
- **Cache**: In-memory via `patrickmn/go-cache` (default 300s TTL); optional PostgreSQL for persistence across restarts
- **Cloudflare bypass**: Sites behind Cloudflare are accessed via `cloudscraper_go` + `CycleTLS` with Firefox TLS fingerprinting; pre-tested at startup to verify each site is reachable

## Platform Support

- ✅ Linux (amd64 / arm64)
- ✅ macOS (amd64 / arm64)
- ✅ Render.com free tier (< 512 MB RAM, GOMEMLIMIT=400MiB)
- ✅ Docker (database-less default)

## Dependencies

- [chi/v5](https://github.com/go-chi/chi) — HTTP router with middleware chain
- [go-resty/v2](https://github.com/go-resty/resty) — HTTP client with retry + timeout
- [goquery](https://github.com/PuerkitoBio/goquery) — jQuery-style HTML parsing
- [go-cache](https://github.com/patrickmn/go-cache) — In-memory cache with TTL
- [lib/pq](https://github.com/lib/pq) — PostgreSQL driver
- [testify](https://github.com/stretchr/testify) — Test framework (assert + mock)
- [cloudscraper_go](https://github.com/RomainMichau/cloudscraper_go) + [CycleTLS](https://github.com/RomainMichau/CycleTLS) — Cloudflare bypass via TLS fingerprinting

## Related

- [JavDB API docs](JavDB%20API/) — API documentation and reference materials
