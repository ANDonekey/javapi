# Public Endpoints: Lists and Articles

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

## `GET /api/v1/lists/related`

- Method: `GET`
- Path: `/api/v1/lists/related`
- Auth state: Public signed route
- Required params:
  - `movie_id`
- Observed response fields:
  - `lists`
  - `current_page`
- Live verified list item fields:
  - `id`, `name`, `description`, `movies_count`, `views_count`, `collections_count`, `is_default`, `share_info`, `created_at`
- Current frontend wiring:
  - movie detail page now uses this route for the `相关清单` block
  - current web client links each related list item to `/lists/:id?page=1`
- Known errors:
  - missing `movie_id` -> `ParameterInvalid: movie_id`

## `GET /api/v1/lists/%s`

- Method: `GET`
- Path: `/api/v1/lists/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = list id
- Observed response fields:
  - `is_creator`
  - `share_info`
  - `has_collected`
  - `list`
- Live verified `list` fields:
  - `id`, `name`, `description`, `movies_count`, `views_count`, `collections_count`, `is_default`, `share_info`, `created_at`
- APK-backed list-page follow-up:
  - list detail pages then request:
    - `GET /api/v1/movies/tags?filter_by=0:l:{list.id}{suffix}&sort_by=release&page={n}&limit={n}`
  - verified suffix family from APK evidence:
    - `all -> :`
    - `playable -> :p`
    - `magnets -> :m`
    - `subtitle -> :c`
- Current frontend wiring:
  - `/lists/:id?page=1` now resolves through the shared entity-style list page and uses the `0:l:{list_id}{suffix}` movie feed pattern
- Known errors:
  - invalid or unauthorized sample ids can return `NoPermission`

## `GET /api/v1/articles`

- Method: `GET`
- Path: `/api/v1/articles`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `articles`
  - `current_page`
  - article list item fields: `id`, `title`, `cover_url`, `author`, `category`, `released_at`
- Known errors: none observed

## `GET /api/v1/articles/%s`

- Method: `GET`
- Path: `/api/v1/articles/%s`
- Auth state: Public signed route
- Required params:
  - path param `%s` = article id
- Observed response fields:
  - article detail fields: `id`, `title`, `origin_name`, `origin_url`, `cover_url`, `author`, `category`, `image_domain`, `content`, `released_at`, `related_movies`
- Known errors: none observed
