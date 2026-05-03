# JAV Search Aggregator API

## TL;DR

> **Quick Summary**: Build a lightweight Go REST API (targeting Render.com free tier) that accepts a JAV code and returns aggregated movie metadata from the JavDB API plus embed video links scraped from 8 video hosting sites — with Cloudflare bypass via cloudscraper_go, per-site proxy support (BrightData), optional database-less mode, and multi-version result support.
>
> **Deliverables**:
> - Go REST API server with chi router, API key authentication
> - JavDB API client (jdsignature auth, search + movie detail)
> - 8 site-specific video link scrapers (MISSAV, Jable, JAVMENU, HAYAV, Supjav, javgg, AV01, 7mmtv)
> - Cloudflare bypass via cloudscraper_go integration (pre-test each site, skip if CF fails)
> - Per-site proxy configuration (BrightData compatible, individual proxy URL + enable toggle)
> - Result aggregation layer with graceful degradation + multi-version support (original, cnsub, mosaic-reduced)
> - Two-tier caching: in-memory TTL + optional PostgreSQL persistence (database-less mode supported)
> - Docker + standalone binary deployment
> - Full test suite (go test + testify, TDD)
>
> **Estimated Effort**: Large (22 tasks, 3 waves + final verification)
> **Parallel Execution**: YES — 3 waves, max 10 concurrent
> **Critical Path**: Project scaffolding → Types/Interfaces → Core HTTP + Cache → JavDB Client → CF Bypass Test → Aggregator → Scrapers (parallel) → Integration
> **Render.com Free Tier**: ✅ Viable — <512MB RAM, <0.1 CPU, cold-start tolerant, PostgreSQL optional

---

## Context

### Original Request
> 制作一个api，用于通过用番号从多个网站里搜索，获取对应影片的资料，获取用于播放嵌入视频链接。有关搜索，参考 javjump userscript 的站点搜索逻辑。影片的资料由JavDB API提供。采用 Golang。

### Interview Summary

**Key Discussions**:
- **API Use Case**: Frontend/Web App backend — REST API returning JSON
- **Tech Stack**: Go + chi router + go-resty HTTP client + colly/goquery for scraping + cloudscraper_go for CF bypass
- **JavDB Integration**: Using `jdforrepam.com` API with `jdsignature` header auth (key material from existing docs)
- **Video Link Sources**: Direct scraping from 8 recommended sites (not Cloudflare Workers)
- **8 Recommended Sites** (from javjump's 28 total, after CF feasibility filtering): MISSAV, Jable, JAVMENU, HAYAV, Supjav, javgg, AV01, 7mmtv
  - 4 GET sites (direct URL): JAVMENU, HAYAV, MISSAV, Jable
  - 2 Parser sites (search+parse): Supjav, javgg
  - 2 Special sites (POST): AV01 (JSON API), 7mmtv (form-urlencoded)
- **Removed sites**: javtrailers (unstable), FANZA 動画 (out of scope), JavBus (CF issues), JAVLib (direct link, not scraper)
- **Error Handling**: Graceful degradation — partial results returned, failed sites marked as errors
- **Caching**: Configurable in-memory TTL + optional PostgreSQL persistence (database-less mode supported)
- **Testing**: Automated (go test + testify), TDD approach
- **Deployment**: Both standalone binary + Docker, config via env vars + YAML, target Render.com free tier
- **Authentication**: Simple API Key via `X-API-Key` header
- **Cloudflare**: Pre-test each site with cloudscraper_go; skip sites where CF cannot be bypassed
- **Proxy**: Per-site proxy configuration (BrightData compatible), individual proxy URL + enable/disable per site
- **Multi-version**: Each site may return multiple video versions (original, cnsub, mosaic-reduced) — aggregated in results

### Render.com Free Tier Resource Assessment

| Resource | Free Tier Limit | Project Estimate | Verdict |
|----------|----------------|-----------------|---------|
| **RAM** | 512 MB | Go binary ~15MB, runtime ~50-100MB (peak with 8 concurrent scrapers), well within limit | ✅ Pass |
| **CPU** | 0.1 shared | Scraping is I/O-bound (network wait), not CPU-bound. Go goroutines are lightweight. | ✅ Pass |
| **Disk** | Ephemeral only | Binary ~15MB, no persistent disk needed (PostgreSQL separate) | ✅ Pass |
| **Runtime** | 750 hrs/month | 24×31 = 744 hrs, so continuous uptime possible | ✅ Pass |
| **Cold Start** | Spins down after 15min inactivity | Acceptable — first request after idle will be slower (cache miss + cold start) | ⚠️ Acceptable |
| **PostgreSQL** | 1 GB storage, 97 max connections | Search cache small (<10KB per entry, even 100K entries = 1GB). Single connection. | ✅ Pass |
| **Concurrency Limit** | No hard limit, but 512MB RAM caps goroutines | Cap concurrent scrapers to 6 (down from 12) to stay within RAM budget | ⚠️ Mitigated |
| **Network** | Outbound only, no rate limit documented | Scraper outbound requests are standard HTTP — no issue | ✅ Pass |

**Recommendations for Render Free Tier**:
1. Reduce max concurrent scrapers from 12 → **6** in the semaphore pool
2. Make PostgreSQL **optional** (off by default for free tier without DB)
3. Set scraper timeout to **8s** (down from 10s) to release resources faster
4. Use `GOMEMLIMIT=400MiB` to trigger GC before hitting the 512MB hard limit
5. Accept cold starts — first request after idle will have ~30s startup + scrape time

### Research Findings
- **javjump**: 28 total sites in 4 categories, code normalization pattern `lowercase + strip [-_ ]`, site-specific formatters needed for FANZA/Jable/fcjav
- **JavDB API**: /api/v2/search (public signed), /api/v4/movies/{id} (public signed), jdsignature format: `{timestamp}.lpw6vgqzsp.{md5(timestamp + suffix)}`
- **Go Patterns**: chi (router), goquery/colly (parsing), go-resty (HTTP client), errgroup + semaphore (concurrency), go-cache/bigcache (memory cache)
- **Reference Architecture**: javinizer-go — scraper registry, self-registering init() plugins, safeSearch() panic wrapper, slot-rollback rate limiter

### Metis Review

**Identified Gaps** (addressed):
- **Concurrency model**: Use semaphore.Weighted for global pool + errgroup per-request (not errgroup alone)
- **Scraper registration**: Adopt `init()` plugin pattern — each site scraper is a self-contained, self-registering package
- **Panic isolation**: MUST wrap every scraper call in `safeSearch()` — one scraper's nil dereference MUST NOT crash the entire request
- **Cache backend**: Consider SQLite as lighter alternative to PostgreSQL for single-binary mode; support both via interface
- **Per-scraper code formatting**: Beyoncé simple normalization — sites like FANZA/JavBus need fundamentally different ID systems
- **Progress feedback**: Nice-to-have for v2 (non-blocking progress updates during long searches)

---

## Work Objectives

### Core Objective
Build a production-grade Go REST API that, given a JAV code, concurrently queries the JavDB API for metadata and scrapes 10 video hosting sites for embed links, then returns an aggregated, cached JSON response with graceful degradation.

### Concrete Deliverables
- `cmd/api/main.go` — Entry point, wires all dependencies, starts chi server with graceful shutdown
- `internal/config/` — YAML + env var configuration loading
- `internal/domain/` — Core types: Movie, VideoResult, VideoSource, SearchRequest, SearchResponse
- `internal/javdb/` — JavDB API client with jdsignature auth, search + movie detail
- `internal/scraper/` — Scraper registry + 10 site-specific scraper implementations
- `internal/aggregator/` — Merges JavDB metadata + video results into unified response
- `internal/cache/` — Two-tier cache: in-memory (go-cache) + PostgreSQL persistence
- `internal/middleware/` — API key auth, request logging, panic recovery, CORS
- `internal/handler/` — HTTP handlers: search endpoint, health check
- `Dockerfile` + `docker-compose.yml` — Containerized deployment with PostgreSQL
- `configs/config.yaml` — Default configuration template

### Definition of Done
- [ ] `GET /api/v1/search?code=ABC-123` returns JSON with JavDB metadata + video links from ≥ 8/10 sites
- [ ] `go test ./...` passes all test suites (unit + integration)
- [ ] `go build -o javapi ./cmd/api` produces a working binary
- [ ] `docker compose up` starts API + PostgreSQL, health check passes
- [ ] Graceful shutdown: SIGTERM → drain connections → close scrapers → close DB
- [ ] API key auth: requests without valid X-API-Key return 401
- [ ] Cached responses returned in <50ms, uncached in <15s

### Must Have
- Single search endpoint: `GET /api/v1/search?code={JAV_CODE}` (or `POST /api/v1/search`)
- JavDB metadata: title, number, actors, release_date, cover_url, thumb_url, duration, score, tags, series, director, maker, publisher, preview_video_url, preview_images
- Video results per site: siteName, status, pageUrl, videoSources[] (url, type m3u8/mp4, quality), subtitle flag, leak flag, error message
- **Multi-version results**: Each site can return multiple `VideoResult` entries for the same code (original, cnsub/chinese-subtitle, mosaic-reduced/leak)
- API key authentication via X-API-Key header
- Concurrent scraping (all 8 sites in parallel, capped at 6 for Render free tier)
- In-memory caching with configurable TTL
- **Optional PostgreSQL persistent cache** (disabled by default for database-less mode)
- **Per-site proxy configuration**: individual proxy URL + enable/disable toggle per site (BrightData compatible)
- **Cloudflare bypass**: Pre-test each site with cloudscraper_go before adding; skip sites where CF cannot be bypassed
- Graceful shutdown with timeout
- Docker + standalone binary deployment
- **Render.com free tier compatible** (<512MB RAM, GOMEMLIMIT=400MiB)

### Must NOT Have (Guardrails)
- **NO Cloudflare Workers integration** — all video links from direct scraping (per user decision)
- **NO frontend UI** — pure API, no HTML templates or static files
- **NO user management system** — single API key, no registration/login
- **NO file organization/NFO generation** — this is not a media manager
- **NO torrent/magnet handling** — only embed video links (magnet search is out of scope)
- **NO JavDB login/registration** — only public signed routes (no authenticated JavDB endpoints requiring user token)
- **NO sites that fail CF bypass** — each site must pass cloudscraper_go pre-test before inclusion
- **EXCLUDED sites**: javtrailers, FANZA 動画, JavBus, JAVLib — removed from scope entirely
- **NO AI slop**: Don't add Swagger/OpenAPI unless user asks; don't over-abstract; don't add observability (Prometheus/Grafana) unless asked
- **NO premature optimization**: Start with in-memory + optional PostgreSQL. Don't add Redis, Kafka, or distributed tracing.
- **Scope boundary**: Only the 8 recommended sites. Remaining javjump sites are explicitly OUT of scope.

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: NO (new project)
- **Automated tests**: YES — TDD (RED-GREEN-REFACTOR)
- **Framework**: go test + testify (assert + mock) + httptest
- **Test setup included in plan**: Task 1 (scaffolding) includes `go mod init`, test directory structure, and a smoke test to verify the test framework works
- **CF bypass testing**: Each scraper task includes a CF bypass pre-test step — if cloudscraper_go fails, the site is skipped (not added)

### QA Policy
Every task includes agent-executed QA scenarios. Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Go tests**: `go test ./... -v -count=1` — unit + integration tests
- **API endpoints**: curl to verify status codes, response JSON shape, headers
- **Docker**: `docker compose up`, curl health check, `docker compose down`
- **Binary**: `./javapi`, curl health check, SIGTERM graceful shutdown
- **CF bypass test**: cloudscraper_go GET request to verify CF is bypassed; 200 OK with expected content = pass

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - foundation + scaffolding, MAX PARALLEL):
├── Task 1: Project scaffolding + test infrastructure [quick]
├── Task 2: Configuration loading (YAML + env vars, proxy + db-less) [quick]
├── Task 3: Core domain types + interfaces (multi-version support) [quick]
├── Task 4: JavDB API client (jdsignature + search + detail) [unspecified-high]
├── Task 5: HTTP server skeleton (chi router, graceful shutdown) [quick]
├── Task 6: API key auth middleware [quick]
├── Task 7: Cache layer interfaces + in-memory implementation [quick]

Wave 2 (After Wave 1 - scrapers + CF bypass + cache, MAX PARALLEL):
├── Task 8: Scraper registry + base types + cloudscraper_go CF bypass [quick]
├── Task 9: PostgreSQL cache implementation (optional, database-less mode) [unspecified-high]
├── Task 10: MISSAV scraper (with CF pre-test) [unspecified-high]
├── Task 11: Jable scraper (with CF pre-test, custom code format) [unspecified-high]
├── Task 13: JAVMENU scraper (with CF pre-test) [unspecified-high]
├── Task 14: HAYAV scraper (with CF pre-test) [unspecified-high]
├── Task 15: Supjav scraper (with CF pre-test, custom headers) [unspecified-high]
├── Task 16: javgg scraper (with CF pre-test, custom headers) [unspecified-high]
├── Task 18: AV01 scraper (POST JSON API, CF pre-test) [unspecified-high]
├── Task 19: 7mmtv scraper (POST form, CF pre-test) [unspecified-high]

Wave 3 (After Wave 2 - integration + handlers):
├── Task 20: Aggregator (merge JavDB + multi-version video results, cache) [deep]
├── Task 21: Search handler + JSON response formatting [quick]
├── Task 22: Dockerfile + docker-compose.yml (optional PG, Render-ready) [quick]
├── Task 23: Integration tests (end-to-end) [deep]

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── Task F1: Plan Compliance Audit (oracle)
├── Task F2: Code Quality Review (unspecified-high)
├── Task F3: Real Manual QA (unspecified-high)
└── Task F4: Scope Fidelity Check (deep)
-> Present results -> Get explicit user okay
```

**Critical Path**: Task 1 → Task 3 → Task 4 → Task 8 → Task 20 → Task 21 → Task 23 → F1-F4 → user okay
**Parallel Speedup**: ~60% faster than sequential
**Max Concurrent**: 8 (Wave 2) — capped to 6 for Render free tier runtime

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | - | 2-7 | 1 |
| 2 | 1 | 4,5,7,9 | 1 |
| 3 | 1 | 4,5,7,8,20 | 1 |
| 4 | 2,3 | 20 | 1 |
| 5 | 1,3 | 6,21,23 | 1 |
| 6 | 5 | 21 | 1 |
| 7 | 2,3 | 9,20 | 1 |
| 8 | 3 | 10-11,13-16,18-19 | 2 |
| 9 | 2,3,7 | 20 | 2 |
| 10-11,13-16,18-19 | 8 | 20 | 2 |
| 20 | 4,7,9,10-11,13-16,18-19 | 21 | 3 |
| 21 | 5,6,20 | 23 | 3 |
| 22 | 1 | 23 | 3 |
| 23 | 5,21,22 | F1-F4 | 3 |

> **Removed tasks**: Task 12 (javtrailers) and Task 17 (JavBus) — excluded per user request.

### Agent Dispatch Summary

- **Wave 1**: **7** — T1 → `quick`, T2 → `quick`, T3 → `quick`, T4 → `unspecified-high`, T5 → `quick`, T6 → `quick`, T7 → `quick`
- **Wave 2**: **10** — T8 → `quick`, T9 → `unspecified-high`, T10-T11,13-16,18-19 → `unspecified-high`
- **Wave 3**: **4** — T20 → `deep`, T21 → `quick`, T22 → `quick`, T23 → `deep`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. **Project Scaffolding + Test Infrastructure**

  **What to do**:
  - Run `go mod init github.com/henry/javapi` (or appropriate module path)
  - Create directory structure: `cmd/api/`, `internal/{config,domain,javdb,scraper,aggregator,cache,middleware,handler}`, `configs/`, `.sisyphus/evidence/`
  - Create `cmd/api/main.go` with minimal `func main()` that prints "javapi starting..."
  - Create `internal/domain/movie_test.go` with a smoke test:
    ```go
    func TestSmoke(t *testing.T) { assert.True(t, true) }
    ```
  - Verify: `go test ./... -v` → all pass
  - Add `.gitignore` (Go binary, `.env`, evidence files)
  - Create `Makefile` with targets: `build`, `test`, `run`, `docker-build`, `docker-up`, `docker-down`, `clean`

  **Must NOT do**:
  - No Dockerfile yet (Task 22)
  - No actual domain types yet (Task 3)
  - No chi router setup yet (Task 5)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Pure scaffolding — directory creation, go mod init, trivial main.go
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Domain overlap — Go project initialization, module setup, idiomatic Go conventions

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4, 5, 6, 7)
  - **Blocks**: Tasks 2, 3, 4, 5, 6, 7, 8, 9, 10-19, 20, 21, 22, 23
  - **Blocked By**: None (can start immediately)

  **References**:
  - **External**: Official docs `https://go.dev/doc/tutorial/create-module` — Go module initialization syntax
  - **External**: `https://github.com/golang-standards/project-layout` — Standard Go project layout reference (not a strict standard, just reference)
  - **External**: `https://pkg.go.dev/github.com/stretchr/testify@v1.10.0` — testify assert API
  - **Pattern**: javinizer-go project structure — `cmd/`, `internal/`, `configs/` layout pattern

  **Acceptance Criteria**:
  - [ ] `go mod init` succeeds, module path set
  - [ ] All directories created: `cmd/api/`, `internal/{config,domain,javdb,scraper,aggregator,cache,middleware,handler}`, `configs/`, `.sisyphus/evidence/`
  - [ ] `cmd/api/main.go` exists and compiles: `go build ./cmd/api` → no errors
  - [ ] `go test ./... -v` → 1 test passes (smoke test)
  - [ ] `.gitignore` exists with Go-specific patterns (`/javapi`, `.env`, `/data/`)
  - [ ] `Makefile` exists with `build`, `test`, `run` targets

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Project compiles and tests pass
    Tool: Bash
    Preconditions: Working directory is project root
    Steps:
      1. Run: go build ./cmd/api
      2. Assert: Exit code 0, binary created at ./cmd/api/api (or project root)
      3. Run: go test ./... -v
      4. Assert: Exit code 0, output contains "PASS" and "ok"
    Expected Result: Binary compiles cleanly, smoke test passes
    Failure Indicators: Build errors, test failures, missing go.sum
    Evidence: .sisyphus/evidence/task-1-smoke.txt

  Scenario: Makefile targets work
    Tool: Bash
    Preconditions: Project scaffolding complete
    Steps:
      1. Run: make build
      2. Assert: Exit code 0, binary exists
      3. Run: make test
      4. Assert: Exit code 0, test output shows PASS
      5. Run: make clean
      6. Assert: Binary removed
    Expected Result: make build and make test succeed
    Failure Indicators: make: *** No rule to make target
    Evidence: .sisyphus/evidence/task-1-makefile.txt
  ```

  **Commit**: YES (alone)
  - Message: `chore: initialize Go project scaffolding with test infra`
  - Files: `go.mod`, `go.sum`, `cmd/api/main.go`, `internal/domain/movie_test.go`, `.gitignore`, `Makefile`

- [x] 2. **Configuration Loading (YAML + Env Vars) — Proxy + DB-less + Render**

  **What to do**:
  - Create `internal/config/config.go` — `Config` struct with all settings
  - Create `internal/config/config_test.go` — test loading from YAML + env var override
  - Create `configs/config.yaml` — default configuration template
  - Config fields:
    ```go
    type Config struct {
        Server    ServerConfig    // host, port, read_timeout, write_timeout, shutdown_timeout
        JavDB     JavDBConfig     // base_url, middle, suffix (jdsignature material)
        Cache     CacheConfig     // memory_ttl_seconds, postgres_url, postgres_enabled (default: false for db-less)
        Scrapers  ScrapersConfig  // timeout_seconds, rate_limit_delay, user_agent, max_concurrent (default: 6 for Render)
        Auth      AuthConfig      // api_keys[]
        Render    RenderConfig    // gomemlimit (default: "400MiB"), cold_start_tolerant (default: true)
    }
    ```
  - **Per-site proxy config** (in scrapers section):
    ```go
    type ScraperSiteConfig struct {
        Name        string   // e.g. "MISSAV"
        Enabled     bool     // default: true
        ProxyURL    string   // e.g. "http://user:pass@brd.superproxy.io:22225" (BrightData)
        ProxyEnabled bool    // default: false
        TimeoutSec  int      // per-site override, default from ScrapersConfig
    }
    ```
  - **Database-less mode**: `cache.postgres_enabled` defaults to `false`
  - Support: `configs/config.yaml` as base, env var overrides (e.g., `JAVDB_MIDDLE=xxx`)
  - Use `os.Getenv` + YAML unmarshaling (`gopkg.in/yaml.v3`)

  **Must NOT do**:
  - No viper/koanf — keep it simple with stdlib `gopkg.in/yaml.v3` + `os.Getenv`
  - No hot reloading — config loaded once at startup
  - Don't put real secrets in `configs/config.yaml` — use env var placeholders

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple struct definition + YAML unmarshaling + env var fallback
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Domain overlap — Go struct tags, YAML parsing

  **Parallelization**:
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4, 5, 6, 7)
  - **Blocks**: Tasks 4, 5, 7, 9
  - **Blocked By**: Task 1 (scaffolding)

  **References**:
  - **External**: `https://pkg.go.dev/gopkg.in/yaml.v3` — YAML v3 unmarshaling API
  - **Pattern**: javinizer-go `internal/config/config.go` — Config struct layout
  - **Internal**: `JavDB API/auth.md` — jdsignature constants
  - **External**: `https://docs.render.com/free` — Render free tier limits (512MB RAM)
  - **External**: BrightData proxy format: `http://{username}:{password}@brd.superproxy.io:22225`

  **Acceptance Criteria**:
  - [ ] Config struct defines all sections including proxy per-site + db-less + Render
  - [ ] `configs/config.yaml` exists with all required sections
  - [ ] `postgres_enabled` defaults to `false` (database-less mode)
  - [ ] Each site config has `proxy_url` + `proxy_enabled` fields
  - [ ] `max_concurrent` defaults to `6` (Render-safe)
  - [ ] `gomemlimit` defaults to `"400MiB"`
  - [ ] `go test ./internal/config/... -v` → tests verify YAML loading + env override

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Config loads with database-less defaults
    Tool: Bash (go test)
    Steps: go test ./internal/config/... -v -run TestDatabaseLessDefault
    Assert: Exit 0, postgres_enabled==false
    Evidence: .sisyphus/evidence/task-2-dbless.txt

  Scenario: Per-site proxy config loads
    Tool: Bash (go test)
    Steps: go test ./internal/config/... -v -run TestProxyConfig
    Assert: Exit 0, MISSAV proxy_url loaded
    Evidence: .sisyphus/evidence/task-2-proxy.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(config): add proxy per-site, db-less mode, Render.com defaults`
  - Files: `internal/config/config.go`, `internal/config/config_test.go`, `configs/config.yaml`

- [x] 3. **Core Domain Types + Interfaces — Multi-Version + Proxy Support**

  **What to do**:
  - Create `internal/domain/movie.go` — `Movie` struct (JavDB metadata, same as before)
  - Create `internal/domain/video.go` — `VideoResult`, `VideoSource`, `VideoVersion`, `ScrapeStatus` structs
  - **Multi-version support**: Each site can return multiple `VideoResult` entries for the same code:
    ```go
    type VideoVersion string
    const (
        VersionOriginal     VideoVersion = "original"      // 原版
        VersionCNSub        VideoVersion = "cnsub"         // 中文字幕
        VersionMosaicReduce VideoVersion = "mosaic_reduce" // 去马赛克
    )
    type VideoResult struct {
        SiteName     string         `json:"siteName"`
        Status       ScrapeStatus   `json:"status"`
        PageURL      string         `json:"pageUrl,omitempty"`
        Version      VideoVersion   `json:"version"`       // NEW: which version this is
        Label        string         `json:"label,omitempty"` // "Reducing", "Chinese Sub", etc.
        VideoSources []VideoSource  `json:"videoSources,omitempty"`
        Subtitle     bool           `json:"subtitle"`
        Leak         bool           `json:"leak"`
        Error        string         `json:"error,omitempty"`
    }
    ```
  - Create `internal/domain/search.go` — `SearchRequest`, `SearchResponse` structs
  - Create `internal/domain/scraper.go` — `Scraper` interface (updated):
    ```go
    type Scraper interface {
        Name() string
        Search(ctx context.Context, code string) ([]VideoResult, error)  // returns multiple versions
        FormatCode(code string) string
        IsEnabled() bool
        RequiresCFBypass() bool     // NEW: does this site need cloudscraper_go?
        GetProxyConfig() ProxyConfig // NEW: per-site proxy settings
    }
    type ProxyConfig struct {
        URL     string
        Enabled bool
    }
    ```
  - Create `internal/domain/cache.go` — `Cache` interface (unchanged)

  **Must NOT do**: No implementation — just types and interfaces

  **Recommended Agent Profile**: `quick` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 1 | **Blocks**: Tasks 4, 5, 7, 8, 20 | **Blocked By**: Task 1

  **References**: `JavDB API/objects.md`, `JavDB API/public/movies.md`, javjump multi-version pattern (reducing_mosaic variant), javinizer-go models

  **Acceptance Criteria**:
  - [ ] VideoResult includes `Version` field (original/cnsub/mosaic_reduce)
  - [ ] Scraper interface includes `RequiresCFBypass()` + `GetProxyConfig()`
  - [ ] All structs have JSON tags
  - [ ] `go build ./...` compiles, `go vet ./...` clean

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Multi-version VideoResult struct marshals correctly
    Tool: Bash (go test)
    Steps: go test ./internal/domain/... -v -run TestVideoResultMarshal
    Assert: Exit 0, json output contains version field
    Evidence: .sisyphus/evidence/task-3-version.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(domain): define core types with multi-version + proxy support`
  - Files: `internal/domain/{movie,video,search,scraper,cache}.go`

- [x] 4. **JavDB API Client (jdsignature + Search + Movie Detail)**

  **What to do**:
  - Create `internal/javdb/client.go` — HTTP client with jdsignature generation
  - Create `internal/javdb/client_test.go` — tests with httptest mock server
  - Implement `jdsignature` generation:
    ```go
    func generateSignature() string {
        ts := strconv.FormatInt(time.Now().Unix(), 10)
        hash := md5sum(ts + suffix)
        return ts + "." + middle + "." + hash
    }
    ```
  - Implement `Search(ctx, code string) (*domain.Movie, error)`:
    1. Call `GET https://jdforrepam.com/api/v2/search?q={code}&page=1`
    2. Set `jdsignature` header + `accept: application/json`
    3. Parse response: extract first movie from `data.movies[]`
    4. If found, return `Movie` (search-level fields)
  - Implement `GetMovie(ctx, movieID string) (*domain.Movie, error)`:
    1. Call `GET https://jdforrepam.com/api/v4/movies/{movieID}`
    2. Parse full detail: actors, tags, series, director, maker, publisher, preview_video_url, preview_images, play_sources
  - Handle API errors: `ParameterInvalid`, `InvalidSignature`, `JWTVerificationError`, `HTTP 404`
  - Use go-resty: retry (3 times, 2s backoff), timeout (10s), circuit breaker

  **Must NOT do**:
  - No HTML scraping of javdb.com — use the API only
  - No authenticated endpoints (no login, no token management)
  - No caching in this layer (caching is Task 7/9)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Moderate complexity — HTTP client with custom auth header, response parsing, error handling, retry logic
  - **Skills**: [`golang-pro`]
    - `golang-pro`: Domain overlap — Go HTTP client patterns, error handling, struct marshaling

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 5, 6, 7)
  - **Blocks**: Task 20 (aggregator)
  - **Blocked By**: Tasks 2 (config), 3 (domain types)

  **References**:
  - **Internal**: `JavDB API/auth.md` — jdsignature format, base host `jdforrepam.com`, minimal request headers
  - **Internal**: `JavDB API/public/search-and-filters.md` — `/api/v2/search` endpoint: required `q` param, response `movies[]` with fields
  - **Internal**: `JavDB API/public/movies.md` — `/api/v4/movies/%s` endpoint: full detail response fields
  - **Internal**: `JavDB API/objects.md` — `movie` object fields (both search-level and detail-level)
  - **Internal**: `JavDB API/errors.md` — Error handling: ParameterInvalid, InvalidSignature, JWTVerificationError, HTTP 404
  - **External**: `https://github.com/go-resty/resty` — go-resty v2 client setup, retry, timeout patterns

  **Acceptance Criteria**:
  - [ ] `internal/javdb/client.go` compiles
  - [ ] `jdsignature` header generated correctly: `{timestamp}.lpw6vgqzsp.{md5(timestamp+suffix)}`
  - [ ] `Search()` method: calls correct URL, parses movie from response
  - [ ] `GetMovie()` method: calls correct URL, parses full detail
  - [ ] Go tests: `go test ./internal/javdb/... -v` → pass (using httptest mock server)
  - [ ] Test covers: successful search, empty search result, invalid signature error, network timeout

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: JavDB search returns movie metadata
    Tool: Bash (go test)
    Preconditions: httptest mock server returns valid JavDB JSON response
    Steps:
      1. Run: go test ./internal/javdb/... -v -run TestSearch
      2. Assert: Exit code 0, Movie.Number not empty, Movie.Title not empty
    Expected Result: Movie struct populated from API response
    Failure Indicators: empty Movie fields, unmarshal errors, nil pointer
    Evidence: .sisyphus/evidence/task-4-search.txt

  Scenario: JavDB invalid signature error handled
    Tool: Bash (go test)
    Preconditions: httptest mock server returns InvalidSignature error
    Steps:
      1. Run: go test ./internal/javdb/... -v -run TestInvalidSignature
      2. Assert: Exit code 0, error returned (not nil), error message contains "signature"
    Expected Result: Error returned, no panic
    Failure Indicators: nil error when should fail, panic
    Evidence: .sisyphus/evidence/task-4-errors.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(javdb): implement JavDB API client with jdsignature auth`
  - Files: `internal/javdb/client.go`, `internal/javdb/client_test.go`

- [x] 5. **HTTP Server Skeleton (chi Router + Graceful Shutdown)**

  **What to do**:
  - Create `cmd/api/main.go` (replace trivial one from Task 1): full server setup with chi
  - Create `internal/middleware/logging.go`: request logging (method, path, status, duration)
  - Create `internal/middleware/recovery.go`: panic recovery → 500, log stack
  - Create `internal/middleware/cors.go`: CORS headers (allow all origins, v1)
  - Create `internal/handler/health.go`: `GET /api/health` → `{"status":"ok"}`
  - Wire: chi router → recovery → logging → cors → health handler
  - Graceful shutdown: SIGTERM → drain connections (25s timeout) → exit

  **Must NOT do**:
  - No search handler yet (Task 21)
  - No API key auth middleware yet (Task 6)
  - No database connections yet

  **Recommended Agent Profile**:
  - **Category**: `quick` — Standard chi setup, minimal logic
  - **Skills**: [`golang-pro`] — Go HTTP server, chi middleware, os/signal

  **Parallelization**:
  - **Parallel Group**: Wave 1 (with Tasks 1-4, 6, 7)
  - **Blocks**: Tasks 6, 21, 23
  - **Blocked By**: Tasks 1, 3

  **References**:
  - **External**: `https://go-chi.io/#/pages/using_chi` — chi router patterns
  - **External**: `https://pkg.go.dev/github.com/go-chi/chi/v5/middleware` — built-in middleware
  - **Pattern**: Go std graceful shutdown: `signal.NotifyContext`, `srv.Shutdown(ctx)`, timeout context

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/api` compiles
  - [ ] `GET /api/health` → 200 `{"status":"ok"}`
  - [ ] Logging middleware logs each request (method, path, status, duration)
  - [ ] SIGTERM triggers graceful shutdown
  - [ ] Panic in handler → 500, no crash
  - [ ] CORS headers present

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Health endpoint responds
    Tool: Bash (curl)
    Steps:
      1. go run ./cmd/api & sleep 2
      2. curl -s http://localhost:8080/api/health
      3. Assert: 200, body={"status":"ok"}
      4. kill %1
    Expected: 200 OK with status json
    Evidence: .sisyphus/evidence/task-5-health.txt

  Scenario: Graceful shutdown on SIGTERM
    Tool: Bash
    Steps:
      1. go run ./cmd/api & sleep 2
      2. kill -TERM %1 && sleep 3
      3. curl -s http://localhost:8080/api/health
      4. Assert: Connection refused
    Expected: Server stops cleanly
    Evidence: .sisyphus/evidence/task-5-shutdown.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(server): add chi router with middleware chain and graceful shutdown`
  - Files: `cmd/api/main.go`, `internal/middleware/{logging,recovery,cors}.go`, `internal/handler/health.go`

- [x] 6. **API Key Auth Middleware**

  **What to do**:
  - Create `internal/middleware/auth.go`: reads `X-API-Key` header, validates against `config.Auth.APIKeys[]`
  - Return 401 `{"error":"unauthorized","message":"invalid or missing API key"}` on failure
  - Whitelist `/api/health` (no auth required)
  - Create `internal/middleware/auth_test.go`: test valid key, invalid key, missing key, health bypass

  **Must NOT do**: No JWT/OAuth, no user management, no rate limiting per key

  **Recommended Agent Profile**:
  - **Category**: `quick` — Simple header check + string comparison
  - **Skills**: [`golang-pro`]

  **Parallelization**: Wave 1 | **Blocks**: Task 21 | **Blocked By**: Task 5

  **References**: `https://go-chi.io/#/pages/using_chi?id=middleware` — chi middleware pattern

  **Acceptance Criteria**:
  - [ ] Missing X-API-Key → 401
  - [ ] Wrong X-API-Key → 401
  - [ ] Valid X-API-Key → passes to handler
  - [ ] `/api/health` accessible without key
  - [ ] `go test ./internal/middleware/... -v` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Missing API key → 401
    Tool: Bash (curl)
    Steps: curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/search?code=ABC-123
    Assert: 401
    Evidence: .sisyphus/evidence/task-6-no-key.txt

  Scenario: Valid API key passes
    Tool: Bash (curl)
    Steps: curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: test-key" http://localhost:8080/api/health
    Assert: 200
    Evidence: .sisyphus/evidence/task-6-valid-key.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(auth): add API key authentication middleware`
  - Files: `internal/middleware/auth.go`, `internal/middleware/auth_test.go`

- [x] 7. **Cache Layer — In-Memory Implementation**

  **What to do**:
  - Create `internal/cache/memory.go`: implements `Cache` interface from Task 3 using `patrickmn/go-cache`
  - `Get(ctx, key)` → `*domain.SearchResponse` + bool
  - `Set(ctx, key, value, ttl)` → stores with configurable TTL (default 300s)
  - `Delete(ctx, key)` → removes entry
  - Cache key: `normalizeCode(code)` — lowercase, no separators
  - Create `internal/cache/memory_test.go`: get/set/delete/expiry tests

  **Must NOT do**: No PostgreSQL implementation (Task 9), no cache warming

  **Recommended Agent Profile**:
  - **Category**: `quick` — Simple go-cache wrapper
  - **Skills**: [`golang-pro`]

  **Parallelization**: Wave 1 | **Blocks**: Tasks 9, 20 | **Blocked By**: Tasks 2, 3

  **References**: `https://github.com/patrickmn/go-cache` — go-cache API; `internal/domain/cache.go` — interface

  **Acceptance Criteria**:
  - [ ] `Set()` + `Get()` round-trip preserves data
  - [ ] `Get()` on missing key → false
  - [ ] `Delete()` removes entry
  - [ ] TTL expiry works (entry gone after TTL)
  - [ ] `go test ./internal/cache/... -v` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Cache get after set preserves data
    Tool: Bash (go test)
    Steps: go test ./internal/cache/... -v -run TestCacheGetSet
    Assert: Exit 0, round-trip preserves data
    Evidence: .sisyphus/evidence/task-7-cache-getset.txt

  Scenario: TTL expiry works
    Tool: Bash (go test)
    Steps: go test ./internal/cache/... -v -run TestCacheExpiry
    Assert: Exit 0, value gone after TTL
    Evidence: .sisyphus/evidence/task-7-cache-expiry.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(cache): implement in-memory cache with TTL support`
  - Files: `internal/cache/memory.go`, `internal/cache/memory_test.go`

- [x] 8. **Scraper Registry + Code Utilities + Cloudscraper CF Bypass**

  **What to do**:
  - Create `internal/scraper/registry.go` — self-registering plugin pattern (map + RWMutex)
  - Create `internal/scraper/code.go` — code normalization utilities:
    - `NormalizeCode(code string) string`: lowercase + strip `[-_ ]` → `"abc123"`
    - `ExtractCode(raw string) string`: regex `[a-zA-Z]{2,10}[-_ ]?\d{2,6}`
  - Create `internal/scraper/safe.go` — `safeSearch()` panic-recovery wrapper
  - **Create `internal/scraper/cf.go` — Cloudflare bypass via cloudscraper_go**:
    ```go
    // CFBypassTest tests whether cloudscraper_go can successfully bypass Cloudflare on a given URL.
    // Returns true if the page loads with expected content (200 OK, no CF challenge page).
    // Uses a lightweight cloudscraper usage pattern.
    func CFBypassTest(ctx context.Context, url string, proxyURL string) (bool, error) {
        // 1. Create a custom http.Client that uses cloudscraper_go's transport
        // 2. Set proxy if proxyURL is non-empty
        // 3. GET the URL
        // 4. Check response: status 200 AND body does NOT contain CF challenge markers
        //    ("Just a moment", "Checking your browser", "cf-browser-verification", "challenge-platform")
        // 5. Return true if pass, false if CF blocked
    }
    ```
  - **Integration**: Use `github.com/RomainMichau/cloudscraper_go` as Go dependency
  - **Proxy support**: If proxyURL set, route through proxy for CF test
  - **Result caching**: Cache CF test results per site (don't re-test on every request)
  - Create registry test file

  **Must NOT do**: No actual scraper implementations (Tasks 10+). Don't bypass CF on sites that don't need it (only test sites with `RequiresCFBypass() == true`).

  **Recommended Agent Profile**:
  - **Category**: `quick` — Registry + regex + cloudscraper_go wrapper
  - **Skills**: [`golang-pro`]

  **Parallelization**: Wave 2 | **Blocks**: Tasks 10-11,13-16,18-19 | **Blocked By**: Task 3

  **References**:
  - `https://github.com/RomainMichau/cloudscraper_go` — cloudscraper_go API: install, setup, usage
  - `internal/domain/scraper.go` — Scraper interface (Task 3)
  - javjump `helpers.ts` — normalizeCode/extractVideoCode
  - javjump `fetcher.ts` — Cloudflare detection markers
  - javinizer-go safeSearch pattern

  **Acceptance Criteria**:
  - [ ] `CFBypassTest()` can test a URL with cloudscraper_go
  - [ ] Returns true when page loads successfully (no CF challenge)
  - [ ] Returns false when CF blocks the request
  - [ ] Proxy support: test routes through proxy when proxyURL set
  - [ ] Results cached (per site, per session)
  - [ ] `NormalizeCode("ABC-123")` → `"abc123"` (4 variants)
  - [ ] `safeSearch()` recovers from panic
  - [ ] `go test ./internal/scraper/... -v` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: CF bypass test on known CF-protected site
    Tool: Bash (go test)
    Steps: go test ./internal/scraper/... -v -run TestCFBypass
    Assert: Exit 0, function returns bool without error (true=pass, false=blocked)
    Evidence: .sisyphus/evidence/task-8-cf-bypass.txt

  Scenario: Code normalization covers all formats
    Tool: Bash (go test)
    Steps: go test ./internal/scraper/... -v -run TestNormalizeCode
    Assert: Exit 0, ABC-123→abc123, ABC_123→abc123, etc.
    Evidence: .sisyphus/evidence/task-8-normalize.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(scraper): add registry, code normalization, safeSearch, and cloudscraper CF bypass`
  - Files: `internal/scraper/{registry,code,safe,cf}.go`, `go.mod` (add cloudscraper_go dep)

- [x] 9. **PostgreSQL Cache Implementation (Optional — Database-Less Mode Supported)**

  **What to do**:
  - Create `internal/cache/postgres.go`: implements `Cache` interface using `database/sql` + `lib/pq`
  - Create `internal/cache/postgres_test.go`
  - **Database-less mode**: When `config.Cache.PostgresEnabled == false` (default!), PostgresCache is never instantiated — only MemoryCache is used
  - Table schema (same as before, only created when enabled)
  - `Set()`: UPSERT response JSON; `Get()`: SELECT where not expired; `Delete()`: remove by code
  - Connection from `config.Cache.PostgresURL`; if empty, auto-disable
  - In `cmd/api/main.go`: conditionally create PostgresCache only if `PostgresEnabled`

  **Must NOT do**: No ORM, no migration framework, no auto-enable on Render

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Tasks 2, 3, 7

  **Acceptance Criteria**:
  - [ ] PostgresCache only created when `postgres_enabled=true`
  - [ ] When `postgres_enabled=false`, server starts normally without PG (database-less mode)
  - [ ] When enabled: Set/Get/Delete work, TTL enforced, concurrent-safe
  - [ ] `go test ./internal/cache/... -v` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Server starts in database-less mode
    Tool: Bash
    Steps: POSTGRES_ENABLED=false go run ./cmd/api & sleep 2; curl -s http://localhost:8080/api/health; kill %1
    Assert: HTTP 200, no PG connection error
    Evidence: .sisyphus/evidence/task-9-dbless.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(cache): add optional PostgreSQL cache with database-less mode support`
  - Files: `internal/cache/postgres.go`, `internal/cache/postgres_test.go`, `cmd/api/main.go` (conditional init)

- [x] 10. **MISSAV Scraper (GET — Direct URL + CF Test + Proxy + Multi-Version)**

  **What to do**:
  - Create `internal/scraper/missav/scraper.go` + test
  - **CF Pre-test**: Before adding, run `CFBypassTest("https://missav.ws/")`. If blocked, mark site as `Enabled: false` and skip.
  - URL: `https://missav.ws/{code}/` (raw code)
  - **Proxy**: Use `config.Scrapers.Sites["MISSAV"].ProxyURL` if `ProxyEnabled`
  - **Multi-version detection**: Check page for:
    - Chinese subtitle variant → create VideoResult with `Version: "cnsub"`, `Label: "Chinese Sub"`
    - Leak/uncensored variant → create VideoResult with `Version: "mosaic_reduce"`, `Label: "Reducing"`
    - Default/original → create VideoResult with `Version: "original"`
  - Parse with goquery: video player, title verification, subtitle/leak tag detection (`.space-y-2 a.text-nord13`, `.order-first div.rounded-md`)
  - Return `[]VideoResult` (may be 1-3 entries)
  - Auto-register via `init()`

  **Must NOT do**: If CF bypass fails, do NOT add the scraper. Log warning: "MISSAV: Cloudflare bypass failed, site disabled"

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8

  **References**: javjump MISSAV config, `internal/scraper/cf.go` — CFBypassTest, BrightData proxy format

  **Acceptance Criteria**:
  - [ ] CF pre-test: site tested before registration
  - [ ] Proxy support: requests routed through proxy when enabled
  - [ ] Multi-version: up to 3 VideoResult entries returned
  - [ ] `go test ./internal/scraper/missav/... -v` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: MISSAV multi-version detection
    Tool: Bash (go test)
    Steps: go test ./internal/scraper/missav/... -v -run TestMultiVersion
    Assert: Exit 0, returns multiple VideoResult with different versions
    Evidence: .sisyphus/evidence/task-10-multiversion.txt
  ```

  **Commit**: YES (alone) — `feat(scraper): add MISSAV scraper with CF test, proxy, multi-version support`
  - Files: `internal/scraper/missav/scraper.go`, `internal/scraper/missav/scraper_test.go`

- [x] 11. **Jable Scraper (GET — Custom Code Format + CF Test + Proxy)**

  **What to do**:
  - Create `internal/scraper/jable/scraper.go` + test
  - **CF Pre-test**: `CFBypassTest("https://jable.tv/")`; skip if blocked
  - `FormatCode()`: replace `[\s_]+` with `-`, uppercase → `abc_123` → `ABC-123`
  - URL: `https://jable.tv/search/{code}/`, specific headers (Referer, Origin, Accept-Language: zh-CN)
  - **Proxy**: Use config proxy if enabled
  - **Multi-version**: detect cnsub variants from search results
  - Parse search results → follow match → extract player sources → return `[]VideoResult`
  - Special 404 detection: check `finalUrl`, response length, 404 indicators

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8
  **Acceptance Criteria**: CF test passed, code formatting works, multi-version detection, proxy support
  **Commit**: YES (alone) — `feat(scraper): add Jable scraper with CF test, proxy, multi-version`

- [ ] ~~12. javtrailers Scraper~~ **REMOVED** — Excluded per user request. (Task number reserved, skip to 13.)

- [x] 13. **JAVMENU Scraper (GET — Direct URL + CF Test + Proxy)**

  **What to do**:
  - Create `internal/scraper/javmenu/scraper.go` + test
  - **CF Pre-test**: `CFBypassTest("https://javmenu.com/")`; skip if blocked
  - URL: `https://javmenu.com/{code}`, player detection: `#primary-player video[src]`, `#seo-main-video[src]`
  - **Proxy**: Use config proxy if enabled
  - **Multi-version**: check page for multiple video source variants

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8
  **Commit**: YES (alone) — `feat(scraper): add JAVMENU scraper with CF test, proxy`

- [x] 14. **HAYAV Scraper (GET — Direct URL + CF Test + Proxy)**

  **What to do**:
  - Create `internal/scraper/hayav/scraper.go` + test
  - **CF Pre-test**: `CFBypassTest("https://hayav.com/")`; skip if blocked
  - URL: `https://hayav.com/video/{code}/`, multi-selector player detection
  - **Proxy**: Use config proxy if enabled
  - **Multi-version**: detect subtitle/leak tagged players

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8
  **Commit**: YES (alone) — `feat(scraper): add HAYAV scraper with CF test, proxy`

- [x] 15. **Supjav Scraper** — REMOVED (CF bypass failed, site excluded)

  **What to do**:
  - Create `internal/scraper/supjav/scraper.go` + test
  - **CF Pre-test**: `CFBypassTest("https://supjav.com/")`; skip if blocked
  - Search URL: `https://supjav.com/zh/?s={code}`, custom Referer/Origin headers
  - **Proxy**: Use config proxy if enabled
  - Parse: `.posts.clearfix>.post>a.img[title]`, follow to video page, extract sources

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8
  **Commit**: YES (alone) — `feat(scraper): add Supjav scraper with CF test, proxy`

- [x] 16. **javgg Scraper (Parser — Custom Headers + CF Test + Proxy)**

  **What to do**:
  - Create `internal/scraper/javgg/scraper.go` + test
  - **CF Pre-test**: `CFBypassTest("https://javgg.net/")`; skip if blocked
  - Search URL: `https://javgg.net/?s={code}`, custom headers (Upgrade-Insecure-Requests, Referer)
  - **Proxy**: Use config proxy if enabled
  - Parse: `article .details .title a[href*='/jav/']`, follow, extract sources

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8
  **Commit**: YES (alone) — `feat(scraper): add javgg scraper with CF test, proxy`

- [ ] ~~17. JavBus Scraper~~ **REMOVED** — Excluded per user request. (Task number reserved, skip to 18.)

- [x] 18. **AV01 Scraper (POST — JSON API + CF Test + Proxy)**

  **What to do**:
  - Create `internal/scraper/av01/scraper.go` + test
  - **CF Pre-test**: `CFBypassTest("https://www.av01.media/")`; skip if blocked
  - POST to `https://www.av01.media/api/v1/videos/search?lang=ja` with JSON body
  - **Proxy**: Use config proxy if enabled
  - Parse JSON: match on `dvd_id` or `dmm_id`, build pageUrl

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8
  **Commit**: YES (alone) — `feat(scraper): add AV01 JSON API scraper with CF test, proxy`

- [x] 19. **7mmtv Scraper (POST — Form-URLEncoded + CF Test + Proxy)**

  **What to do**:
  - Create `internal/scraper/7mmtv/scraper.go` + test
  - **CF Pre-test**: `CFBypassTest("https://7mmtv.sx/")`; skip if blocked
  - POST form: `search_keyword={code}&search_type=searchall&op=search`
  - **Proxy**: Use config proxy if enabled
  - Custom parse: filter `a[target='_top'][href$='.html']`, deduplicate, match code via slug + image alt

  **Recommended Agent Profile**: `unspecified-high` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 2 | **Blocks**: Task 20 | **Blocked By**: Task 8
  **Commit**: YES (alone) — `feat(scraper): add 7mmtv form POST scraper with CF test, proxy`

- [x] 20. **Aggregator — Merge JavDB + Multi-Version Video Results + Cache**

  **What to do**:
  - Create `internal/aggregator/aggregator.go` — core orchestration logic
  - Create `internal/aggregator/aggregator_test.go` — test with mock scrapers
  - `Aggregate(ctx, code string) (*domain.SearchResponse, error)`:
    1. Check cache (in-memory first, then PostgreSQL if enabled)
    2. If cache hit: return cached response
    3. If cache miss:
       a. Launch JavDB search + movie detail in parallel with all enabled scrapers
       b. Use `semaphore.Weighted` with `max_concurrent` (default: 6 for Render)
       c. Wrap each scraper in `safeSearch()`
       d. **Multi-version collection**: each scraper returns `[]VideoResult`, flatten all versions into videos array
       e. Sort: original first, then cnsub, then mosaic_reduce within each site
       f. Build SearchResponse: movie metadata + all video results (successes and failures, all versions)
       g. Cache result (in-memory + PostgreSQL if enabled)
       h. Return SearchResponse
  - Graceful degradation: only return error if JavDB AND all scrapers fail
  - **Concurrency cap**: Respect `max_concurrent` from config (6 for Render, higher for dedicated servers)

  **Must NOT do**: No retry logic in aggregator (retry is at scraper level)

  **Recommended Agent Profile**: `deep` — Complex orchestration | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 3 | **Blocks**: Task 21 | **Blocked By**: Tasks 4, 7, 9, 10-11,13-16,18-19

  **References**: `internal/domain/`, `internal/scraper/`, javinizer-go aggregator

  **Acceptance Criteria**:
  - [ ] Cache hit returns immediately (no scraping)
  - [ ] Cache miss fires JavDB + all enabled scrapers in parallel (capped at max_concurrent)
  - [ ] Multi-version: 1 site returning 3 versions → 3 VideoResult entries in response
  - [ ] Partial success: 3/8 scrapers succeed → response with video results + errors
  - [ ] All fail → returns error
  - [ ] safeSearch isolation: one panicking scraper doesn't kill others
  - [ ] `go test ./internal/aggregator/... -v` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Multi-version results aggregated correctly
    Tool: Bash (go test)
    Steps: go test ./internal/aggregator/... -v -run TestMultiVersion
    Assert: Exit 0, 3 VideoResult entries from one site, versions differ
    Evidence: .sisyphus/evidence/task-20-multiversion.txt

  Scenario: Concurrency capped at max_concurrent
    Tool: Bash (go test)
    Steps: go test ./internal/aggregator/... -v -run TestConcurrencyCap
    Assert: Exit 0, only N scrapers run simultaneously (N=max_concurrent)
    Evidence: .sisyphus/evidence/task-20-concurrency.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(aggregator): implement parallel aggregation with multi-version, cache, concurrency cap`
  - Files: `internal/aggregator/aggregator.go`, `internal/aggregator/aggregator_test.go`

- [x] 21. **Search Handler + JSON Response Formatting**

  **What to do**:
  - Create `internal/handler/search.go` — `GET /api/v1/search?code={JAV_CODE}`
  - Create `internal/handler/search_test.go` — test with httptest + mock aggregator
  - Validate `code` query param: not empty, matches `[a-zA-Z0-9-_ ]{4,20}`
  - Call `aggregator.Aggregate(ctx, code)`
  - Format JSON response:
    ```json
    {
      "code": "ABC-123",
      "movie": { ... JavDB metadata ... },
      "videos": [
        { "siteName": "MISSAV", "status": "success", "pageUrl": "...", "videoSources": [...], "subtitle": false, "leak": false },
        { "siteName": "Jable", "status": "error", "error": "not found" }
      ],
      "cache": { "hit": false, "age": 0 },
      "took_ms": 8234
    }
    ```
  - Create `internal/handler/response.go` — standardized JSON response helpers: `writeJSON(w, status, data)`, `writeError(w, status, message)`
  - Wire: `r.Get("/api/v1/search", searchHandler)` with auth middleware

  **Must NOT do**: No pagination for v1 (single code search)

  **Recommended Agent Profile**:
  - **Category**: `quick` — HTTP handler + JSON formatting + input validation
  - **Skills**: [`golang-pro`]

  **Parallelization**: Wave 3 | **Blocks**: Task 23 | **Blocked By**: Tasks 5, 6, 20

  **References**:
  - `https://go-chi.io/#/pages/using_chi?id=routing` — chi URL param extraction: `chi.URLParam(r, "code")` for POST, `r.URL.Query().Get("code")` for GET
  - `internal/domain/search.go` — SearchResponse struct

  **Acceptance Criteria**:
  - [ ] `GET /api/v1/search?code=ABC-123` → 200 JSON with movie + videos
  - [ ] `GET /api/v1/search` (missing code) → 400 `{"error":"code is required"}`
  - [ ] `GET /api/v1/search?code=` (empty code) → 400
  - [ ] `GET /api/v1/search?code=<script>` → 400 (invalid chars)
  - [ ] Response includes `took_ms` timing
  - [ ] Response includes `cache.hit` boolean
  - [ ] `go test ./internal/handler/... -v` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Valid search returns JSON with movie and videos
    Tool: Bash (curl)
    Preconditions: Server running with mock aggregator returning known data
    Steps:
      1. curl -s -H "X-API-Key: test-key" "http://localhost:8080/api/v1/search?code=ABC-123"
      2. Assert: HTTP 200
      3. Assert: response.code == "ABC-123"
      4. Assert: response.movie != null
      5. Assert: response.videos is array
      6. Assert: response.took_ms > 0
    Expected: 200 with complete JSON structure
    Evidence: .sisyphus/evidence/task-21-search-success.json

  Scenario: Missing code returns 400
    Tool: Bash (curl)
    Steps:
      1. curl -s -H "X-API-Key: test-key" "http://localhost:8080/api/v1/search"
      2. Assert: HTTP 400
      3. Assert: body contains "code"
    Expected: 400 validation error
    Evidence: .sisyphus/evidence/task-21-missing-code.txt

  Scenario: Unauthorized request returns 401
    Tool: Bash (curl)
    Steps:
      1. curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/api/v1/search?code=ABC-123"
      2. Assert: HTTP 401
    Expected: 401 Unauthorized
    Evidence: .sisyphus/evidence/task-21-unauthorized.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(handler): add search endpoint with JSON response and input validation`
  - Files: `internal/handler/search.go`, `internal/handler/response.go`, `internal/handler/search_test.go`, `cmd/api/main.go` (wire)

- [x] 22. **Dockerfile + docker-compose.yml (Render-Ready + Optional PostgreSQL)**

  **What to do**:
  - Create `Dockerfile` — multi-stage build with Render optimizations:
    ```dockerfile
    FROM golang:1.24-alpine AS builder
    WORKDIR /app
    COPY go.mod go.sum ./
    RUN go mod download
    COPY . .
    RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /javapi ./cmd/api

    FROM alpine:latest
    RUN apk --no-cache add ca-certificates
    COPY --from=builder /javapi /javapi
    COPY configs/ /configs/
    ENV GOMEMLIMIT=400MiB
    EXPOSE 8080
    ENTRYPOINT ["/javapi"]
    ```
  - Create `docker-compose.yml` — **two profiles**: database-less (default) + with PostgreSQL:
    ```yaml
    services:
      api:
        build: .
        ports: ["8080:8080"]
        environment:
          - CACHE_POSTGRES_ENABLED=false   # database-less by default
          - SCRAPERS_MAX_CONCURRENT=6      # Render-safe
          - GOMEMLIMIT=400MiB
          - AUTH_API_KEYS=test-key-123
        # For PG mode: uncomment db service + set POSTGRES_ENABLED=true
      # db:  # Optional PostgreSQL (for Render free tier PG addon)
      #   image: postgres:16-alpine
      #   ...
    ```
  - Create `.env.example` — template documenting all env vars (Render-compatible)
  - Create `render.yaml` — Render Blueprint for one-click deploy:
    ```yaml
    services:
      - type: web
        name: javapi
        env: go
        buildCommand: go build -o javapi ./cmd/api
        startCommand: ./javapi
        envVars:
          - key: GOMEMLIMIT
            value: 400MiB
          - key: SCRAPERS_MAX_CONCURRENT
            value: "6"
    ```
  - Update `Makefile` with `docker-build`, `docker-up`, `docker-down`, `render-deploy` targets

  **Must NOT do**: No Kubernetes, no CI/CD beyond Render blueprint

  **Recommended Agent Profile**: `quick` | **Skills**: [`golang-pro`]
  **Parallelization**: Wave 3 | **Blocks**: Task 23 | **Blocked By**: Task 1

  **Acceptance Criteria**:
  - [ ] `make docker-build` → image built
  - [ ] `make docker-up` → API starts (database-less mode by default), health check passes
  - [ ] PG profile: `docker compose --profile pg up` starts API + PostgreSQL
  - [ ] `render.yaml` valid (can be deployed to Render via Blueprint)
  - [ ] `.env.example` documents all env vars for Render deployment

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Docker Compose database-less mode
    Tool: Bash
    Steps: make docker-up && sleep 3 && curl -s http://localhost:8080/api/health && make docker-down
    Assert: 200 without PostgreSQL
    Evidence: .sisyphus/evidence/task-22-docker-dbless.txt
  ```

  **Commit**: YES (alone)
  - Message: `feat(deploy): add Render-ready Dockerfile, compose (db-less default), and render.yaml`
  - Files: `Dockerfile`, `docker-compose.yml`, `render.yaml`, `.env.example`, `Makefile` (update)

- [x] 23. **Integration Tests (End-to-End)**

  **What to do**:
  - Create `internal/handler/integration_test.go` — full integration test
  - Start server with all real dependencies (or test doubles)
  - Test flow:
    1. POST search request with valid code
    2. Verify 200 response
    3. Verify JSON structure: code, movie, videos[], cache, took_ms
    4. Verify cache: second request for same code returns cache.hit=true
    5. Verify auth: request without key → 401
    6. Verify graceful degradation: mock scraper failure → still 200 with partial results
  - Use `httptest.NewServer` for test server
  - Test takes real time (no mocking of scrapers unless needed for speed)

  **Must NOT do**: No external network calls in tests (use mock HTTP servers)

  **Recommended Agent Profile**:
  - **Category**: `deep` — Complex integration testing with multiple mock servers, cache verification, auth
  - **Skills**: [`golang-pro`]

  **Parallelization**: Wave 3 (sequential — depends on all other tasks) | **Blocks**: F1-F4 | **Blocked By**: Tasks 5, 21, 22

  **References**:
  - `https://pkg.go.dev/net/http/httptest` — httptest.NewServer for test HTTP server
  - `internal/handler/search.go` — search handler
  - `internal/middleware/auth.go` — auth middleware

  **Acceptance Criteria**:
  - [ ] Full search flow: request → response JSON structure verified
  - [ ] Cache verification: second request cache hit
  - [ ] Auth: 401 without key
  - [ ] Partial success: mock scraper failures → 200 with mixed results
  - [ ] Response time measured (took_ms > 0)
  - [ ] `go test ./internal/handler/... -v -tags=integration` → pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Integration test — full search flow
    Tool: Bash (go test)
    Steps: go test ./internal/handler/... -v -tags=integration -run TestIntegrationSearch
    Assert: Exit 0, all assertions pass
    Evidence: .sisyphus/evidence/task-23-integration.txt
  ```

  **Commit**: YES (alone)
  - Message: `test(integration): add end-to-end integration tests for search flow`
  - Files: `internal/handler/integration_test.go`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval.**

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go test ./...`. Review all Go files for: `panic()` outside safeSearch, `interface{}` instead of `any`, empty catch blocks, `fmt.Println` in production code, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
  Output: `Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Start server, run curl commands for all scenarios (health, search success, missing code, 401, graceful degradation). Test cache behavior. Test Docker deployment. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Docker [PASS/FAIL] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

All Wave 1 tasks committed individually. Wave 2 scrapers committed individually per site. Tasks 12 and 17 removed (site excluded).

- **1**: `chore: initialize Go project scaffolding with test infra`
- **2**: `feat(config): add proxy per-site, db-less, Render defaults`
- **3**: `feat(domain): define core types with multi-version + proxy support`
- **4**: `feat(javdb): implement JavDB API client with jdsignature auth`
- **5**: `feat(server): add chi router with middleware chain and graceful shutdown`
- **6**: `feat(auth): add API key authentication middleware`
- **7**: `feat(cache): implement in-memory cache with TTL support`
- **8**: `feat(scraper): add registry, code normalization, safeSearch, and cloudscraper CF bypass`
- **9**: `feat(cache): add optional PostgreSQL cache with database-less mode`
- **10**: `feat(scraper): add MISSAV scraper with CF test, proxy, multi-version`
- **11**: `feat(scraper): add Jable scraper with CF test, proxy, multi-version`
- ~~12~~: **REMOVED** — javtrailers excluded
- **13**: `feat(scraper): add JAVMENU scraper with CF test, proxy`
- **14**: `feat(scraper): add HAYAV scraper with CF test, proxy`
- **15**: `feat(scraper): add Supjav scraper with CF test, proxy`
- **16**: `feat(scraper): add javgg scraper with CF test, proxy`
- ~~17~~: **REMOVED** — JavBus excluded
- **18**: `feat(scraper): add AV01 JSON API scraper with CF test, proxy`
- **19**: `feat(scraper): add 7mmtv form POST scraper with CF test, proxy`
- **20**: `feat(aggregator): implement parallel aggregation with multi-version, cache, concurrency cap`
- **21**: `feat(handler): add search endpoint with JSON response and input validation`
- **22**: `feat(deploy): add Render-ready Dockerfile, compose (db-less default), render.yaml`
- **23**: `test(integration): add end-to-end integration tests for search flow`

---

## Success Criteria

### Verification Commands
```bash
# Build
go build -o javapi ./cmd/api                    # Expected: binary created

# Test
go test ./... -v                                 # Expected: all pass

# Run (database-less mode)
GOMEMLIMIT=400MiB POSTGRES_ENABLED=false ./javapi
# Expected: server starts, /api/health → 200, no PG error

# Docker (database-less default)
make docker-build && make docker-up
curl http://localhost:8080/api/health           # Expected: {"status":"ok"}
make docker-down

# Docker (with PostgreSQL)
docker compose --profile pg up
curl http://localhost:8080/api/health           # Expected: {"status":"ok"}
docker compose --profile pg down

# Search
curl -s -H "X-API-Key: test-key" "http://localhost:8080/api/v1/search?code=ABC-123"
# Expected: 200, JSON with movie + videos[] + multi-version support
```

### Final Checklist
- [ ] All 22 implementation tasks complete (2 removed)
- [ ] All 8 scrapers registered (skipped if CF bypass failed)
- [ ] JavDB API client working with valid jdsignature
- [ ] In-memory cache working (TTL, get/set/delete)
- [ ] Database-less mode: server starts without PostgreSQL
- [ ] Optional PostgreSQL: persistent cache when enabled
- [ ] Per-site proxy: BrightData-compatible URL config per site
- [ ] Cloudflare bypass: cloudscraper_go pre-test each site, skip if blocked
- [ ] Multi-version: sites returning original/cnsub/mosaic_reduce variants
- [ ] Aggregator: parallel scraping, concurrency capped (default 6), graceful degradation
- [ ] API key auth: 401 for missing/wrong key, health bypass
- [x] Health endpoint responds
- [x] Graceful shutdown on SIGTERM
- [x] Render.com compatibility: <512MB RAM, GOMEMLIMIT=400MiB, max_concurrent=6
- [x] Docker Compose: database-less by default, PG optional via profile
- [x] Excluded sites NOT present: javtrailers, FANZA, JavBus, JAVLib
- [x] All "Must Have" present, all "Must NOT Have" absent
- [x] All tests pass (unit + integration)
- [x] Evidence files present for all QA scenarios

---

## Post-Implementation Status (May 2026)

### Live Deployment
- **URL**: https://javapi-rxgl.onrender.com
- **Platform**: Render.com free tier
- **Repo**: https://github.com/ANDonekey/javapi

### Scraper Status (tested with MIDA-492 on Render)

| Scraper | CF Bypass | Proxy | Live Status |
|---------|-----------|-------|-------------|
| MISSAV | cloudscraper_go+CycleTLS | ✅ | — (startup timeout?) |
| Jable | ✅ | ✅ | ✅ success (1 source) |
| JAVMENU | ✅ | ✅ | not_found |
| HAYAV | ✅ | ✅ | blocked → **proxy added** |
| javgg | ✅ | ✅ | ✅ success (3 sources) |
| AV01 | ✅ | ✅ | error |
| 7mmtv | ✅ | ✅ | blocked → **proxy added** |

### Completed Post-Plan Improvements
- [x] **cloudscraper_go + CycleTLS** integrated (was stub, now real)
- [x] **CF marker fix**: removed "Cloudflare"/"challenge-platform" from blockers
- [x] **Per-site proxy**: YAML + `SCRAPER_{NAME}_PROXY_URL` env var
- [x] **ApplyConfig()**: post-init config injection in scraper registry
- [x] **Render deployment**: live, auto-deploy on push

### Known Improvements (for user to decide)

| Priority | Improvement | Effort | Impact |
|----------|-------------|--------|--------|
| ~~🔴 P0~~ | ~~验证 hayav/7mmtv 代理~~ | ✅ Done | 代理生效但数据中心IP仍被CF拦 |
| 🟡 P1 | **hayav 标记不支持** (CF不可绕过) | 小 | 清理代码 |
| 🟡 P1 | **javmenu 标记不支持** (内容欠佳) | 小 | 清理代码 |
| 🟡 P1 | **AV01 修复** — 搜索端点 + M3U8提取 | 中 | 7→5→6 爬虫可用 |
| 🟡 P1 | **Jable URL 格式修复** — pageUrl 含搜索URL | 小 | 用户体验 |
| 🟢 P2 | **缓存 TTL 可配置化** | 小 | — |
| 🟢 P2 | **健康检查中显示爬虫状态** | 小 | — |
| 🔵 P3 | **Ja3 指纹 + 住宅代理** 突破CF | 大 | — |
| 🔵 P3 | **更多爬虫站点** | 大 | — |

---

## P1 修复计划: 爬虫清理 + AV01/Jable 修复

### 1. 标记不支持站点 (小)

| 站点 | 原因 | 操作 |
|------|------|------|
| **hayav** | CF 无法绕过 (数据中心代理也被拦) | 移除 hayav 爬虫目录，从 main.go 移除 import |
| **javmenu** | 内容欠佳 (视频质量/可用性不足) | 移除 javmenu 爬虫目录，从 main.go 移除 import |
| **supjav** | CF 无法绕过 (之前已标记) | 已排除，确认无残留 |

### 2. AV01 修复 (中)

**当前问题**：搜索端点错误 + M3U8 提取逻辑错

**正确 API**（用户发现）：
```
POST https://www.av01.media/api/v1/videos/search?lang=cn&comp=true
Body: {"pagination":{"limit":20,"page":1},"query":"MIDA-492"}
Response: {"videos":[{"id":203184,"dvd_id":"MIDA-492","dmm_id":"mida00492"}]}
M3U8: /api/v1/videos/{id}/manifest/master.m3u8?hb=XXXX
```

**修改**：恢复 JSON API（原版类似但参数不同 lang=ja→cn, 加comp=true）

### 3. Jable URL 修复 (小)

**Bug 位置**：`jable/scraper.go:153` — `extractPlayers(doc)` 在搜索页文档上调用
- 搜索页有 `data-src` 或 `<script>` 含模板 URL `%QUERY%`
- `extractPlayers` 误提取为视频源 → 跳过视频页抓取
- **修复**：移除搜索页上的 `extractPlayers`，始终抓取视频页

### 4. MISSAV 垃圾内容修复 (中)

**Bug 1 — 垃圾内容**（行 141-157）：
- 缺少中文反爬检测 "大量垃圾内容"
- Title 不匹配时应直接返回 NotFound

**Bug 2 — 重复结果**（行 167-205）：
- `hasSub`/`hasLeak` 在垃圾页上误判
- 多版本检测应在 titleOk 后才执行
- 所有变体共享同一个 mutable sources slice

## P2 修复: M3U8 提取增强 (借鉴参考项目 missav)

### 问题
参考项目 `/home/henry/code/missav` 能成功提取 MISSAV/Jable 的 M3U8 URL，但 JAVprovide 不能。
JAVprovide 用 DOM 选择器 (goquery)，参考项目用**原始 HTML 正则搜索**。

### 修复 1: 通用 M3U8 正则提取 (中)

在 Jable 和 MISSAV 中添加 `extractM3U8FromRawHTML(html string) []string`:
```go
func extractM3U8FromRawHTML(html string) []string {
    re := regexp.MustCompile(`https?://[^'"\\\s<>]+\.m3u8[^'"\\\s<>]*`)
    return re.FindAllString(html, -1)
}
```
- 扫描整个原始 HTML body（不限 DOM 元素）
- 找到所有 `.m3u8` URL（包括藏在 script/data-* 中的）
- 附加到现有 DOM 提取结果之后

## M3U8 代理端点 — 为前端提供可播放的 M3U8

### 需求
嵌入页解析返回的 M3U8 URL 存在 CORS 跨域问题。前端无法直接 fetch。
需要 API 提供代理端点，获取 M3U8 内容并返回给前端。

### 方案

```
前端                     JAV API                 CDN
  │                        │                      │
  │ GET /api/v1/m3u8?url=<encoded>                │
  │──────────────────────→│                      │
  │                        │ GET <m3u8_url>       │
  │                        │─────────────────────→│
  │                        │      M3U8 content    │
  │                        │←─────────────────────│
  │    M3U8 content        │                      │
  │    (CORS headers)      │                      │
  │←──────────────────────│                      │
```

实现：
1. 新端点 `GET /api/v1/m3u8?url=<base64_encoded_m3u8_url>`
2. 获取目标 M3U8 文件
3. 返回内容 + CORS headers
4. 验证 URL 为已知 M3U8 域名（安全）
5. 缓存 M3U8 内容（短 TTL，如 60s）

### 背景
javgg 返回 `https://jav-vids.xyz/embed/me3diq35ekqx` 但这不是直接可播放的 M3U8。
需要进一步获取嵌入页，从中提取真正的 M3U8 URL。

参考项目 `missav/embed.go` 有完整的嵌入解析系统（5 个提取器 + 通用后备）。

### 验证范围
**仅 `jav-vids.xyz`** — 作为技术验证。成功后扩展到其他嵌入站点。

### 技术方案

```
javgg scraper 返回:
  https://jav-vids.xyz/embed/me3diq35ekqx

↓ 新增: EmbedResolver 服务

1. 匹配 embed host (jav-vids.xyz)
2. HTTP GET 嵌入页 (使用 CycleTLS)
3. 正则/JS解包 提取 M3U8 URL
4. 替换 VideoSource URL

javgg 现在返回:
  https://cdn.xxx.com/stream.m3u8  ← 真实 M3U8
```

### 实现步骤

1. 创建 `internal/embed/` 包
   - `resolver.go` — EmbedResolver 接口 + 注册表
   - `javvids.go` — jav-vids.xyz 提取器
   - `generic.go` — 通用正则后备

2. 注册提取器 (init 模式)
   ```go
   type EmbedExtractor interface {
       MatchHost(host string) bool
       Extract(ctx, pageURL string) ([]string, error)
   }
   ```

3. 在 aggregator 中集成
   - 收集所有 VideoSources
   - 对匹配 embed host 的 URL 调用解析
   - 替换为解析后的 M3U8 URL

4. 测试: `MIDA-492` → javgg 返回真实 M3U8 而非嵌入页 URL

Jable 使用纯 `net/http`（无 CF 绕过）。
替换为 `scraper.NewCFClient(proxyURL)`（与 MISSAV 相同）:
- init() 中创建 CF 客户端
- SetProxyConfig 中重建 CF 客户端

