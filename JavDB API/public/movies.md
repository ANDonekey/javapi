# Public Endpoints: Movies and Reviews

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

## `GET /api/v1/movies/latest`

- Method: `GET`
- Path: `/api/v1/movies/latest`
- Auth state: Public signed route
- Required params: none observed for the default first page
- Confirmed source-backed query params:
  - `type`
  - `filter_by`
  - `sort_by`
  - `page`
  - `limit`
- Observed response fields:
  - `movies`
  - `current_page`
  - movie fields: `id`, `number`, `title`, `origin_title`, `thumb_url`, `cover_url`, `duration`, `magnets_count`, `can_play`, `play_subtitle`, `has_preview_video`, `has_cnsub`, `has_preview_images`, `release_date`, `new_magnets`, `preview_images`
- Latest page button mapping from APK static analysis:

| UI button | `type` | `filter_by` | `sort_by` | `page` | `limit` | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `All` | `all` | `all` | `release` | `1` | `48` | latest page default tab |
| `Magnets` | `all` | `magnets` | current page sort, provider default `update` | `1` | `48` | latest page filter tab |
| `Playable` | `all` | `can_play` | current page sort, provider default `update` | `1` | `48` | latest page filter tab |
| `Subtitle` | `all` | `subtitle` | current page sort, provider default `update` | `1` | `48` | latest page filter tab |

- Static behavior notes:
  - source path: [latest_presenter.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/presenter/latest_presenter.dart)
  - when `filter_by=all`, the presenter forces `sort_by=release`
  - when `filter_by` is `magnets`, `can_play`, or `subtitle`, the presenter uses the page's current sort value instead
  - latest-page provider default sort is `update`, see [latest_page_provider.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/provider/latest_page_provider.dart)

- Safe latest-page request shapes:
  - `GET /api/v1/movies/latest?type=all&filter_by=all&sort_by=release&page=1&limit=48`
  - `GET /api/v1/movies/latest?type=all&filter_by=magnets&sort_by=update&page=1&limit=48`
  - `GET /api/v1/movies/latest?type=all&filter_by=can_play&sort_by=update&page=1&limit=48`
  - `GET /api/v1/movies/latest?type=all&filter_by=subtitle&sort_by=update&page=1&limit=48`

- Latest page sort menu mapping from APK static analysis:

| UI label | `sort_by` | `order_by` | Notes |
| --- | --- | --- | --- |
| `Sort by update` | `update` | not separately exposed in recovered latest-page menu | default sort for non-`all` tabs |
| `Sort by release DESC` | `release` | `desc` | explicit menu item |
| `Sort by release ASC` | `release` | `asc` | explicit menu item |
| `Sort by score` | `score` | not separately exposed in recovered latest-page menu | explicit menu item |
| `Sort by hit` | `hit` | not separately exposed in recovered latest-page menu | explicit menu item |
| `Sort by want watch count` | `want_watch_count` | not separately exposed in recovered latest-page menu | explicit menu item |
| `Sort by watched count` | `watched_count` | not separately exposed in recovered latest-page menu | explicit menu item |

- Combined button and sort examples:
  - `All + Sort by release DESC`
    - `GET /api/v1/movies/latest?type=all&filter_by=all&sort_by=release&order_by=desc&page=1&limit=48`
  - `All + Sort by release ASC`
    - `GET /api/v1/movies/latest?type=all&filter_by=all&sort_by=release&order_by=asc&page=1&limit=48`
  - `Magnets + Sort by update`
    - `GET /api/v1/movies/latest?type=all&filter_by=magnets&sort_by=update&page=1&limit=48`
  - `Magnets + Sort by score`
    - `GET /api/v1/movies/latest?type=all&filter_by=magnets&sort_by=score&page=1&limit=48`
  - `Playable + Sort by hit`
    - `GET /api/v1/movies/latest?type=all&filter_by=can_play&sort_by=hit&page=1&limit=48`
  - `Subtitle + Sort by watched count`
    - `GET /api/v1/movies/latest?type=all&filter_by=subtitle&sort_by=watched_count&page=1&limit=48`

- Sorting interpretation notes:
  - latest-page sort source: [latest_movie_page.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/latest_movie_page.dart)
  - `release` is the only recovered latest-page sort with explicit `asc/desc` variants
  - for `update`, `score`, `hit`, `want_watch_count`, and `watched_count`, the recovered menu switches on `sort_by` alone; no separate `order_by` toggle was recovered for those labels

- Home page section `近期磁链更新`:
  - this is the home page card strip, not `/api/v1/rankings/playback`
  - source labels: `LatestLine` and `LatestLineContent` in [splash_page.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/main/splash_page.dart)
  - request builder: [home_presenter.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/presenter/home_presenter.dart) `getLatestMovies`
  - confirmed request shape:
    - `GET /api/v1/movies/latest?type=all&filter_by=magnets&sort_by=update&page=1&limit=18`
  - semantic mapping:
    - `filter_by=magnets` = recent magnet updates feed on the home page
    - `sort_by=update` = ordered by magnet update recency rather than release date
    - `limit=18` = compact home page strip size

- Related home page sibling feed:
  - built by [home_presenter.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/presenter/home_presenter.dart) `getNewArrivalMovies`
  - request shape:
    - `GET /api/v1/movies/latest?type=all&filter_by=can_play&sort_by=update&page=1&limit=18`
  - this is not the `近期磁链更新` section
- Known errors: none observed

## `GET /api/v4/movies/%s`

- Method: `GET`
- Path: `/api/v4/movies/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = movie id
- Observed response fields:
  - `share_info`
  - `show_vip_banner`
  - `movie`
  - nested `movie` fields: `id`, `type`, `number`, `number_letter`, `title`, `origin_title`, `summary`, `thumb_url`, `cover_url`, `has_cnsub`, `has_preview_images`, `duration`, `score`, `reviews_count`, `comments_count`, `want_watch_count`, `watched_count`, `magnets_count`, `has_preview_video`, `can_play`, `play_subtitle`, `play_sources`, `release_date`, `top_rankings`, `maker_id`, `maker_name`, `director_id`, `director_name`, `publisher_id`, `publisher_name`, `series_id`, `series_name`, `tags`, `actors`, `review`, `preview_video_url`, `preview_images`, `relative_movies`, `actor_movies`
- Current frontend wiring:
  - detail page now renders `series_name + series_id` as the `系列` jump target
  - detail page now renders `actor_movies` as a standalone `TA还出演过` section
  - detail page keeps `relative_movies` separate for `相关推荐`, instead of merging the two arrays together
  - detail page now also loads `GET /api/v1/lists/related?movie_id={movie.id}` for the `相关清单` block
- Known errors: none observed

## `GET /api/v1/movies/%s/magnets`

- Method: `GET`
- Path: `/api/v1/movies/%s/magnets`
- Auth state: Public signed route
- Required params:
  - path param `%s` = movie id
- Observed response fields:
  - `magnets`
  - magnet fields: `name`, `hash`, `size`, `cnsub`, `hd`, `files_count`, `created_at`, `pikpak_url`
- Known errors: none observed

## `GET /api/v1/movies/%s/reviews`

- Method: `GET`
- Path: `/api/v1/movies/%s/reviews`
- Auth state: Public signed route
- Required params:
  - path param `%s` = movie id
- Observed response fields:
  - `reviews`
  - review fields: `id`, `user_id`, `username`, `watched_count`, `status`, `status_title`, `score`, `content`, `likes_count`, `liked`, `created_at`
- Known errors: none observed

## `POST /api/v1/movies/%s/reviews/%s/like`

- Method: `POST`
- Path: `/api/v1/movies/%s/reviews/%s/like`
- Auth state: Public for the current test round
- Required params:
  - path param `%s` = movie id
  - path param `%s` = review id
- Observed response fields:
  - response body observed as `success: 1`, `action: null`, `message: null`, `data: null`
- Known errors:
  - `GET` on this route -> `404`

## `GET /api/v1/movies/recommend`

- Method: `GET`
- Path: `/api/v1/movies/recommend`
- Auth state: Public signed route (jdsignature only, no `authorization` required)
- Required params: none observed
- Query params (all observed to have no effect on current host):
  - `type` candidate: `0`, `1`
  - `period` candidate: `monthly`
  - `page` candidate: page number
- Observed response fields:
  - `period` (number): current period id (for example `554`)
  - `movies[]`: simplified movie list for hot/trending display
- Observed movie fields (simplified subset, no `thumb_url` / `preview_images` / `magnets_count`):
  - `id` (string), `number` (string)
  - `title` (string), `origin_title` (string)
  - `duration` (number), `cover_url` (string)
  - `score` (string, for example `"4.3"`), `watched_count` (number)
  - `can_play` (boolean), `release_date` (string)
- Personalization: feed is global, with identical results in the current round with and without auth
- Known errors: none observed

## `GET /api/v1/rankings/playback`

- Method: `GET`
- Path: `/api/v1/rankings/playback`
- Auth state: Public signed route
- Source-backed query params:
  - `filter_by`
  - `period`
- Observed response fields:
  - `movies`
  - `current_page`
  - simplified movie fields observed through shared `SearchNewEntity` handling: `id`, `number`, `title`, `origin_title`, `cover_url`, `duration`, `score`, `watched_count`, `can_play`, `release_date`
- APK source path:
  - request builder: [hot_movie_presenter.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/presenter/hot_movie_presenter.dart)
  - page entry: [hot_movie_page.dart](/F:/codx/javdbapkre/recovered_flutter_source/lib/astarte/home/hot_movie_page.dart)
- App surface:
  - this route backs a dedicated playback-ranking / hot-movie style page
  - it is not the home page `近期磁链更新` card strip
- Confirmed `filter_by` mapping from APK static analysis:

| UI tab | `filter_by` | Notes |
| --- | --- | --- |
| `All` | `all` | top tab recovered as `hotMoviePageTabAll` |
| `High score` | `high_score` | top tab recovered as `hotMoviePageTabHighScore` |

- Confirmed `period` mapping from APK static analysis:

| UI period/tab index | `period` |
| --- | --- |
| first period tab / index `0` | `daily` |
| second period tab / index `1` | `weekly` |
| third period tab / fallback | `monthly` |

- Complete parameter matrix:

| Top tab | Period tab | `filter_by` | `period` | Example |
| --- | --- | --- | --- | --- |
| `All` | `Daily` | `all` | `daily` | `GET /api/v1/rankings/playback?filter_by=all&period=daily` |
| `All` | `Weekly` | `all` | `weekly` | `GET /api/v1/rankings/playback?filter_by=all&period=weekly` |
| `All` | `Monthly` | `all` | `monthly` | `GET /api/v1/rankings/playback?filter_by=all&period=monthly` |
| `High score` | `Daily` | `high_score` | `daily` | `GET /api/v1/rankings/playback?filter_by=high_score&period=daily` |
| `High score` | `Weekly` | `high_score` | `weekly` | `GET /api/v1/rankings/playback?filter_by=high_score&period=weekly` |
| `High score` | `Monthly` | `high_score` | `monthly` | `GET /api/v1/rankings/playback?filter_by=high_score&period=monthly` |

- Safe request shapes:
  - `GET /api/v1/rankings/playback?filter_by=all&period=daily`
  - `GET /api/v1/rankings/playback?filter_by=all&period=weekly`
  - `GET /api/v1/rankings/playback?filter_by=all&period=monthly`
  - `GET /api/v1/rankings/playback?filter_by=high_score&period=daily`
  - `GET /api/v1/rankings/playback?filter_by=high_score&period=weekly`
  - `GET /api/v1/rankings/playback?filter_by=high_score&period=monthly`

- Practical interpretation:
  - this route is separate from `/api/v1/movies/latest`
  - it behaves more like a ranked playback / magnet-update page than a plain latest-arrivals feed
  - no `page`, `limit`, `sort_by`, or `order_by` parameters were recovered for this page's request builder in the current static round
- Current frontend wiring:
  - the top-navbar `排行榜 -> 热播` entry now lands on a dedicated rankings page backed by this route
  - current web client exposes APK-confirmed `period=daily|weekly|monthly` and `filter_by=all|high_score`
  - current web client now keeps the `热播榜` branch separate from the `TOP榜` branch so the playback-only `all|high_score` filter is no longer shown on other ranking modes
- Known errors:
  - none re-verified in the current round

## `GET /api/v1/rankings`

- Method: `GET`
- Path: `/api/v1/rankings`
- Auth state: Public signed route
- Required params:
  - `type`
  - `period`
- Observed response fields:
  - `movies`
  - `current_page`
  - movie fields align with ranking/list cards: `id`, `number`, `title`, `origin_title`, `thumb_url`, `cover_url`, `duration`, `magnets_count`, `can_play`, `play_subtitle`, `has_preview_video`, `has_cnsub`, `has_preview_images`, `release_date`, `new_magnets`, `preview_images`
- Live verified on current host: `2026-04-29`
- Live-verified type mapping used by the current web client:

| APK/Web tab | `type` | Example |
| --- | --- | --- |
| `有码` | `0` | `GET /api/v1/rankings?type=0&period=monthly` |
| `无码` | `1` | `GET /api/v1/rankings?type=1&period=monthly` |
| `欧美` | `2` | `GET /api/v1/rankings?type=2&period=monthly` |
| `FC2` | `3` | `GET /api/v1/rankings?type=3&period=monthly` |

- Live-verified `period` values:
  - `daily`
  - `weekly`
  - `monthly`
- Practical interpretation:
  - this route is the correct public movie-ranking feed for `有码 / 无码 / 欧美 / FC2`
  - current frontend now uses this route for those four ranking tabs instead of the authenticated `/api/v1/movies/top`
  - `TOP250` is still not fully confirmed on this route and remains separate
- Known errors:
  - missing `type` -> `ParameterInvalid: type`
  - missing `period` -> `ParameterInvalid: period`

## `GET /api/v1/movies/recommend_periods`

- Method: `GET`
- Path: `/api/v1/movies/recommend_periods`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `periods`
  - `current_page`
- Known errors: none observed

## `GET /api/v1/reviews/hotly`

- Method: `GET`
- Path: `/api/v1/reviews/hotly`
- Auth state: Public signed route
- Required params:
  - `period`
- Observed response fields:
  - `reviews`
  - review fields: `id`, `user_id`, `username`, `watched_count`, `content`, `score`, `likes_count`, `liked`, `created_at`
  - nested `movie` fields: `id`, `number`, `title`, `origin_title`, `score`, `thumb_url`, `release_date`
- Known errors:
  - missing `period` -> `ParameterInvalid: period`
