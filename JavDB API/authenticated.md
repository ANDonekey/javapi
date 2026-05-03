# Authenticated Live Verified Endpoints

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

This file contains live verified interfaces that either returned `JWTVerificationError` without login or were only verified with authenticated requests in the source notes.

## `POST /api/v1/movies/%s/reviews/%s/report`

- Method: `POST`
- Path: `/api/v1/movies/%s/reviews/%s/report`
- Auth state: Auth required
- Required params:
  - path param `%s` = movie id
  - path param `%s` = review id
- Observed response fields: none observed from successful authenticated call
- Known errors:
  - `GET` on this route -> `404`
  - unauthenticated `POST` -> `JWTVerificationError`

## `DELETE /api/v1/movies/%s/reviews/%s`

- Method: `DELETE`
- Path: `/api/v1/movies/%s/reviews/%s`
- Auth state: Auth required
- Required params:
  - path param `%s` = movie id
  - path param `%s` = review id
- Observed response fields: none observed from successful authenticated call
- Known errors:
  - `GET` on this route -> `404`
  - unauthenticated `DELETE` -> `JWTVerificationError`

## `GET /api/v1/movies/%s/play`

- Method: `GET`
- Path: `/api/v1/movies/%s/play`
- Auth state: Auth required (returns `JWTVerificationError` without `authorization`)
- Required params:
  - path param `%s` = movie id
  - `source_id` (query) — streaming source identifier; determines playback availability
- Optional query params:
  - `from_rankings` — candidate: `"true"`, `"false"`
  - `operation` — candidate: `"play"`
- Observed `source_id` behavior (tested 2026-04-26 with non-VIP account on host `jdforrepam.com`):
  - `source_id=1, 2, 4` → recognised the movie but requires VIP: `PermissionDeniedToPayment: 需要VIP权限才能访问此内容`
  - `source_id=0, 3, 5` → no online source available for this movie: `"此影片不支持在线播放"`
- Expected successful response (inferred, not observed — requires active VIP):
  - Success response likely contains streaming URL / playback token
- Known errors:
  - unauthenticated `GET` -> `JWTVerificationError`
  - `POST` on this route -> `404`
  - authenticated `GET` without `source_id` -> `"此影片不支持在线播放"`
  - authenticated `GET` with a valid but VIP-gated `source_id` -> `PermissionDeniedToPayment: 需要VIP权限才能访问此内容`

## `GET /api/v1/movies/%s/resume_play`

- Method: `GET`
- Path: `/api/v1/movies/%s/resume_play`
- Auth state: Auth required
- Required params:
  - path param `%s` = movie id
- Observed response fields: none observed from successful authenticated call
- Known errors:
  - unauthenticated `GET` -> `JWTVerificationError`
  - `POST` on this route -> `404`
  - authenticated `GET /resume_play?source_id=4&episode=1&resolution=720p&platform=android` returned `HTTP 200`, `Content-Type: text/plain; charset=utf-8`, body `ERROR:PermissionDeniedToPayment`

## `GET /api/v1/users`

- Method: `GET`
- Path: `/api/v1/users`
- Auth state: Auth required
- Required params: none observed
- Observed response fields:
  - `data.user`
  - `data.banner_type`
  - observed `data.user` fields: `id`, `username`, `email`, `is_vip`, `vip_expired_at`, `want_watch_count`, `watched_count`, `promote_users_count`, `share_url`, `promotion_code`, `promotion_days`, `checkin_days`, `last_checkin_at`
- Known errors:
  - plain signed request without login -> `JWTVerificationError`

## `POST /api/v1/users/change_password`

- Method: `POST`
- Path: `/api/v1/users/change_password`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `old_password`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: old_password`

## `POST /api/v1/users/change_username`

- Method: `POST`
- Path: `/api/v1/users/change_username`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `current_password`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: current_password`

## `GET /api/v1/users/additional`

- Method: `GET`
- Path: `/api/v1/users/additional`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - `reports_count`
  - `deleted_comments_count`
  - `muted_count`
  - `max_muted_count`
  - `uncorrected_count`
  - `corrections_count`
- Known errors: none observed

## `GET /api/v1/users/recent_viewed`

- Method: `GET`
- Path: `/api/v1/users/recent_viewed`
- Auth state: Auth required
- Required params: none observed
- Observed response fields:
  - `movies`
- Current frontend wiring:
  - navbar user menu `影片的 -> 近期瀏覽` now lands on `/account/recent-viewed?page=1`
  - the current web client renders this route as a movie grid in the logged-in account page
- Known errors:
  - plain signed request without login -> `JWTVerificationError`

## `GET /api/v1/users/transaction_logs`

- Method: `GET`
- Path: `/api/v1/users/transaction_logs`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - `notice`
  - `logs`
  - `current_page`
- Known errors: none observed

## `GET /api/v1/users/promotion_logs`

- Method: `GET`
- Path: `/api/v1/users/promotion_logs`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - `logs`
  - `current_page`
- Known errors: none observed

## `POST /api/v1/users/feedback`

- Method: `POST`
- Path: `/api/v1/users/feedback`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `content`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: content`

## `GET /api/v1/wallets`

- Method: `GET`
- Path: `/api/v1/wallets`
- Auth state: Auth required
- Required params: none observed
- Observed response fields:
  - `minimum_withdraw_amount`
  - `maximum_withdraw_amount`
  - `withdraw_fee`
  - `current_level`
  - `current_rate`
  - `next_level`
  - `next_rate`
  - `yesterday_income`
  - `today_income`
  - `withdrawable`
  - `total_income`
  - `users_count`
  - `withdraw_methods`
- Known errors:
  - plain signed request without login -> `JWTVerificationError`

## `POST /api/v1/wallets/verify_email`

- Method: `POST`
- Path: `/api/v1/wallets/verify_email`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `code`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: code`

## `POST /api/v1/wallets/send_verification_email`

- Method: `POST`
- Path: `/api/v1/wallets/send_verification_email`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - empty authenticated request returned `success:1`
- Known errors: none observed

## `POST /api/v1/wallets/bind_withdraw_account`

- Method: `POST`
- Path: `/api/v1/wallets/bind_withdraw_account`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `withdraw_type`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: withdraw_type`

## `GET /api/v1/wallets/binded_withdraw_accounts`

- Method: `GET`
- Path: `/api/v1/wallets/binded_withdraw_accounts`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - `accounts`
- Known errors: none observed

## `DELETE /api/v1/wallets/unbind_withdraw_account/%s`

- Method: `DELETE`
- Path: `/api/v1/wallets/unbind_withdraw_account/%s`
- Auth state: Authenticated route shape confirmed
- Required params:
  - path param `%s` = account id
- Observed response fields: none observed
- Known errors:
  - sample id `1` -> `ResourceNotFound`

## `GET /api/v1/wallets/withdraw_logs`

- Method: `GET`
- Path: `/api/v1/wallets/withdraw_logs`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - `logs`
  - `current_page`
- Known errors: none observed

## `GET /api/v1/wallets/rebate_logs`

- Method: `GET`
- Path: `/api/v1/wallets/rebate_logs`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - `logs`
  - `current_page`
- Known errors: none observed

## `GET /api/v1/wallets/usdt_chain_types`

- Method: `GET`
- Path: `/api/v1/wallets/usdt_chain_types`
- Auth state: Auth required
- Required params: none observed
- Observed response fields:
  - `data.chain_types`
  - observed value: `["TRC20"]`
- Known errors:
  - plain signed request without login -> `JWTVerificationError`

## `GET /api/v1/wallets/sfpay_banks`

- Method: `GET`
- Path: `/api/v1/wallets/sfpay_banks`
- Auth state: Authenticated route verified
- Required params: none observed
- Observed response fields:
  - `banks`
- Known errors: none observed

## `POST /api/v2/wallets/withdraw`

- Method: `POST`
- Path: `/api/v2/wallets/withdraw`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `withdraw_account_id`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: withdraw_account_id`

## `GET /api/v4/plans`

- Method: `GET`
- Path: `/api/v4/plans`
- Auth state: Endpoint section says authenticated; bulk sweep also recorded live verification for the route
- Required params: none observed
- Observed response fields:
  - bulk sweep notes say response shape is close to `/api/v3/plans`, including `plan_message`, `unavailable_plan_message`, `intros`, `plans`, `platforms`
- Known errors: none recorded in the split source text for the authenticated call

## `POST /api/v2/plans/payment_order`

- Method: `POST`
- Path: `/api/v2/plans/payment_order`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `plan_id`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: plan_id`

## `POST /api/v3/plans/payment_order`

- Method: `POST`
- Path: `/api/v3/plans/payment_order`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `plan_id`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: plan_id`

## `GET /api/v1/movies/top`

- Method: `GET`
- Path: `/api/v1/movies/top`
- Auth state: Auth required (returns `JWTVerificationError` without `authorization`)
- Query params:
  - `page` (optional, default `1`) — page number for pagination
  - `period` (optional) — candidate values: `daily`, `weekly`, `monthly`
  - `type` (optional) — candidate values: `0`, `1`
  - APK static evidence for the dedicated `TOP250` page also recovered:
    - `start_rank`
    - `type_value`
    - `ignore_watched`
    - `limit`
  - Note: on the current host (2026-04-26), all param combinations returned the same pre-generated ranking data; the params may be functional on hosts with more frequent ranking refresh cycles
- Observed response fields:
  - `generated_at` (string) — date the ranking was generated (e.g. `"2026-04-21"`)
  - `movies[]` — movie list with a `ranking` field (1–10 observed)
  - `current_page` (number)
- Observed movie fields:
  - `ranking` (number) — rank position (1-based)
  - `id` (string), `number` (string), `title` (string), `origin_title` (string)
  - `thumb_url` (string), `cover_url` (string)
  - `duration` (number), `magnets_count` (number)
  - `can_play` (boolean), `play_subtitle` (number 0/1)
  - `has_preview_video` (boolean), `has_cnsub` (boolean), `has_preview_images` (boolean)
  - `release_date` (string), `new_magnets` (boolean)
  - `preview_images[]` — array of `{ large_url, thumb_url }`
- Current frontend wiring:
  - top-navbar `排行榜 -> TOP250` currently remains the only ranking-page branch that still uses this route
  - the public `有码 / 无码 / 欧美 / FC2` ranking tabs have been moved to `GET /api/v1/rankings`
  - current web client now exposes APK-confirmed `start_rank=1|51|101|151|201`
  - current web client now exposes APK-confirmed `ignore_watched=true|false`
  - current web client still passes `period=daily|weekly|monthly` here while `TOP250` remains only partially live-verified
- APK static evidence for `TOP250`:
  - source page: [top250_page.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/top250_page.dart)
  - source presenter: [top250_presenter.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/presenter/top250_presenter.dart)
  - rankings entry presenters also call the same route for their `TOP250` branch:
    - [rank_presenter.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/rank/presenter/rank_presenter.dart)
    - [rank_sub_presenter.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/rank/presenter/rank_sub_presenter.dart)
  - recovered query builder shape for `getTop250Movies`:
    - `GET /api/v1/movies/top?start_rank={...}&type={...}&type_value={...}&ignore_watched={true|false}&page={...}&limit=50`
  - recovered dedicated `TOP250` UI strings:
    - `Can play`
    - `start ranking:`
    - `Unmarked「watched」`
    - `View only videos that can be played online`
    - `View only videos that have not been marked 「watched」`
  - interpretation:
    - `TOP250` is a dedicated page model with its own filters, not just another `period + type` variant of the public ranking tabs
  - currently wired APK-safe request shape in the web client:
    - `GET /api/v1/movies/top?start_rank=1&type=all&type_value=&ignore_watched=false&page=1&limit=50`
    - `GET /api/v1/movies/top?start_rank=51&type=all&type_value=&ignore_watched=true&page=1&limit=50`
- Known errors:
  - plain signed request without login -> `JWTVerificationError: 请登录账号`
- Verified live: 2026-04-26 with authenticated session

## `GET /api/v1/movies/may_also_like`

- Method: `GET`
- Path: `/api/v1/movies/may_also_like`
- Auth state: Current classification in source notes is authenticated
- Required params:
  - at least one context parameter; live example used `movie_id=bKOeYA`
- Observed response fields: none observed from successful authenticated call
- Known errors:
  - `movie_id=bKOeYA` without auth -> `JWTVerificationError`

## `POST /api/v1/actors/%s/collect_actions`

- Method: `POST`
- Path: `/api/v1/actors/%s/collect_actions`
- Auth state: Authenticated route shape confirmed
- Required params:
  - multipart form field `name`
  - live verified values: `collect`, `uncollect`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: name`

## `POST /api/v1/directors/%s/collect_actions`

- Method: `POST`
- Path: `/api/v1/directors/%s/collect_actions`
- Auth state: Authenticated route shape confirmed
- Required params:
  - multipart form field `name`
  - live verified values: `collect`, `uncollect`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: name`

## `POST /api/v1/makers/%s/collect_actions`

- Method: `POST`
- Path: `/api/v1/makers/%s/collect_actions`
- Auth state: Authenticated route shape confirmed
- Required params:
  - multipart form field `name`
  - live verified values: `collect`, `uncollect`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: name`

## `POST /api/v1/series/%s/collect_actions`

- Method: `POST`
- Path: `/api/v1/series/%s/collect_actions`
- Auth state: Authenticated route shape confirmed
- Required params:
  - multipart form field `name`
  - live verified values: `collect`, `uncollect`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: name`

## `GET /api/v1/lists`

- Method: `GET`
- Path: `/api/v1/lists`
- Auth state: Auth required
- Required params: none observed
- Observed response fields: none observed from successful call; authenticated sample returned `HTTP 500`
- Known errors:
  - without login -> `JWTVerificationError`
  - authenticated sample request -> `HTTP 500`

## `GET /api/v1/lists/simple`

- Method: `GET`
- Path: `/api/v1/lists/simple`
- Auth state: Auth required
- Required params: none observed
- Observed response fields:
  - `lists`
  - list item fields: `id`, `name`, `privacy`, `is_default`, `movies_count`, `has_movie`
- Current frontend wiring:
  - movie detail page uses this route when the user taps `加入清单`
  - the current web client uses `has_movie` to decide whether each list button should show add or remove state
  - navbar user menu `影片的 -> 我的清單` now reuses this route to render the logged-in user's list index page
- Known errors:
  - without login -> `JWTVerificationError`

## `POST /api/v1/lists/%s/collect_actions`

- Method: `POST`
- Path: `/api/v1/lists/%s/collect_actions`
- Auth state: Authenticated route shape confirmed
- Required params:
  - multipart form field `name`
  - live verified values: `collect`, `uncollect`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: name`

## `POST /api/v1/lists/%s/movie_actions`

- Method: `POST`
- Path: `/api/v1/lists/%s/movie_actions`
- Auth state: Authenticated route shape confirmed
- Required params:
  - multipart form field `movie_id`
  - multipart form field `name`
  - live verified values for `name`: `add`, `remove`
- Observed response fields: none observed
- Current frontend wiring:
  - movie detail page posts `movie_id` plus `name=add|remove`
  - the current web client updates the in-panel list state locally after success
- Known errors:
  - empty request -> `ParameterInvalid: movie_id`
  - `movie_id` only -> `ParameterInvalid: name`

## `POST /api/v1/movies/%s/reviews`

- Method: `POST`
- Path: `/api/v1/movies/%s/reviews`
- Auth state: APK-backed authenticated mutation; current web client uses it for movie-detail `想看 / 看过`
- Required params observed from recovered APK request body:
  - path param `%s` = movie id
  - multipart form field `status`
  - multipart form field `score`
  - multipart form field `content`
- APK-backed candidate values:
  - `status=want_watch`
  - `status=watched`
- Current frontend wiring:
  - movie detail page posts `status` plus placeholder `score=0` and empty `content`
  - after success, the web client refreshes `/api/v4/movies/%s` and `/api/v1/users`
- Known gaps:
  - this mutation shape is supported by recovered APK evidence, but still needs separate live verification in docs

## `GET /api/v2/users/review_movies`

- Method: `GET`
- Path: `/api/v2/users/review_movies`
- Auth state: Auth required
- Verified live: `2026-04-29` with authenticated session on host `jdforrepam.com`
- Verified query params:
  - `status`
  - `type`
  - `sort_by`
  - `order_by`
  - `page`
  - `limit`
- Verified sample requests:
  - `GET /api/v2/users/review_movies?status=want_watch&type=0&sort_by=create&order_by=desc&page=1&limit=48`
  - `GET /api/v2/users/review_movies?status=watched&type=0&sort_by=create&order_by=desc&page=1&limit=48`
- Observed response fields:
  - `movies`
  - `current_page`
- Observed movie fields from both `want_watch` and `watched` samples:
  - `id`
  - `number`
  - `title`
  - `origin_title`
  - `cover_url`
  - `thumb_url`
  - `has_cnsub`
  - `duration`
  - `magnets_count`
  - `has_preview_video`
  - `has_preview_images`
  - `can_play`
  - `play_subtitle`
  - `release_date`
  - `new_magnets`
  - `preview_images`
- Observed counts with the current test account:
  - `status=want_watch` -> `22` movies on `current_page=1`
  - `status=watched` -> `7` movies on `current_page=1`
- Current frontend wiring:
  - navbar user menu `影片的 -> 想看`
  - navbar user menu `影片的 -> 看過`
  - current web client exposes these sort mappings in URL param `sort`:
    - `create`
    - `release-desc`
    - `release-asc`
    - `score`
    - `hit`
    - `want-watch-count`
    - `watched-count`

## `GET /api/v1/users/collected_actors`

- Method: `GET`
- Path: `/api/v1/users/collected_actors`
- Auth state: Auth required
- Verified live: `2026-04-29` with authenticated session on host `jdforrepam.com`
- Verified query params:
  - `page`
  - `limit=60`
- Verified sample request:
  - `GET /api/v1/users/collected_actors?page=1&limit=60`
- Observed response fields:
  - `actors`
  - `current_page`
- Observed actor fields:
  - `id`
  - `type`
  - `avatar_url`
  - `name`
  - `name_zht`
  - `other_name`
  - `uncensored`
  - `gender`
  - `videos_count`
- Observed current-account result:
  - `current_page=1`
  - `actors.length=50`
  - first sample had `avatar_url=null`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 演員`

## `GET /api/v1/users/collected_series`

- Method: `GET`
- Path: `/api/v1/users/collected_series`
- Auth state: Auth required
- Verified live: `2026-04-29` with authenticated session on host `jdforrepam.com`
- Verified query params:
  - `page`
  - `limit=48`
- Verified sample request:
  - `GET /api/v1/users/collected_series?page=1&limit=48`
- Observed response fields:
  - `series`
  - `current_page`
- Observed series item fields:
  - `id`
  - `type`
  - `name`
  - `videos_count`
- Observed current-account result:
  - `current_page=1`
  - `series.length=7`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 系列`

## `GET /api/v1/users/collected_makers`

- Method: `GET`
- Path: `/api/v1/users/collected_makers`
- Auth state: Auth required
- Verified live: `2026-04-29` with authenticated session on host `jdforrepam.com`
- Verified query params:
  - `page`
  - `limit=48`
- Verified sample request:
  - `GET /api/v1/users/collected_makers?page=1&limit=48`
- Observed response fields:
  - `makers`
  - `current_page`
- Observed maker item fields:
  - `id`
  - `type`
  - `name`
  - `videos_count`
- Observed current-account result:
  - `current_page=1`
  - `makers.length=11`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 片商`

## `GET /api/v1/users/collected_directors`

- Method: `GET`
- Path: `/api/v1/users/collected_directors`
- Auth state: Auth required
- Verified live: `2026-04-29` with authenticated session on host `jdforrepam.com`
- Verified query params:
  - `page`
  - `limit=48`
- Verified sample request:
  - `GET /api/v1/users/collected_directors?page=1&limit=48`
- Observed response fields:
  - `directors`
  - `current_page`
- Observed current-account result:
  - `current_page=1`
  - `directors.length=0`
  - route itself still returned `HTTP 200` and `success=1`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 導演`

## `GET /api/v1/users/collected_codes`

- Method: `GET`
- Path: `/api/v1/users/collected_codes`
- Auth state: Auth required
- Verified live: `2026-04-29` with authenticated session on host `jdforrepam.com`
- Verified query params:
  - `page`
  - `limit=48`
- Verified sample request:
  - `GET /api/v1/users/collected_codes?page=1&limit=48`
- Observed response fields:
  - `codes`
  - `current_page`
- Observed code item fields:
  - `id`
  - `name`
  - `videos_count`
  - `type`
- Observed current-account result:
  - `current_page=1`
  - `codes.length=3`
- Current frontend wiring:
  - navbar user menu `收藏的 -> 番號`

## `POST /api/v1/codes/%s/collect_actions`

- Method: `POST`
- Path: `/api/v1/codes/%s/collect_actions`
- Auth state: Authenticated route shape confirmed
- Required params:
  - multipart form field `name`
  - live verified values: `collect`, `uncollect`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: name`

## `POST /api/v1/following_tags/batch_destroy`

- Method: `POST`
- Path: `/api/v1/following_tags/batch_destroy`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `ids`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: ids`

## `POST /api/v1/following_tags/batch_push`

- Method: `POST`
- Path: `/api/v1/following_tags/batch_push`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `tags`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: tags`

## `POST /api/v1/logs/movie_played`

- Method: `POST`
- Path: `/api/v1/logs/movie_played`
- Auth state: Authenticated route shape confirmed
- Required params:
  - `movie_id`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: movie_id`
