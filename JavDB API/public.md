# Public Live Verified Endpoints

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

This file is now an index for public live verified routes. Detailed route notes were moved into `docs/api/public/` so the public documentation can grow without turning back into a single large file.

Important navigation note:
- The app home page section labeled `近期磁链更新` maps to `GET /api/v1/movies/latest` with the home-strip query shape documented in [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md).
- `GET /api/v1/rankings/playback` is documented separately and should be treated as a dedicated playback-ranking page, not the home page `近期磁链更新` strip.

## Detail Files

- [core.md](/F:/codx/javdbweb/docs/api/public/core.md): startup, about, helps, plans, sessions, ads, utility routes
- [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md): movie detail, magnets, reviews, recommendations, hot reviews
- [search-and-filters.md](/F:/codx/javdbweb/docs/api/public/search-and-filters.md): keyword search, magnet search, tag catalogs, category filters, image search
- [people.md](/F:/codx/javdbweb/docs/api/public/people.md): actors, directors, makers, publishers, series, actor rankings
- [lists-and-articles.md](/F:/codx/javdbweb/docs/api/public/lists-and-articles.md): public lists and articles

## Route Index

| Method | Path | Auth state | Detail |
| --- | --- | --- | --- |
| `GET` | `/api/v1/startup` | Public signed route | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `GET` | `/api/v1/about` | Public signed route | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `GET` | `/api/v1/helps` | Public signed route | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `GET` | `/api/v3/plans` | Public signed route | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `POST` | `/api/v1/sessions` | Public login entrypoint | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `GET` | `/api/v1/movies/latest` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v4/movies/%s` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v1/movies/%s/magnets` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v1/movies/%s/reviews` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `POST` | `/api/v1/movies/%s/reviews/%s/like` | Public for the current test round | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v1/movies/recommend` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v1/movies/recommend_periods` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v1/rankings/playback` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v1/reviews/hotly` | Public signed route | [movies.md](/F:/codx/javdbweb/docs/api/public/movies.md) |
| `GET` | `/api/v2/search` | Public signed route | [search-and-filters.md](/F:/codx/javdbweb/docs/api/public/search-and-filters.md) |
| `GET` | `/api/v1/search_magnet` | Public signed route | [search-and-filters.md](/F:/codx/javdbweb/docs/api/public/search-and-filters.md) |
| `GET` | `/api/v2/tags` | Public signed route | [search-and-filters.md](/F:/codx/javdbweb/docs/api/public/search-and-filters.md) |
| `GET` | `/api/v1/movies/tags` | Public signed route | [search-and-filters.md](/F:/codx/javdbweb/docs/api/public/search-and-filters.md) |
| `POST` | `/api/v2/search_image` | Public signed route | [search-and-filters.md](/F:/codx/javdbweb/docs/api/public/search-and-filters.md) |
| `GET` | `/api/v1/actors` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/actors/%s` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/actors/recommend` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/directors` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/directors/%s` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/makers` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/makers/%s` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/publishers/%s` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/series` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/series/%s` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/series/letters` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/rankings/actors` | Public signed route | [people.md](/F:/codx/javdbweb/docs/api/public/people.md) |
| `GET` | `/api/v1/lists/related` | Public signed route | [lists-and-articles.md](/F:/codx/javdbweb/docs/api/public/lists-and-articles.md) |
| `GET` | `/api/v1/lists/%s` | Public signed route | [lists-and-articles.md](/F:/codx/javdbweb/docs/api/public/lists-and-articles.md) |
| `GET` | `/api/v1/articles` | Public signed route | [lists-and-articles.md](/F:/codx/javdbweb/docs/api/public/lists-and-articles.md) |
| `GET` | `/api/v1/articles/%s` | Public signed route | [lists-and-articles.md](/F:/codx/javdbweb/docs/api/public/lists-and-articles.md) |
| `GET` | `/api/v1/ads` | Public signed route | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `POST` | `/api/v1/ads/splash_log` | Public route shape confirmed | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `GET` | `/api/v1/magnet_apps` | Public signed route | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
| `POST` | `/api/v2/logs/activated` | Public signed route shape confirmed | [core.md](/F:/codx/javdbweb/docs/api/public/core.md) |
