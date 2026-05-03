# Public Endpoints: Search and Filters

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

## `GET /api/v2/search`

- Method: `GET`
- Path: `/api/v2/search`
- Auth state: Public signed route
- Required params:
  - `q`
- Optional params:
  - `page`
  - `type`
  - `movie_type`
  - `movie_sort_by`
  - `movie_filter_by`
- Observed response fields:
  - `movies`
  - `current_page`
  - movie fields: `id`, `number`, `title`, `origin_title`, `thumb_url`, `cover_url`, `duration`, `magnets_count`, `can_play`, `play_subtitle`, `has_preview_video`, `has_cnsub`, `has_preview_images`, `release_date`, `new_magnets`, `preview_images`
- Verified typed result branches:
  - omitting `type` or using the default movie-search path returns `movies[]`
  - `type=actor` returns `actors[]`
  - `type=series` returns `series[]`
  - `type=maker` returns `makers[]`
  - `type=director` returns `directors[]`
  - `type=code` returns `codes[]`
- Verified typed item fields:
  - `actors[]`: `id`, `type`, `avatar_url`, `name`, `name_zht`, `other_name`, `uncensored`, `gender`, `videos_count`
  - `series[]`: `id`, `type`, `name`, `videos_count`
  - `makers[]`: `id`, `type`, `name`, `videos_count`
  - `directors[]`: `id`, `type`, `name`, `videos_count`
  - `codes[]`: `id`, `type`, `name`, `videos_count`
- Verified canonical follow-up routes from typed result ids:
  - `actor.id -> GET /api/v1/actors/%s`
  - `series.id -> GET /api/v1/series/%s`
  - `maker.id -> GET /api/v1/makers/%s`
  - `director.id -> GET /api/v1/directors/%s`
  - `code.id -> GET /api/v1/codes/%s`
- Typed branch pagination behavior observed in the current round:
  - typed responses did not include `current_page`, `total`, or `total_pages`
  - `page=1`, `page=2`, and `page=3` returned identical result sets for the tested `actor`, `maker`, and `code` queries
  - this is consistent with typed branches currently behaving like unpaginated lookup lists for the tested samples
- Additional typed-query parameter behavior observed in the current round:
  - adding `sort_by` and `order_by` to typed `actor` and `code` requests did not change the returned ordering
  - adding `filter_by` to typed `code` requests did not change the returned data
  - adding `page_size=1` to a typed `actor` request did not reduce the returned item count
  - an unrelated query param such as `foo=bar` was accepted and did not change the returned `maker` data
  - current evidence is therefore consistent with typed branches ignoring these extra params rather than exposing typed-only sort or secondary filter controls
- Verified sample calls:
  - `GET /api/v2/search?q=ipzz&page=1`
  - `GET /api/v2/search?q=ipzz&page=1&type=code`
  - `GET /api/v2/search?q=yua&page=1&type=actor`
  - `GET /api/v2/search?q=madonna&page=1&type=series`
  - `GET /api/v2/search?q=madonna&page=1&type=maker`
  - `GET /api/v2/search?q=kimura&page=1&type=director`
- Verified follow-up calls:
  - `GET /api/v1/actors/Av2e`
  - `GET /api/v1/series/P899`
  - `GET /api/v1/makers/8ZJ9`
  - `GET /api/v1/directors/6xmE`
  - `GET /api/v1/codes/IPZZ`
- Practical frontend note:
  - typed search responses are not all movie lists; consumers must branch on the returned top-level collection key instead of assuming `movies[]`
  - APK request assembly for the default movie-search path uses explicit `type=movie`, plus `movie_type`, `movie_sort_by`, and `movie_filter_by`
  - current frontend now keeps normal search in a dedicated search-mode state:
    - top search tabs switch `searchType` between movie / actor / series / maker / director / code
    - movie-search mode additionally drives `movie_type`, `movie_filter_by`, and `movie_sort_by`
    - typed entity clicks now jump directly to `/actors/:id`, `/series/:id`, `/makers/:id`, `/directors/:id`, or `/codes/:id` instead of falling back to plain text name search
- Known errors:
  - missing `q` -> `ParameterInvalid: q`

## `GET /api/v1/search_magnet`

- Method: `GET`
- Path: `/api/v1/search_magnet`
- Auth state: Public signed route
- Required params:
  - `q`
- Observed response fields:
  - `magnets`
  - `current_page`
  - magnet fields: `id`, `title`, `hash`, `size`, `files_count`, `created_at`
- Known errors: none observed

## `GET /api/v2/tags`

- Method: `GET`
- Path: `/api/v2/tags`
- Auth state: Public signed route
- Required params:
  - `type`
- Observed response fields:
  - `tags`
  - `tags[]` group fields: `category`, `category_id`, `tags`
  - nested tag item fields: `id`, `name`
- Confirmed `type` mapping:
  - `0 = Censored`
  - `1 = Uncensored`
  - `2 = Western`
  - `3 = FC2`
  - `4 = Carton/Anime`
- Stable response shape:
  - response body is a grouped tag dictionary, not pagination
  - top-level `data.tags` is an array of filter groups
  - each group contains a user-facing `category`, a stable programmatic `category_id`, and a nested `tags[]` array
- Common group ids observed across types:
  - `main`
  - `year`
  - `month`
  - `duration`
- Type-specific group ids observed:
  - `type=0`: `subject`, `role`, `cloth`, `body`, `behavior`, `play_method`, `category`
  - `type=1`: `subject`, `role`, `cloth`, `other`
  - `type=2`: `subject`, `body`, `behavior`, `cloth`, `place`, `role`, `other`
  - `type=3`: `tag`
  - `type=4`: `subject`, `role`, `behavior`, `body`, `cloth`, `other`
- Verified sample calls:
  - `GET /api/v2/tags?type=0`
  - `GET /api/v2/tags?type=1`
  - `GET /api/v2/tags?type=2`
  - `GET /api/v2/tags?type=3`
  - `GET /api/v2/tags?type=4`
- Developer shortcut:
  - see [filter-by-cheatsheet.md](/F:/codx/javdbweb/docs/api/filter-by-cheatsheet.md) for representative group ids and starter filter values
- Known errors:
  - missing `type` was not re-tested in this round

## `GET /api/v1/movies/tags`

- Method: `GET`
- Path: `/api/v1/movies/tags`
- Auth state: Public signed route
- Developer shortcut:
  - see [filter-by-cheatsheet.md](/F:/codx/javdbweb/docs/api/filter-by-cheatsheet.md) for a compact `filter_by` reference
- Required params:
  - `filter_by`
  - current verified category-feed requests also included `sort_by=release`, `order_by=desc`, `page=1`, `limit=48`
- Observed response fields:
  - `movies`
  - `has_collected`
  - `current_page`
- Confirmed category request chain:
  - `GET /api/v2/tags?type={0|1|2|3|4}`
  - `GET /api/v1/movies/tags?filter_by=...&sort_by=release&order_by=desc&page=1&limit=48`
- Confirmed default sort from APK static analysis:
  - category page provider initializes `sort_by=release`
  - category page provider initializes `order_by=desc`
- Confirmed top-row category mapping:
  - `0 = Censored`
  - `1 = Uncensored`
  - `2 = Western`
  - `3 = FC2`
  - `4 = Carton/Anime`
- Confirmed `filter_by` syntax, current best reconstruction:
  - `{type}:t:{main}:{extra}:{year}:{duration}:{month}`
- Confirmed `main` values:
  - `p = playable`
  - `m = downloadable`
  - `c = subtitle`
  - `s = single title`
  - `i = has preview images`
  - `v = has preview video`
- Confirmed trailing slot meanings:
  - slot 4 `{extra}` = comma-joined lower-sheet tag ids from non-core groups such as `subject`, `role`, `cloth`, `body`, `behavior`, `play_method`, `category`, `other`, or type-specific groups
  - slot 5 `{year}` = comma-joined year ids such as `2025`
  - slot 6 `{duration}` = comma-joined duration ids such as `lt-45`, `45-90`, `90-120`, `gt-120`
  - slot 7 `{month}` = comma-joined month ids such as `1` or `12`
- Verified usable request shapes:
  - `GET /api/v1/movies/tags?filter_by=0:t:p::::&sort_by=release&order_by=desc&page=1&limit=48`
  - `GET /api/v1/movies/tags?filter_by=1:t:p::::&sort_by=release&order_by=desc&page=1&limit=48`
  - `GET /api/v1/movies/tags?filter_by=2:t:p::::&sort_by=release&order_by=desc&page=1&limit=48`
  - `GET /api/v1/movies/tags?filter_by=3:t:p::::&sort_by=release&order_by=desc&page=1&limit=48`
  - `GET /api/v1/movies/tags?filter_by=4:t:p::::&sort_by=release&order_by=desc&page=1&limit=48`
  - download-oriented variants also work by changing slot 3 from `p` to `m`
- Verified slot-specific examples:
  - `GET /api/v1/movies/tags?filter_by=0:t:p:23:::&sort_by=release&order_by=desc&page=1&limit=3`
    proves slot 4 accepts an ordinary tag id such as `subject=23`
  - `GET /api/v1/movies/tags?filter_by=0:t:p::2025::&sort_by=release&order_by=desc&page=1&limit=3`
    returns `release_date` values anchored in `2025`, proving slot 5 is `year`
  - `GET /api/v1/movies/tags?filter_by=0:t:p:::lt-45:&sort_by=release&order_by=desc&page=1&limit=3`
    returns short-duration titles such as `34`, `36`, and `38` minutes, proving slot 6 is `duration`
  - `GET /api/v1/movies/tags?filter_by=0:t:p::::1&sort_by=release&order_by=desc&page=1&limit=3`
    returns `release_date` values in January, proving slot 7 is `month`
- Verified multi-value behavior:
  - `GET /api/v1/movies/tags?filter_by=0:t:p:23,24:::&sort_by=release&order_by=desc&page=1&limit=8`
    and `...filter_by=0:t:p:24,23:::` return the same narrowed list, which is consistent with slot 4 `extra` behaving like an order-insensitive intersection across multiple tag ids
  - `GET /api/v1/movies/tags?filter_by=0:t:p::2025,2026::&sort_by=release&order_by=desc&page=1&limit=8`
    returns `2025` titles, while `...filter_by=0:t:p::2026,2025::` returns `2026` titles, which is consistent with slot 5 `year` accepting comma syntax but effectively honoring only the first value
  - `GET /api/v1/movies/tags?filter_by=0:t:p::::1,12&sort_by=release&order_by=desc&page=1&limit=8`
    returns January titles, while `...filter_by=0:t:p::::12,1` returns December titles, which is consistent with slot 7 `month` accepting comma syntax but effectively honoring only the first value
  - `GET /api/v1/movies/tags?filter_by=0:t:p:::lt-45,45-90:&sort_by=release&order_by=desc&page=1&limit=8`
    returns a mixed list containing both short titles and 45-90 minute titles, so slot 6 `duration` can behave like a union in at least this ordering
  - `GET /api/v1/movies/tags?filter_by=0:t:p:::45-90,lt-45:&sort_by=release&order_by=desc&page=1&limit=6`
    and `...filter_by=0:t:p:::45-90,90-120:` fell back to what looks like an unfiltered/default feed, so multi-value `duration` behavior is not stable enough to treat as fully decoded
- Verified cross-group `extra` behavior:
  - `GET /api/v1/movies/tags?filter_by=0:t:p:23,158:::&sort_by=release&order_by=desc&page=1&limit=8`
    and `...filter_by=0:t:p:158,23:::` return the same narrowed list, which is consistent with a cross-group intersection between `subject=23` and `role=158`
  - `GET /api/v1/movies/tags?filter_by=0:t:p:23,17:::&sort_by=release&order_by=desc&page=1&limit=8`
    and `...filter_by=0:t:p:17,23:::` return the same narrowed list, which is consistent with a cross-group intersection between `subject=23` and `body=17`
  - `GET /api/v1/movies/tags?filter_by=0:t:p:17,18:::&sort_by=release&order_by=desc&page=1&limit=8`
    and `...filter_by=0:t:p:18,17:::` return the same narrowed list, which is consistent with a cross-group intersection between `body=17` and `behavior=18`
  - `GET /api/v1/movies/tags?filter_by=0:t:p:158,58:::&sort_by=release&order_by=desc&page=1&limit=8`
    and `...filter_by=0:t:p:58,158:::` return the same narrowed list, which is consistent with a cross-group intersection between `role=158` and `cloth=58`
  - `GET /api/v1/movies/tags?filter_by=0:t:p:347,17:::&sort_by=release&order_by=desc&page=1&limit=8`
    and `...filter_by=0:t:p:17,347:::` return the same narrowed list, which is consistent with a cross-group intersection between `category=347` and `body=17`
- Expanded `play_method / category / other / tag` matrix:
  - `GET /api/v1/movies/tags?filter_by=0:t:p:21,347:::&sort_by=release&order_by=desc&page=1&limit=8`
    returns a narrower cross-group list, which is consistent with an intersection between `play_method=21` and `category=347`
  - `GET /api/v1/movies/tags?filter_by=0:t:p:21,158:::&sort_by=release&order_by=desc&page=1&limit=8`
    returns a narrower cross-group list, which is consistent with an intersection between `play_method=21` and `role=158`
  - `GET /api/v1/movies/tags?filter_by=0:t:p:21,17:::&sort_by=release&order_by=desc&page=1&limit=8`
    returns a narrower cross-group list, which is consistent with an intersection between `play_method=21` and `body=17`
  - `GET /api/v1/movies/tags?filter_by=1:t:p:117,96:::&sort_by=release&order_by=desc&page=1&limit=8`
    returns a non-empty narrower list on `type=1`, showing that `other=117` and `cloth=96` can also be mixed inside `extra`
  - `GET /api/v1/movies/tags?filter_by=1:t:p:117,61:::&sort_by=release&order_by=desc&page=1&limit=8`
    and `...filter_by=1:t:p:117,55:::` returned empty lists, which is still consistent with intersection semantics rather than a transport failure
  - `GET /api/v1/movies/tags?filter_by=3:t:p:42,47:::&sort_by=release&order_by=desc&page=1&limit=8`
    returned an empty list on `type=3`, showing that FC2 `tag` ids also participate in `extra` and can intersect down to zero results
- Practical interpretation of slot 4:
  - `extra` accepts raw tag ids without a visible group prefix in the transport string
  - the backend can combine ids from different lower-sheet groups inside the same comma-joined `extra` segment
  - for the tested pairs, the result set is narrower than either single-id feed and is order-insensitive
  - empty result sets from mixed `extra` combinations should be treated as valid intersections that found no matches, not as evidence that the transport grammar is wrong
- Verified cross-slot conjunction examples:
  - `GET /api/v1/movies/tags?filter_by=0:t:p:23:2025::&sort_by=release&order_by=desc&page=1&limit=8`
    returns tag-filtered titles anchored in `2025`, proving `extra + year` can be combined
  - `GET /api/v1/movies/tags?filter_by=0:t:p:::lt-45:1&sort_by=release&order_by=desc&page=1&limit=8`
    returns short January titles, proving `duration + month` are applied conjunctively across different trailing slots
- Representative mapping table for `type=1` (`Uncensored`):

| `category_id` | Representative tag ids | Verified sample |
| --- | --- | --- |
| `main` | `p`, `m`, `c` | `1:t:p::::` |
| `subject` | `55`, `117` | `1:t:p:55:::` |
| `role` | `61`, `70`, `88` | `1:t:p:61:::` |
| `cloth` | `96`, `97`, `144` | `1:t:p:96:::` |
| `other` | `117`, `121`, `124` | `1:t:p:117:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `1:t:p:::lt-45:` |

- Representative `type=1` combinations:
  - `1:t:p:117,96:::` = `other=117` + `cloth=96`, non-empty intersection
  - `1:t:p:117,61:::` = `other=117` + `role=61`, empty intersection in current round
  - `1:t:p:117,55:::` = `other=117` + `subject=55`, empty intersection in current round

- Representative mapping table for `type=3` (`FC2`):

| `category_id` | Representative tag ids | Verified sample |
| --- | --- | --- |
| `main` | `p`, `m`, `c` | `3:t:p::::` |
| `tag` | `42`, `47`, `49` | `3:t:p:42:::`, `3:t:p:47:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `3:t:p:::45-90:` |

- Representative `type=3` combinations:
  - `3:t:p:42,47:::` = `tag=42` + `tag=47`, empty intersection in current round
  - `3:t:p:42:2025::` = `tag=42` + `year=2025`, empty intersection in current round

- Representative mapping table for `type=4` (`Carton/Anime`):

| `category_id` | Representative tag ids | Verified sample |
| --- | --- | --- |
| `main` | `p`, `m`, `c` | `4:t:p::::` |
| `subject` | `1`, `6`, `18` | `4:t:p:1:::` |
| `role` | `37`, `47`, `61` | `4:t:p:37:::` |
| `behavior` | `88`, `91`, `92` | `4:t:p:88:::` |
| `body` | `98`, `101`, `102` | `4:t:p:98:::` |
| `cloth` | `107`, `109`, `111` | `4:t:p:107:::` |
| `other` | `121`, `122` | `4:t:p:121:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `4:t:p:::lt-45:` |

- Representative `type=4` combinations:
  - `4:t:p:98,88:::` = `body=98` + `behavior=88`, non-empty intersection
  - `4:t:p:121,107:::` = `other=121` + `cloth=107`, non-empty intersection
  - `4:t:p:1,37:::` = `subject=1` + `role=37`, empty intersection in current round
  - `4:t:p:1:2025::` = `subject=1` + `year=2025`, empty intersection in current round

- Developer quick reference:

| Type | Label | Primary lower-sheet groups seen | Recommended starter filter | Example combo |
| --- | --- | --- | --- | --- |
| `0` | `Censored` | `subject`, `role`, `cloth`, `body`, `behavior`, `play_method`, `category`, `duration` | `0:t:p:23:::` | `0:t:p:21,347:::` |
| `1` | `Uncensored` | `subject`, `role`, `cloth`, `other`, `duration` | `1:t:p:55:::` | `1:t:p:117,96:::` |
| `2` | `Western` | `subject`, `body`, `behavior`, `cloth`, `place`, `role`, `other`, `duration` | `2:t:p:40:::` | `2:t:p:19,86:::` (to verify in future rounds) |
| `3` | `FC2` | `tag`, `duration` | `3:t:p:42:::` | `3:t:p:42,47:::` |
| `4` | `Carton/Anime` | `subject`, `role`, `behavior`, `body`, `cloth`, `other`, `duration` | `4:t:p:1:::` | `4:t:p:98,88:::` |

- Filter builder snippets for frontend use:
  - category first page:
    - ```${type}:t:p::::```
  - category with one extra chip:
    - ```${type}:t:p:${tagId}:::```
  - category with cross-group extra chips:
    - ```${type}:t:p:${tagIdA},${tagIdB}:::```
  - category with extra + year:
    - ```${type}:t:p:${tagId}:${year}::```
  - category with duration + month:
    - ```${type}:t:p:::${duration}:${month}```

- Safe implementation notes:
  - Prefer sending exactly one `year` and one `month`; current live behavior suggests only the first comma-joined value is honored.
  - Treat multi-value `duration` as unstable until further verification; single values such as `lt-45` or `90-120` are safer.
  - `extra` ids can be mixed across groups and are order-insensitive in the tested pairs.
  - Empty movie lists from mixed `extra` queries should be treated as valid no-match results, not malformed requests.
- Sort button to API parameter mapping from APK static analysis:

| UI label | `sort_by` | `order_by` | Confidence |
| --- | --- | --- | --- |
| `Sort by update` | `update` | not separately exposed in recovered category menu | static-confirmed |
| `Sort by release DESC` | `release` | `desc` | static-confirmed |
| `Sort by release ASC` | `release` | `asc` | static-confirmed |
| `Sort by score` | `score` | not separately exposed in recovered category menu | static-confirmed |
| `Sort by hit` | `hit` | not separately exposed in recovered category menu | static-confirmed |
| `Sort by want watch count` | `want_watch_count` | not separately exposed in recovered category menu | static-confirmed |
| `Sort by watched count` | `watched_count` | not separately exposed in recovered category menu | static-confirmed |

- Practical sorting guidance:
  - safest live-verified feed sort remains `sort_by=release&order_by=desc`
  - `release` is the only recovered category-menu sort with explicit `asc/desc` button variants
  - for `update`, `score`, `hit`, `want_watch_count`, and `watched_count`, the recovered UI logic switches on `sort_by` alone; current category-page source did not expose a parallel `order_by` toggle for those labels
- Known errors:
  - empty request -> `ParameterInvalid: filter_by`
  - naive `filter_by=can_play` -> `HTTP 500`
  - bare `filter_by=main:m&page=1&limit=48` -> `HTTP 500`
  - bare `filter_by=main:p&page=1&limit=48` -> `HTTP 500`
  - `filter_by=main:m&sort_by=release_date&order_by=desc&page=1&limit=48` -> `HTTP 200`
  - `filter_by=main:p&sort_by=release_date&order_by=desc&page=1&limit=48` -> `HTTP 200`
- Verification notes:
  - current verification scope is limited to category/home/list-related filter sources
  - detail-page ids such as maker / actor / series / director / publisher are no longer treated as valid `/api/v1/movies/tags` filter candidates
  - the trailing-slot order is now live-confirmed as `extra -> year -> duration -> month`
  - individual slots accept comma-joined values in the APK builder, but multi-value semantics are not fully documented yet

## `POST /api/v2/search_image`

- Method: `POST`
- Path: `/api/v2/search_image`
- Auth state: Public signed route (jdsignature sufficient, authorization not required)
- Content-Type: `multipart/form-data`
- Required params:
  - `image` (file field) 鈥?supports JPG, PNG, GIF, MP4; field name must be `image`
- Observed response fields for actor recognition:
  - `type`: `"actor"` 鈥?indicates recognition mode
  - `actors[]`:
    - `id` (string) 鈥?actor javdb id
    - `type` (number) 鈥?actor type (0 observed)
    - `name` (string) 鈥?actor name in Chinese
    - `avatar_url` (string, nullable) 鈥?actor avatar url
    - `uncensored` (boolean)
    - `percentage` (number) 鈥?match confidence percentage (0鈥?00)
- Known errors:
  - `GET` on this route -> `HTTP 404`
  - missing `image` field or wrong field name -> `ParameterInvalid: image`
  - unrecognizable image -> `Unrecognized: 鏈壘鍒板尮閰嶇殑缁撴灉`
  - rate limited -> `妫€绱㈣繃浜庨绻侊紝璇风◢鍚庡啀璇昤
  - corrupted/unprocessable file -> `鏃犳晥鐨勬枃浠舵牸寮廯
- Verified live: 2026-04-26 with a real JFIF image containing actor content
