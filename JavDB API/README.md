# JavDB API Docs

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

These files are a split of the source notes and subsequent category-filter verification updates.

## Files

- [auth.md](/F:/codx/javdbweb/docs/api/auth.md): request signing, auth token usage, common failure modes, auth matrix
- [public.md](/F:/codx/javdbweb/docs/api/public.md): live verified routes that worked as public signed routes or public entrypoints
- [authenticated.md](/F:/codx/javdbweb/docs/api/authenticated.md): live verified routes that required authenticated context or were verified only after login
- [errors.md](/F:/codx/javdbweb/docs/api/errors.md): summarized error and action values observed in the source notes
- [pending.md](/F:/codx/javdbweb/docs/api/pending.md): static inference, probable contracts, unresolved branches, and "to verify next" notes from the source file
- [test-plan.md](/F:/codx/javdbweb/docs/api/test-plan.md): next-step verification plan regrouped from the source notes
- [objects.md](/F:/codx/javdbweb/docs/api/objects.md): reusable response objects explicitly documented in the source notes

## Frontend Integration

Currently integrated in the React app:

- `GET /api/v1/startup`: search bar hot keywords
- `GET /api/v1/movies/latest`: home page default feed
  - home toolbar availability is wired with confirmed APK values:
    - `all -> filter_by=all`
    - `playable -> filter_by=can_play`
    - `magnet -> filter_by=magnets`
    - `subtitle -> filter_by=subtitle`
  - movie list view-mode now switches between `cover_url` (large cover) and `thumb_url` (small cover)
- `GET /api/v1/movies/recommend`: home page hot tab
- `GET /api/v1/movies/tags`: categories page feed with confirmed top-row mappings:
  - `Censored -> 0:t:p::::`
  - `Uncensored -> 1:t:p::::`
  - `Western -> 2:t:p::::`
  - `FC2 -> 3:t:p::::`
  - `Carton/Anime -> 4:t:p::::`
  - categories page sort buttons now map to live params:
    - `magnet-updated -> sort_by=update`
    - `released -> sort_by=release&order_by=desc`
  - list view-mode now switches between `cover_url` (large cover) and `thumb_url` (small cover)
- `GET /api/v2/tags`: category filter dictionary with confirmed `type` mapping:
  - `0=Censored`
  - `1=Uncensored`
  - `2=Western`
  - `3=FC2`
  - `4=Carton/Anime`
  - categories page now renders live filter groups from `category_id` and uses selected tag ids to build `filter_by`
- `GET /api/v2/search`: search results on categories page
  - `type=all` renders `movies[]`
  - `type=actor` renders `actors[]`
  - `type=series` renders `series[]`
  - `type=maker` renders `makers[]`
  - `type=director` renders `directors[]`
  - `type=code` renders `codes[]`
  - API-level detail landing routes are now confirmed as `/api/v1/actors/%s`, `/api/v1/series/%s`, `/api/v1/makers/%s`, `/api/v1/directors/%s`, and `/api/v1/codes/%s`
  - current frontend still links typed entity results back into a normal movie search using the clicked name/code
- Advanced search page now submits into the existing movie list stack using:
  - `GET /api/v2/search` as the primary search candidate feed
  - confirmed APK-backed search query params currently used by frontend:
    - `q`
    - `type=movie`
    - `movie_type`
    - `movie_filter_by=all|can_play|magnets|subtitle`
    - `movie_sort_by=relevance|release|update|score`
    - `page`
    - `limit`
  - actor autocomplete now prefers `GET /api/v2/search?type=actor` and falls back to extracting unique actor names from the normal movie-search branch when the typed branch is sparse for the current keyword
  - `GET /api/v4/movies/%s` as a detail fallback when the form needs fields not stably exposed in the search list, such as `publisher` and `series`
- `GET /api/v4/movies/%s`: movie detail base data
  - detail-page links now use entity detail endpoints before showing corresponding movie lists:
    - actor -> `GET /api/v1/actors/%s`
    - director -> `GET /api/v1/directors/%s`
    - maker -> `GET /api/v1/makers/%s`
- `GET /api/v1/movies/%s/magnets`: movie detail magnet list
- `GET /api/v1/movies/%s/reviews`: movie detail review list

Current frontend gaps:

- `filter_by` is now confirmed as `{type}:t:{main}:{extra}:{year}:{duration}:{month}`
- remaining `filter_by` gaps are limited to multi-value joining behavior and full lower-sheet group-to-`extra` mappings
- `/api/v1/movies/latest` has confirmed availability-style `filter_by` values, but its sort parameter contract is still not documented as clearly as category feed sorting
- category-page sort is wired for the two currently exposed UI buttons, but `release asc`, `score`, `hit`, `want_watch_count`, and `watched_count` are still not exposed in the current UI
- search dropdown typed results are now rendered, and the backend detail routes are confirmed, but the dedicated actor/series/maker/director/code detail navigation is still not wired in the current UI
- advanced search is now anchored on `/api/v2/search`, but the current public docs still do not expose a full dedicated advanced-search parameter contract; score/date/name filters beyond the confirmed search params are therefore still applied within the fetched candidate window rather than through a fully documented server-native advanced-search API
- review pagination is still undocumented, so the detail page currently loads only the default first response body

## Split Rules

- `public.md` and `authenticated.md` contain live verified interfaces only.
- `pending.md` contains content explicitly marked or described as static inference, probable, or to-verify-next.
- Mixed or conflicting route notes stay in `pending.md` even if some live verification exists.
- Field names are preserved from the source notes.
- No `src` files were modified as part of this split.
