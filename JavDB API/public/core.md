# Public Endpoints: Core and Utility

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

## `GET /api/v1/startup`

- Method: `GET`
- Path: `/api/v1/startup`
- Auth state: Public signed route
- Required params:
  - `last_ad_id`
  - `platform`
  - `app_channel`
  - `app_version`
  - `app_version_number`
- Observed response fields:
  - `splash_ad`
  - `user`
  - `backup_domains_data`
  - `recent_keywords`
  - `recent_magnet_keywords`
  - `settings`
  - `feedback`
  - `logging_enabled`
  - `recognize_actor_enabled`
  - `recognize_movie_enabled`
  - `web_image_prefix`
  - `ypay_payment_enabled`
- Known errors:
  - missing `jdsignature` -> `ParameterInvalid: jdsignature`
  - invalid `jdsignature` -> `InvalidSignature`
  - valid signature but missing query params -> parameter error

## `GET /api/v1/about`

- Method: `GET`
- Path: `/api/v1/about`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `name`
  - `meta`
  - `url`
- Known errors: none observed

## `GET /api/v1/helps`

- Method: `GET`
- Path: `/api/v1/helps`
- Auth state: Public signed route
- Required params:
  - `category`
- Observed response fields:
  - `helps`
  - `current_page`
  - help entry fields: `question`, `answer`
- Known errors:
  - missing `category` -> `ParameterInvalid: category`

## `GET /api/v3/plans`

- Method: `GET`
- Path: `/api/v3/plans`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `plan_message`
  - `unavailable_plan_message`
  - `intros`
  - `plans`
  - `platforms`
  - nested plan fields: `id`, `name`, `price`, `origin_price`, `currency_unit`, `currency_symbol`, `days`
  - nested payment fields: `platforms[].id`, `platforms[].channels[].id`, `platforms[].channels[].methods[].id`, `platforms[].channels[].methods[].price_type`, `platforms[].channels[].methods[].limited_prices`
- Known errors: none observed

## `POST /api/v1/sessions`

- Method: `POST`
- Path: `/api/v1/sessions`
- Auth state: Public login entrypoint
- Required params:
  - `username`
  - `password`
- Observed response fields:
  - `data.user`
  - `data.following_tags`
  - `data.token`
  - observed `data.user` fields: `id`, `username`, `email`, `is_vip`, `vip_expired_at`, `want_watch_count`, `watched_count`, `promote_users_count`, `share_url`, `promotion_code`, `promotion_days`, `checkin_days`, `last_checkin_at`
- Current frontend wiring:
  - navbar login now uses this route directly
  - current web client only exposes username/password login
  - web request body is sent as multipart `FormData`, matching the recovered APK `FormData.fromMap` evidence
- Known errors:
  - `GET /api/v1/sessions` -> `404`
  - empty `POST` body -> `ParameterInvalid: username`
  - `username` present but empty `password` -> `ParameterInvalid: password`

## `GET /api/v1/ads`

- Method: `GET`
- Path: `/api/v1/ads`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `enabled`
  - `ads`
  - observed placement groups include `magnets_top`, `web_magnets_top`, `index_top`
- Known errors: none observed

## `POST /api/v1/ads/splash_log`

- Method: `POST`
- Path: `/api/v1/ads/splash_log`
- Auth state: Public route shape confirmed in the source notes
- Required params:
  - `id`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: id`

## `GET /api/v1/magnet_apps`

- Method: `GET`
- Path: `/api/v1/magnet_apps`
- Auth state: Public signed route
- Required params: none observed
- Observed response fields:
  - `apps`
  - app item fields: `name`, `description`, `recommended`, `links`
- Known errors: none observed

## `POST /api/v2/logs/activated`

- Method: `POST`
- Path: `/api/v2/logs/activated`
- Auth state: Public signed route shape confirmed
- Required params:
  - `device_uuid`
- Observed response fields: none observed
- Known errors:
  - empty request -> `ParameterInvalid: device_uuid`
