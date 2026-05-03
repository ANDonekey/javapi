# External Integrations

## Full Video Resolver

Frontend integration added on `2026-04-29`.

- Base URL:
  - `https://javwebvid.henry99a.workers.dev`
  - configurable in frontend via `VITE_FULL_VIDEO_API_BASE_URL`
- Verified route:
  - `GET /api/video?code=WAAA-637`
- Optional query params:
  - `lang` — defaults to `zh`
- Verified response fields:
  - `code`
  - `results`
  - `sourceSite`
  - `fetchedAt`
- Verified `results[]` fields:
  - `pageUrl`
  - `title`
  - `variant`
  - `label`
  - `poster`
  - `defaultQuality`
  - `qualityOptions`
  - `sources`
- Verified `results[].sources[]` fields:
  - `url`
  - `location`
  - `redirectStatus`
  - `type`
  - `quality`
  - `expires`
  - `expiresAt`
  - `temporary`

Current frontend wiring:

- movie detail page reuses the existing preview-video slot
- adds a `预览视频 / 完整视频` toggle
- when `完整视频` is selected, the web client lazily requests:
  - `GET /api/video?code={movie.number}&lang=zh`
- if the worker returns multiple `results[]`, the web client exposes a second in-place version switcher
- the web client prefers `results[].sources[].location`, then falls back to `results[].sources[].url`
- the current implementation uses the worker-returned `type` to choose direct mp4 playback vs HLS handling

Observed live sample on `2026-04-29`:

- movie id `AqnxRO` resolved to movie number `WAAA-637`
- worker response returned:
  - `results[0].defaultQuality=720`
  - `results[0].qualityOptions=[720]`
  - `results[0].sources[0].type="video/mp4"`
  - `results[0].sources[0].redirectStatus=302`
  - `results[0].sources[0].temporary=true`

Observed multi-version sample on `2026-04-29`:

- `GET /api/video?code=SNOS-200`
- worker response returned `results.length=2`
- verified variants:
  - `variant="reducing_mosaic"` with `label="Reducing"`
  - `variant="original"` with `label=null`
- both entries carried playable mp4 sources

Known caveats:

- `results[].sources[].url` and `results[].sources[].location` are temporary upstream links
- the frontend should not cache them long-term
- some movies may have no upstream full-video result and should fall back to preview mode

## Preview Proxy Worker

Frontend integration extended on `2026-04-30`.

- Base URL:
  - `https://fourhoi-preview-proxy-worker.henry99a.workers.dev`
  - configurable in frontend via `VITE_PREVIEW_PROXY_BASE_URL`
- Verified routes:
  - `GET /preview.mp4?number=SNOS-200`
  - `GET /api/preview/javstore?code=SNOS-200`
  - `GET /api/preview/javstore/image?code=SNOS-200`

Verified `GET /api/preview/javstore` response fields:

- `code`
- `source`
- `previewUrl`
- `proxiedPreviewUrl`
- `matchedUrl`
- `cached`

Current frontend wiring:

- movie detail page keeps using `preview.mp4` for the inline `Preview Clip`
- movie detail page additionally calls:
  - `GET /api/preview/javstore?code={movie.number}`
- if `proxiedPreviewUrl` exists, that image is merged into the existing preview image list
- duplicate preview image URLs are deduplicated in the service layer before rendering

Observed live sample on `2026-04-30`:

- `GET /api/preview/javstore?code=SNOS-200`
- worker response returned:
  - `source="javstore"`
  - `proxiedPreviewUrl="/api/preview/javstore/image?code=SNOS-200"` on the same worker host
- `GET /api/preview/javstore/image?code=SNOS-200` returned `image/jpeg`
