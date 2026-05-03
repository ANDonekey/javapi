# Pending, Static, and To-Verify Notes

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

This file contains only material that the source notes treated as static inference, probable contract, unresolved branch behavior, or "to verify next".

## Inventory-only unverified routes

These paths appeared in the source note inventory but did not have fuller route sections in the same document.

### `GET /api/v1/debug_logging`

- Method: `GET`
- Path: `/api/v1/debug_logging`
- Auth state: unknown
- Required params: unknown
- Observed response fields: none
- Known errors: none recorded in the source notes

## Playback query details not yet live-completed

### `GET /api/v1/movies/%s/resume_play`

- Static inference:
  - AOT request builder includes `source_id`, `episode`, `resolution`, `platform=android`
- To verify next:
  - authenticated call with full playback query params

## User and account routes with pending contract details

### `POST /api/v1/users/forgot_password`

- Auth state in notes: public route shape confirmed
- Required params currently observed:
  - `email`
- Known errors:
  - empty request -> `ParameterInvalid: email`
- Probable body from source notes:
  - `email` or `username`
- To verify next:
  - exact field names
  - whether body is multipart or urlencoded

### `POST /api/v1/users/reset_password`

- Auth state in notes: public route shape confirmed
- Required params currently observed:
  - `email`
- Known errors:
  - empty request -> `ParameterInvalid: email`
- Probable body from source notes:
  - `email` or `username`
  - `code`
  - `password`
  - `password_confirmation`
- To verify next:
  - exact field names
  - pairing with `/users/forgot_password`

### `POST /api/v1/users/change_password`

- Additional probable body from source notes:
  - `old_password`
  - `password`
  - `password_confirmation`
- To verify next:
  - exact authenticated body fields

### `POST /api/v1/users/change_username`

- Additional probable body from source notes:
  - `username`
- To verify next:
  - exact field names and validation behavior

### `POST /api/v1/users/resend_activation_code`

- Auth state in notes: public route shape confirmed
- Required params currently observed:
  - `email`
- Known errors:
  - empty request -> `ParameterInvalid: email`
- Probable body from source notes:
  - `email` or registration-bound token/context
- To verify next:
  - whether it accepts email, username, or token context

### `POST /api/v1/users/activate_registration`

- Auth state in notes: public route shape confirmed
- Required params currently observed:
  - `email`
- Known errors:
  - empty request -> `ParameterInvalid: email`
- Probable body from source notes:
  - `code`
  - `email` or `username`
- To verify next:
  - exact required form fields

### `GET /api/v1/users/recent_viewed`

- To verify next:
  - pagination support
  - whether `current_page` exists on successful responses

### `GET /api/v2/users/review_movies`

- Auth state: APK-backed authenticated route family, live verified on `2026-04-29`
- Resolved from APK static analysis:
  - request path is `/api/v2/users/review_movies`
  - recovered query params include:
    - `status`
    - `type`
    - `sort_by`
    - `order_by`
    - `star`
    - `page`
    - `limit=48`
  - recovered status values include:
    - `want_watch`
    - `watched`
- Current frontend wiring:
  - navbar user menu `影片的 -> 想看` now requests:
    - `GET /api/v2/users/review_movies?status=want_watch&type=0&sort_by=create&order_by=desc&page=1&limit=48`
  - navbar user menu `影片的 -> 看過` now requests:
    - `GET /api/v2/users/review_movies?status=watched&type=0&sort_by=create&order_by=desc&page=1&limit=48`
  - current web client now exposes these APK-backed sort choices in the `sort` URL param:
    - `create`
    - `release-desc`
    - `release-asc`
    - `score`
    - `hit`
    - `want-watch-count`
    - `watched-count`
- To verify next:
  - whether `type=all` is ever emitted by the APK, or only `0|1|2|3`
  - whether `star` is optional or only used for rated sub-filters

### `GET /api/v1/users/collected_actors`

- Auth state: APK-backed authenticated route family, live verified on `2026-04-29`
- Resolved from APK static analysis:
  - request path is `/api/v1/users/collected_actors`
  - recovered query params include:
    - `page`
    - `limit=60`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 演員` now lands on `/account/collected-actors?page=1`
- To verify next:
  - whether later pages keep returning the same full actor item contract when `avatar_url` is null on page 1

### `GET /api/v1/users/collected_lists`

- Auth state: APK-backed authenticated route family
- Resolved from APK static analysis:
  - request path is `/api/v1/users/collected_lists`
  - recovered query params include:
    - `sort_by`
    - `page`
    - `limit=48`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 清單` now lands on `/account/collected-lists?page=1`
  - the current web client renders the returned list items with the same card style as other list pages
- To verify next:
  - final response field shape
  - whether the route shares the same item contract as `/api/v1/lists`

### `GET /api/v1/users/collected_makers`

- Auth state: APK-backed authenticated route family, live verified on `2026-04-29`
- Resolved from APK static analysis:
  - request path is `/api/v1/users/collected_makers`
  - recovered query params include:
    - `page`
    - `limit=48`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 片商` now lands on `/account/collected-makers?page=1`
- To verify next:
  - whether maker items ever include extra metadata beyond `id`, `type`, `name`, `videos_count`

### `GET /api/v1/users/collected_series`

- Auth state: APK-backed authenticated route family, live verified on `2026-04-29`
- Resolved from APK static analysis:
  - request path is `/api/v1/users/collected_series`
  - recovered query params include:
    - `page`
    - `limit=48`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 系列` now lands on `/account/collected-series?page=1`
- To verify next:
  - whether series items ever include extra metadata beyond `id`, `type`, `name`, `videos_count`

### `GET /api/v1/users/collected_directors`

- Auth state: APK-backed authenticated route family, live verified on `2026-04-29`
- Resolved from APK static analysis:
  - request path is `/api/v1/users/collected_directors`
  - recovered query params include:
    - `page`
    - `limit=48`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 導演` now lands on `/account/collected-directors?page=1`
- To verify next:
  - collect a non-empty authenticated sample, since the current test account returned `directors=[]`

### `GET /api/v1/users/collected_codes`

- Auth state: APK-backed authenticated route family, live verified on `2026-04-29`
- Resolved from APK static analysis:
  - request path is `/api/v1/users/collected_codes`
  - recovered query params include:
    - `page`
    - `limit=48`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 番號` now lands on `/account/collected-codes?page=1`
- To verify next:
  - whether code items ever include extra metadata beyond `id`, `name`, `videos_count`, `type`

### `GET /api/v1/users/transaction_logs`

- Probable query params from source notes:
  - `page`
  - `type`
- To verify next:
  - supported filters and pagination

### `GET /api/v1/users/unpaid_tickets`

- Auth state in notes: authenticated route family
- Observed response fields: none
- Known errors:
  - current host -> `HTTP 404`
- To verify next:
  - whether result shape is order-oriented or plan-oriented

### `POST /api/v1/users/feedback`

- Additional probable body from source notes:
  - `content`
  - `contact`
  - `category`

## Wallet and payment pending details

### `POST /api/v1/wallets/verify_email`

- Additional probable body from source notes:
  - `code`
  - `email`
- To verify next:
  - final accepted body

### `POST /api/v1/wallets/send_verification_email`

- To verify next:
  - rate-limit behavior
  - whether any body is required beyond authenticated `POST`

### `POST /api/v1/wallets/bind_withdraw_account`

- Additional probable body from source notes:
  - `type`
  - `account`
  - `name`
  - `bank_code`
  - `chain_type`
  - `code`
- To verify next:
  - supported withdraw account types
  - validation fields

### `GET /api/v1/wallets/binded_withdraw_accounts`

- To verify next:
  - item shape for `accounts`

### `DELETE /api/v1/wallets/unbind_withdraw_account/%s`

- To verify next:
  - whether confirmation fields are required

### `GET /api/v1/wallets/withdraw_logs`

- To verify next:
  - pagination
  - status fields

### `GET /api/v1/wallets/rebate_logs`

- To verify next:
  - overlap with promotion logs or commission records

### `GET /api/v1/wallets/usdt_chain_types`

- To verify next:
  - whether chain items include extra metadata beyond the observed `["TRC20"]`

### `GET /api/v1/wallets/sfpay_banks`

- To verify next:
  - whether the route is region- or channel-specific

### `POST /api/v2/wallets/withdraw`

- Additional probable body from source notes:
  - `amount`
  - `withdraw_account_id`
  - `chain_type` or `bank_code`
  - `code`
- To verify next:
  - minimum required fields
  - invalid-amount behavior

### `GET /api/v4/plans`

- Source-note conflict preserved:
  - bulk sweep says public signed route
  - endpoint section says it also requires `authorization`
- To verify next:
  - whether it supersedes `/api/v3/plans`
  - exact structure differences from `/api/v3/plans`

### `POST /api/v2/plans/payment_order`

- Additional probable body from source notes:
  - `plan_id`
  - `platform_id`
  - `channel_id`
  - `method_id`
  - `price` or `price_id`
- To verify next:
  - exact required ids
  - whether anonymous order creation is allowed

### `POST /api/v3/plans/payment_order`

- Additional probable body from source notes:
  - `plan_id`
  - `platform_id`
  - `channel_id`
  - `method_id`
- To verify next:
  - structural differences from v2

## Movie discovery and ranking pending details

### `GET /api/v2/search`

- Method: `GET`
- Path: `/api/v2/search`
- Auth state: public signed route; typed result branches are now live-confirmed in `public/search-and-filters.md`
- Confirmed unresolved parts:
  - field-presence guarantees are still incomplete for typed arrays, especially optional fields such as `avatar_url`, `name_zht`, and `other_name`
- To verify next:
  - whether any typed branch ever returns `current_page`, `total`, `total_pages`, or another stable end-of-list signal for other keywords or larger datasets
  - whether any typed branch has a real server-side sort or secondary filter parameter beyond `q`, `page`, and `type`

### `GET /api/v1/movies/recommend_periods`

- To verify next:
  - whether this feeds the UI filter for `/movies/recommend`

### `GET /api/v1/movies/may_also_like`

- Probable query params from source notes:
  - `movie_id`
  - `page`
- To verify next:
  - final required parameter set

### `GET /api/v1/movies/tags`

- Method: `GET`
- Path: `/api/v1/movies/tags`
- Auth state: public signed route with category defaults now confirmed; this section only tracks the unresolved parts of the filter grammar
- Required params currently observed:
  - `filter_by`
  - currently successful live examples also included `sort_by=release`, `order_by=desc`, `page=1`, `limit=48`
- Observed response fields:
  - `movies`
  - `has_collected`
  - `current_page`
- Confirmed top-row category mapping:
  - `Censored -> 0`
  - `Uncensored -> 1`
  - `Western -> 2`
  - `FC2 -> 3`
  - `Carton/Anime -> 4`
- Confirmed working category patterns:
  - `0:t:p::::`
  - `1:t:p::::`
  - `2:t:p::::`
  - `3:t:p::::`
  - `4:t:p::::`
  - and download-oriented variants replacing slot 3 with `m`
- Confirmed syntax baseline:
  - `{type}:t:{main}:{extra}:{year}:{duration}:{month}`
- Confirmed slot meanings:
  - slot 1 `{type}` = top-row category
  - slot 2 `t` = tag-filter mode
  - slot 3 `{main}` = one of `p / m / c / s / i / v`
  - slot 4 `{extra}` = lower-sheet tag ids from non-core groups such as `subject`, `role`, `cloth`, `body`, `behavior`, `play_method`, `category`, `other`, or type-specific groups
  - slot 5 `{year}` = year ids such as `2025`
  - slot 6 `{duration}` = duration ids such as `lt-45`, `45-90`, `90-120`, `gt-120`
  - slot 7 `{month}` = month ids such as `1` or `12`
- Confirmed live examples:
  - `0:t:p:23:::` proves slot 4 is the generic `extra` slot
  - `0:t:p::2025::` proves slot 5 is `year`
  - `0:t:p:::lt-45:` proves slot 6 is `duration`
  - `0:t:p::::1` proves slot 7 is `month`
- Still unresolved:
  - exact group-to-id mapping coverage for every type-specific lower-sheet section
  - exact server rule behind slot 6 `duration` multi-value parsing, which is currently inconsistent across orderings
  - whether any special precedence exists when `extra`, `year`, `duration`, and `month` are combined together in dense queries
- Static evidence retained:
  - category provider injects `group="main", id="m", name="Download"` and `group="main", id="p", name="Playable"`
  - category page contains a `movie_default_filter_download` branch that interpolates `:t:m::::` in the truthy branch and `:t:p::::` in the false branch before calling `getCategoryMovies`
  - direct APK transport proof shows the category top row uses actual page index `0..4` as `/api/v2/tags?type`
  - provider state buckets are explicitly named `main`, `year`, `month`, `duration`, plus a catch-all list for additional tags
  - category-page request builder appends the provider buckets in the order `main -> extra -> year -> duration -> month`
- To verify next:
  - expand the per-type `extra` matrix beyond the currently verified pairs, especially `type=1` `other` combinations and `type=3` FC2 `tag` combinations
  - live behavior of these APK-backed list-derived patterns from `MyListsDetailPagePresenter::getListsMovies`:
    - `0:l:{list_id}:`
    - `0:l:{list_id}:p`
    - `0:l:{list_id}:m`
    - `0:l:{list_id}:c`
  - additional `duration` multi-value probes beyond `lt-45,45-90`, `45-90,lt-45`, and `45-90,90-120`

### `GET /api/v1/reviews/hotly`

- To verify next:
  - pagination
  - whether more nested movie info exists

### `GET /api/v1/rankings`

- Method: `GET`
- Path: `/api/v1/rankings`
- Auth state: resolved public on current host for the verified movie-ranking branches
- Required params currently observed:
  - `type`
  - `period`
- Observed response fields:
  - `current_page`
  - `movies`
- Known errors:
  - missing `type` -> `ParameterInvalid: type`
  - `type=0` -> `ParameterInvalid: period`
  - `type=1` -> `ParameterInvalid: period`
  - `type=0&period=552` -> `JWTVerificationError`
- Resolved on 2026-04-29:
  - `type=0/1/2/3` with `period=daily|weekly|monthly` all returned `HTTP 200`
  - current frontend now treats these as the public `有码 / 无码 / 欧美 / FC2` movie-ranking branches
- Resolved from APK static analysis on 2026-04-29:
  - `TOP250` does not belong to this route family
  - the dedicated `TOP250` page stays on `/api/v1/movies/top`

### `GET /api/v1/movies/top`

- Auth state: authenticated route already live-verified, but `TOP250`-specific filter behavior is still only partially live-verified
- Resolved from APK static analysis on 2026-04-29:
  - the dedicated `TOP250` page presenter calls `/api/v1/movies/top`
  - the rankings-section `TOP250` branch presenters also call `/api/v1/movies/top`
  - recovered `TOP250` query builder includes:
    - `start_rank`
    - `type`
    - `type_value`
    - `ignore_watched`
    - `page`
    - `limit=50`
  - recovered `TOP250` UI labels include:
    - `Can play`
    - `start ranking:`
    - `Unmarked「watched」`
  - current web client now exposes the APK-confirmed subset:
    - `start_rank=1|51|101|151|201`
    - `ignore_watched=true|false`
    - `type=all`
    - `type_value=''`
    - `limit=50`
- To verify next:
  - exact live values of `type` and `type_value` for the `TOP250` filters
  - whether `Can play` maps to `type_value`, `type`, or another hidden branch
  - actual result differences for:
    - `ignore_watched=true|false`
    - `start_rank=1|51|101|151|201`
  - whether `period=daily|weekly|monthly` is really meaningful for the dedicated `TOP250` page on the current host

### `GET /api/v1/rankings/actors`

- To verify next:
  - full allowed `filter_by` values beyond observed `views` and `movies_count`

### `GET /api/v1/series/letters`

- Resolved in scope:
  - the route is APK-backed and public-signed
- To verify next:
  - how the APK series page turns a selected letter into a concrete list request
  - whether the letter selection further mutates `/api/v1/series` with a hidden query param or uses another route branch

### `GET /api/v1/rankings/playbackP`

- Method: `GET`
- Path: `/api/v1/rankings/playbackP`
- Auth state: unknown
- Required params: unknown
- Observed response fields: none
- Known errors:
  - current host -> `HTTP 404`
- Static inference:
  - source notes list it as a playback-related ranking variant
- To verify next:
  - whether it is obsolete, host-specific, or incorrectly recovered from static strings

## Actor, director, maker, publisher, series pending details

### `POST /api/v1/actors/batch_uncollection`

- Method: `POST`
- Path: `/api/v1/actors/batch_uncollection`
- Auth state: authenticated family in source notes
- Required params:
  - probable `actor_ids`
- Observed response fields: none
- Known errors:
  - current host -> `HTTP 404`

### `GET /api/v1/directors`

- To verify next:
  - whether list/detail fully mirror actor APIs

### `GET /api/v1/makers`

- To verify next:
  - fuller maker detail response shape

### `GET /api/v1/publishers/%s`

- To verify next:
  - whether movie list data is available beyond the observed metadata-first payload

## Lists, codes, tags, and ads pending details

### `GET /api/v1/following_tags`

- Method: `GET`
- Path: `/api/v1/following_tags`
- Auth state: authenticated family in source notes
- Required params: unknown
- Observed response fields: none
- Known errors:
  - current host -> `HTTP 404`

### `GET /api/v1/following_tags/%s`

- Method: `GET`
- Path: `/api/v1/following_tags/%s`
- Auth state: authenticated family in source notes
- Required params:
  - path param `%s` = tag id
- Observed response fields: none
- Known errors:
  - current host -> `HTTP 404`

### `POST /api/v1/following_tags/%s/sort`

- Method: `POST`
- Path: `/api/v1/following_tags/%s/sort`
- Auth state: authenticated family in source notes
- Required params: unknown
- Observed response fields: none
- Known errors:
  - current host -> `HTTP 404`
- Probable body from source notes:
  - `sort`, `direction`, or ordinal index

### `POST /api/v1/following_tags/batch_destroy`

- Additional probable body from source notes:
  - `tag_ids`

### `POST /api/v1/following_tags/batch_push`

- Additional probable body from source notes:
  - `tag_ids`
  - possible push flag

### `POST /api/v1/ads/splash_log`

- Additional probable body from source notes:
  - `ad_id`
  - `action`
  - `position`

## Article and utility pending details

### `GET /api/v1/articles`

- To verify next:
  - whether more filters or pagination params exist

### `GET /api/v1/magnet_apps`

- To verify next:
  - whether app items include fields beyond `name`, `description`, `recommended`, `links`

### `POST /api/v1/logs/movie_played`

- Additional probable body from source notes:
  - `movie_id`
  - `source_id`
  - `episode`
  - `resolution`
  - `progress`
  - `operation`

### `POST /api/v2/logs/activated`

- To verify next:
  - required analytics fields beyond observed `device_uuid`

## Source-note backlog preserved

- Map which endpoints require `authorization` in addition to `jdsignature`
- (Resolved 2026-04-26) `/api/v2/search_image` — now documented in `public.md` as `POST` with multipart `image` field
- (Resolved 2026-04-26) `GET /api/v1/movies/%s/play` — verified query params `source_id`, `from_rankings`, `operation`; source_id behaviour documented in `authenticated.md`; successful playback response still unobserved without VIP
- Verify whether review pagination or ordering params exist
- Test `/resume_play` with a real logged-in token and full query params
- Add a higher-level parameter matrix for paginated feeds, ranking filters, and playback query fields

