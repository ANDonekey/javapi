# Error and Action Summary

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

Only error and action values explicitly present in the source notes are included here.

## `ParameterInvalid`

- Meaning preserved from source:
  - the route exists, but one or more required parameters are missing or empty
- Common response form:

```json
{
  "success": 0,
  "action": "ParameterInvalid",
  "message": "鍙冩暩涓嶈兘鐖茬┖: ...",
  "data": null
}
```

- Observed examples from the source notes:
  - missing `jdsignature` on `/api/v1/startup`
  - missing `category` on `/api/v1/helps`
  - missing `q` on `/api/v2/search`
  - missing `username` or `password` on `/api/v1/sessions`
  - missing `period` on `/api/v1/reviews/hotly`
  - missing `type` on `/api/v1/actors`, `/api/v1/series`, `/api/v1/directors`, `/api/v1/makers`, `/api/v1/rankings`
  - missing `movie_id`, `name`, `ids`, `tags`, `email`, `code`, `withdraw_type`, `withdraw_account_id`, `device_uuid` on various mutation routes

## `InvalidSignature`

- Meaning preserved from source:
  - the route exists and the current host validates `jdsignature`
- Observed examples from the source notes:
  - invalid `jdsignature` on `/api/v1/startup`

## `JWTVerificationError`

- Meaning preserved from source:
  - the route exists and likely needs authenticated user context
- Observed examples from the source notes:
  - `/api/v1/movies/%s/reviews/%s/report`
  - `/api/v1/movies/%s/reviews/%s`
  - `/api/v1/movies/%s/play`
  - `/api/v1/movies/%s/resume_play`
  - `/api/v1/users`
  - `/api/v1/users/recent_viewed`
  - `/api/v1/wallets`
  - `/api/v1/wallets/usdt_chain_types`
  - `/api/v1/movies/top`
  - `/api/v1/lists`
  - `/api/v1/lists/simple`
  - `/api/v1/movies/may_also_like`
  - `/api/v1/rankings` for the observed `type=0&period=552` branch

## `PermissionDeniedToPayment`

- Meaning preserved from source context:
  - authentication may already have succeeded, but playback is blocked by a payment or VIP business-permission gate
- Observed examples from the source notes:
  - `GET /api/v1/movies/gyQbKZ/play?source_id=4` returned `action: "PermissionDeniedToPayment"`
  - `GET /api/v1/movies/gyQbKZ/resume_play?source_id=4&episode=1&resolution=720p&platform=android` returned plain-text body `ERROR:PermissionDeniedToPayment`

## Related Non-action Failure Shape

### `HTTP 404`

- Meaning preserved from source:
  - method may be wrong, route may be unavailable on that host, or the real endpoint shape may differ
- Observed examples from the source notes:
  - `GET /api/v1/movies/%s/reviews/%s/like`
  - `GET /api/v1/movies/%s/reviews/%s/report`
  - `GET /api/v1/movies/%s/reviews/%s`
  - `POST` to `/api/v1/movies/%s/play` and `/resume_play`
  - `/api/v1/rankings/playbackP`
  - `/api/v1/actors/batch_uncollection`
  - several `following_tags` routes
  - `/api/v1/users/unpaid_tickets`
- Resolved 2026-04-26: `GET /api/v2/search_image` was a method mismatch — now verified as `POST` with `multipart/form-data`.
