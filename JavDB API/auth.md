# Auth and Request Conventions

Source: [docs/javdb_api_notes.md](/F:/codx/javdbweb/docs/javdb_api_notes.md)

## Base Host

```text
https://jdforrepam.com
```

## Common Response Envelope

```json
{
  "success": 1,
  "action": null,
  "message": null,
  "data": {}
}
```

## `jdsignature`

Recovered format from the source notes:

```text
jdsignature = "{timestamp}.{middle}.{md5(timestamp + suffix)}"
```

Recovered constants:

```text
middle = lpw6vgqzsp
suffix = 71cf27bb3c0bcdf207b64abecddc970098c7421ee7203b9cdae54478478a199e7d5a6e1a57691123c1a931c057842fb73ba3b3c83bcd69c17ccf174081e3d8aa
```

Notes preserved from source:

- `timestamp` is a Unix timestamp in seconds.
- These values were recovered from Flutter AOT plus `libsecurity.so`.
- These values are tied to this APK sample's original signing context.
- Re-signing the APK breaks the startup flow because the key chain changes.

## Minimal Request Headers

```http
jdsignature: {generated_value}
user-agent: Mozilla/5.0
accept: application/json
```

## Authorization Token Reuse

After a successful login:

- copy `data.token`
- send it back as the raw `authorization` header value
- keep sending `jdsignature` as well

Notes preserved from source:

- No confirmed route has required a `Bearer ` prefix so far.
- No successful route has shown a token-only flow that skips `jdsignature`.

## Failure Modes

- `action: "ParameterInvalid"`: route shape accepted, but one or more required params are missing or empty
- `action: "InvalidSignature"`: route exists and the current host validates `jdsignature`
- `action: "JWTVerificationError"`: route exists and likely needs authenticated user context
- `HTTP 404`: method may be wrong, route may be unavailable on that host, or endpoint shape may differ

## Practical Auth Buckets

- Public signed route: `jdsignature` required, `authorization` not required for basic access
- Authenticated route: route shape accepted, but unauthenticated access returned `JWTVerificationError`
- Unconfirmed route: static-only, `404`, or still ambiguous

## Auth Matrix From Source Notes

| Route | Method | Current auth state | Evidence |
| --- | --- | --- | --- |
| `/api/v1/startup` | `GET` | Public | Live verified |
| `/api/v1/about` | `GET` | Public | Live verified |
| `/api/v1/helps` | `GET` | Public | Live verified |
| `/api/v3/plans` | `GET` | Public | Live verified |
| `/api/v1/movies/latest` | `GET` | Public | Live verified |
| `/api/v4/movies/%s` | `GET` | Public | Live verified |
| `/api/v1/movies/%s/magnets` | `GET` | Public | Live verified |
| `/api/v1/movies/%s/reviews` | `GET` | Public | Live verified |
| `/api/v2/search` | `GET` | Public | Live verified |
| `/api/v1/search_magnet` | `GET` | Public | Live verified |
| `/api/v1/reviews/hotly` | `GET` | Public with required `period` | Live verified |
| `/api/v1/actors` | `GET` | Public with required `type` | Live verified |
| `/api/v1/series` | `GET` | Public with required `type` | Live verified |
| `/api/v1/movies/%s/reviews/%s/like` | `POST` | Public for current test round | Live verified |
| `/api/v1/movies/%s/reviews/%s/report` | `POST` | Auth required | Live verified |
| `/api/v1/movies/%s/reviews/%s` | `DELETE` | Auth required | Live verified |
| `/api/v1/movies/%s/play` | `GET` | Auth required | Live verified |
| `/api/v1/movies/%s/resume_play` | `GET` | Auth required | Live verified |
| `/api/v1/sessions` | `POST` | Public login entrypoint | Live verified |
| `/api/v1/users` | `GET` | Auth required | Live verified |
| `/api/v1/users/recent_viewed` | `GET` | Auth required | Live verified |
| `/api/v1/wallets` | `GET` | Auth required | Live verified |
| `/api/v1/wallets/usdt_chain_types` | `GET` | Auth required | Live verified |
| `/api/v1/movies/top` | `GET` | Auth required | Live verified |
| `/api/v2/search_image` | `POST` | Public signed route | Live verified 2026-04-26 — multipart `image` field, `jdsignature` only |

## Login Payload Metadata Seen In Notes

Observed additional fields for `POST /api/v1/sessions`:

- `device_uuid`
- `device_name`
- `device_model`
- `platform`
- `system_version`
- `app_channel`
- `app_version`
- `app_version_number`

Observed fixed/default values:

- `platform=android`
- `app_version=official`
- `app_version_number=1.9.35`
- `app_channel` defaults to `official` unless `key_app_channel` exists in local storage
