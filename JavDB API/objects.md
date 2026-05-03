# Reusable Response Objects

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

Only object groups explicitly named in the source notes are included here.

## `movie`

Observed across:

- `/api/v1/movies/latest`
- `/api/v2/search`
- `/api/v4/movies/%s`

Common list/detail fields observed:

- `id`
- `number`
- `title`
- `origin_title`
- `thumb_url`
- `cover_url`
- `duration`
- `magnets_count`
- `can_play`
- `play_subtitle`
- `has_preview_video`
- `has_cnsub`
- `has_preview_images`
- `release_date`
- `new_magnets`
- `preview_images`

Detail-only or richer fields seen on `/api/v4/movies/%s`:

- `type`
- `number_letter`
- `summary`
- `score`
- `reviews_count`
- `comments_count`
- `want_watch_count`
- `watched_count`
- `play_sources`
- `top_rankings`
- `maker_id`
- `maker_name`
- `director_id`
- `director_name`
- `publisher_id`
- `publisher_name`
- `series_id`
- `series_name`
- `tags`
- `actors`
- `review`
- `preview_video_url`
- `relative_movies`
- `actor_movies`

## `review`

Observed on:

- `/api/v1/movies/%s/reviews`

Fields observed:

- `id`
- `user_id`
- `username`
- `watched_count`
- `status`
- `status_title`
- `score`
- `content`
- `likes_count`
- `liked`
- `created_at`

## `magnet`

Observed on:

- `/api/v1/movies/%s/magnets`
- `/api/v1/search_magnet`

Common fields observed:

- `hash`
- `size`
- `files_count`
- `created_at`

Movie magnet list fields:

- `name`
- `cnsub`
- `hd`
- `pikpak_url`

Magnet search result fields:

- `id`
- `title`

## `tag_group`

Observed on:

- `/api/v2/tags`

Fields observed:

- `category`
- `category_id`
- `tags`

Notes:

- `tags` is an array of `tag_item`
- response shape is grouped, not paginated

## `tag_item`

Observed on:

- `/api/v2/tags`

Fields observed:

- `id`
- `name`
