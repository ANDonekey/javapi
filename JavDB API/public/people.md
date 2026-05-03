# Public Endpoints: People, Studios, Series, Rankings

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

## `GET /api/v1/actors`

- Method: `GET`
- Path: `/api/v1/actors`
- Auth state: Public signed route
- Required params:
  - `type`
- Optional params:
  - `page`
- Observed response fields:
  - `actors`
  - `current_page`
  - actor fields: `id`, `type`, `avatar_url`, `name`, `name_zht`, `other_name`, `uncensored`, `gender`, `videos_count`
- Verified live behavior:
  - `type=0`, `type=1`, `type=2`, and `type=3` return actor lists on the current host
  - `type=4` currently returns an empty `actors[]` list for the tested sample
  - `page` is live and changes the returned slice; `type=0&page=1`, `page=2`, and `page=3` returned different actor sets with `current_page` advancing accordingly
- Verified sample calls:
  - `GET /api/v1/actors?type=0&page=1`
  - `GET /api/v1/actors?type=0&page=2`
  - `GET /api/v1/actors?type=0&page=3`
  - `GET /api/v1/actors?type=1&page=1`
  - `GET /api/v1/actors?type=2&page=1`
  - `GET /api/v1/actors?type=3&page=1`
  - `GET /api/v1/actors?type=4&page=1`
- Practical frontend note:
  - the current frontend now uses this endpoint as an actor-autocomplete fallback pool keyed by category `type`, then filters the returned actor names locally against the user's query
- Current frontend wiring:
  - top-navbar `演员 -> 有码` now lands on `GET /api/v1/actors?type=0&page=1`
  - top-navbar `演员 -> 无码` now lands on `GET /api/v1/actors?type=1&page=1`
- Known errors:
  - missing `type` -> `ParameterInvalid: type`

## `GET /api/v1/actors/%s`

- Method: `GET`
- Path: `/api/v1/actors/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = actor id
- Observed response fields:
  - `share_info`
  - `has_collected`
  - `actor`
  - `filter_tags`
  - `tags`
  - `actor` fields: `id`, `type`, `avatar_url`, `name`, `name_zht`, `other_name`, `birthday`, `age`, `cons`, `blood_type`, `height`, `bust`, `cup`, `waist`, `hips`, `birthplace`, `twitter_id`, `instagram_id`, `videos_count`
  - `filter_tags` fields: `id`, `name`
  - `tags` fields: `id`, `name`, `videos_count`
- APK-backed list-page follow-up:
  - actor detail page does not fall back to `/api/v2/search` for its movie grid
  - after `GET /api/v1/actors/%s`, the APK actor page requests:
    - `GET /api/v1/movies/tags?filter_by={actor.type}:a:{actor.id}{suffix}&sort_by=release&order_by=desc&page={n}`
  - APK-backed basic filter suffixes shared with detail-list pages:
    - `all -> :`
    - `playable -> :p`
    - `magnets -> :m`
    - `subtitle -> :c`
  - current frontend now follows that actor-page transport instead of using actor name text search
- Known errors: none observed

## `GET /api/v1/actors/recommend`

- Method: `GET`
- Path: `/api/v1/actors/recommend`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `new_actors`
  - `monthly_actors`
  - `recommend_actors`
  - actor item fields: `id`, `type`, `avatar_url`, `name`, `other_name`, `gender`, `videos_count`
- Current frontend wiring:
  - top-navbar `演员 -> 推荐` now lands on `/actors?mode=recommend&page=1`
  - the web client renders the three API branches as separate sections:
    - `recommend_actors`
    - `monthly_actors`
    - `new_actors`
- Known errors: none observed

## `GET /api/v1/directors`

- Method: `GET`
- Path: `/api/v1/directors`
- Auth state: Public signed route
- Required params:
  - `type`
- Observed response fields:
  - `directors`
  - `current_page`
- Known errors:
  - missing `type` -> `ParameterInvalid: type`

## `GET /api/v1/directors/%s`

- Method: `GET`
- Path: `/api/v1/directors/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = director id
- Observed response fields:
  - `share_info`
  - `has_collected`
  - `director`
- APK-backed list-page follow-up:
  - detail-page actor/maker/director/series jumps all feed `/api/v1/movies/tags`
  - the current best-supported director movie-list shape is:
    - `GET /api/v1/movies/tags?filter_by={director.type}:d:{director.id}{suffix}&sort_by=release&order_by=desc&page={n}`
- Known errors: none observed

## `GET /api/v1/makers`

- Method: `GET`
- Path: `/api/v1/makers`
- Auth state: Public signed route
- Required params:
  - `type`
- Observed response fields:
  - `makers`
  - `current_page`
- Current frontend wiring:
  - top-navbar `片商` now lands on `/makers?type=censored&page=1`
  - the web client currently exposes APK-aligned category switching through `type=0|1|2|3`
    - `censored -> 0`
    - `uncensored -> 1`
    - `western -> 2`
    - `fc2 -> 3`
- Known errors:
  - missing `type` -> `ParameterInvalid: type`

## `GET /api/v1/makers/%s`

- Method: `GET`
- Path: `/api/v1/makers/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = maker id
- Observed response fields:
  - `share_info`
  - `has_collected`
  - `maker`
- APK-backed list-page follow-up:
  - maker detail pages then request:
    - `GET /api/v1/movies/tags?filter_by={maker.type}:m:{maker.id}{suffix}&sort_by=release&order_by=desc&page={n}`
  - recovered basic filter suffix mapping from `MakerDetailPagePresenter::getDetailMovies`:
    - `all -> :`
    - `playable -> :p`
    - `magnets -> :m`
    - `subtitle -> :c`
  - current frontend now exposes this basic filter row on entity pages and writes it to the `filter` URL param
- Known errors: none observed

## `GET /api/v1/codes/%s`

- Method: `GET`
- Path: `/api/v1/codes/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = code id
- Observed response fields:
  - `share_info`
  - `has_collected`
  - `code`
  - `code` fields: `name`, `videos_count`
- Verified live behavior:
  - `GET /api/v1/codes/GVH` returned a successful payload on the current host
- APK-backed list-page follow-up:
  - code detail pages then request:
    - `GET /api/v1/movies/tags?filter_by=0:c:{code.id}{suffix}&sort_by=release&order_by=desc&page={n}`
  - verified live sample:
    - `GET /api/v1/movies/tags?filter_by=0:c:GVH:&sort_by=release&order_by=desc&page=1&limit=3`
- Practical frontend note:
  - movie detail page `number` links should land on the code entity page using `movie.number_letter` as the route id, not a plain text search for the full number
- Known errors:
  - invalid sample ids such as raw movie ids return `ResourceNotFound`

## `GET /api/v1/publishers/%s`

- Method: `GET`
- Path: `/api/v1/publishers/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = publisher id
- Observed response fields:
  - `share_info`
  - `publisher`
- Known errors:
  - invalid sample id -> `ResourceNotFound`

## `GET /api/v1/series`

- Method: `GET`
- Path: `/api/v1/series`
- Auth state: Public signed route
- Required params:
  - `type`
- Observed response fields:
  - `series`
  - `current_page`
  - series item fields: `id`, `type`, `name`, `videos_count`
- Current frontend wiring:
  - top-navbar `系列` now lands on `/series?type=censored&page=1`
  - the web client currently exposes category switching through `type=0|1|2|3`
    - `censored -> 0`
    - `uncensored -> 1`
    - `western -> 2`
    - `fc2 -> 3`
- Known errors:
  - missing `type` -> `ParameterInvalid: type`

## `GET /api/v1/series/%s`

- Method: `GET`
- Path: `/api/v1/series/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = series id
- Observed response fields:
  - `share_info`
  - `has_collected`
  - `series`
  - series fields: `id`, `type`, `name`, `videos_count`
- APK-backed list-page follow-up:
  - series detail pages then request:
    - `GET /api/v1/movies/tags?filter_by={series.type}:s:{series.id}{suffix}&sort_by=release&order_by=desc&page={n}`
  - current movie-detail frontend now uses `/series/:id?page=1` as the detail-page landing route for the `系列` field, then follows the same list-page transport above
- Known errors: none observed

## `GET /api/v1/series/letters`

- Method: `GET`
- Path: `/api/v1/series/letters`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `letters`
  - letter fields: `id`, `letter`, `type`, `description`, `videos_count`, `views_count`
- Known errors: none observed

## `GET /api/v1/rankings/actors`

- Method: `GET`
- Path: `/api/v1/rankings/actors`
- Auth state: Public signed route
- Required params:
  - `type`
  - `filter_by`
- Observed response fields:
  - `actors`
  - actor ranking item fields: `id`, `name`, `name_zht`, `other_name`, `avatar_url`
- Known errors:
  - missing `type` -> `ParameterInvalid: type`
  - `type=0` without `filter_by` -> `ParameterInvalid: filter_by`
