# API Verification Test Plan

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

This plan is generated only from the source note backlog, "to verify next" items, and unresolved live-verification branches.

## P0

- [RESOLVED 2026-04-26] `/api/v2/search_image` — verified as `POST` multipart `image` upload; see `public.md` for full contract.
- [RESOLVED 2026-04-26] `GET /api/v1/movies/top` — verified with auth; full response shape including `ranking` field documented in `authenticated.md`.
- [RESOLVED 2026-04-26] `GET /api/v1/movies/recommend` — verified as global (non-personalized) public feed; documented in `public.md`.
- [RESOLVED 2026-04-26] `GET /api/v1/movies/%s/play` — verified query params and source_id behavior (VIP-gated); documented in `authenticated.md`.
- Continue `/api/v1/movies/tags` verification only for unresolved `duration` multi-value parsing and denser per-type `extra` matrices.
- Confirm the mixed auth behavior of `/api/v1/rankings`, especially the difference between public `type=1` behavior and auth-gated `type=0` behavior.

## P1

- [PARTIALLY RESOLVED 2026-04-28] `/api/v2/search` typed result branches:
  - confirmed canonical follow-up routes:
    - `actor -> /api/v1/actors/%s`
    - `series -> /api/v1/series/%s`
    - `maker -> /api/v1/makers/%s`
    - `director -> /api/v1/directors/%s`
    - `code -> /api/v1/codes/%s`
  - current live evidence shows typed responses do not expose `current_page`, `total`, or `total_pages`
  - current live evidence shows `page`, `sort_by`, `order_by`, `filter_by`, and `page_size` are accepted but had no observable effect in the tested typed queries
  - keep only these follow-up checks open:
    - whether any larger typed dataset ever returns a stable pagination boundary signal
    - whether any typed branch has a real server-side sort or secondary filter parameter under other keywords or result sizes
- Verify whether `/api/v1/movies/tags` always needs `sort_by=release_date&order_by=desc` for accepted `main:*` requests, because current reduced-param probes return `HTTP 500`.
- Top-row category mapping is now confirmed as `Censored=0`, `Uncensored=1`, `Western=2`, `FC2=3`, `Carton/Anime=4`; remaining work is the lower filter-sheet slot encoding.
- Refine the slot-6 `duration` multi-value rule, because `lt-45,45-90` behaves like a union while `45-90,lt-45` and `45-90,90-120` fall back to default-looking feeds.
- Expand the verified `extra` matrix beyond the current cross-group pairs (`subject+role`, `subject+body`, `body+behavior`, `role+cloth`, `category+body`, `play_method+category`, `play_method+role`, `play_method+body`, `other+cloth`) and map more ids per lower-sheet group.
- Resolve list-derived `filter_by` values used by `MyListsDetailPagePresenter::getListsMovies`.
- Verify pagination or ordering parameters for `/api/v1/movies/%s/reviews`.
- Verify pagination and nested movie context completeness for `/api/v1/reviews/hotly`.
- Confirm the full allowed `filter_by` set for `/api/v1/rankings/actors`.
- Verify whether `/api/v1/movies/may_also_like` only needs `movie_id` or additional context.
- Check whether `/api/v1/users/recent_viewed`, `/api/v1/users/transaction_logs`, `/api/v1/wallets/withdraw_logs`, and `/api/v1/wallets/rebate_logs` expose confirmed pagination fields beyond the currently observed data.

## P2
- Verify item shapes for metadata-style routes that are present but still incomplete in the notes:
  - `/api/v2/tags`
  - `/api/v1/articles`
- Re-check inventory-only or route-shape-only paths that remain unresolved:
  - `/api/v1/rankings/playbackP`
  - `/api/v1/actors/batch_uncollection`
  - `/api/v1/following_tags`
  - `/api/v1/following_tags/%s`
  - `/api/v1/following_tags/%s/sort`
  - `/api/v1/codes/%s`
