# `filter_by` Cheatsheet

This file is a compact developer reference for the category feed route:

- `GET /api/v1/movies/tags`

Use it together with:

- [public.md](/F:/codx/javdbweb/docs/api/public.md)
- [pending.md](/F:/codx/javdbweb/docs/api/pending.md)

## Base Shape

Current best live-verified grammar:

```text
{type}:t:{main}:{extra}:{year}:{duration}:{month}
```

## Slot Meaning

| Slot | Name | Meaning | Notes |
| --- | --- | --- | --- |
| 1 | `{type}` | top-row category | `0..4` |
| 2 | `t` | tag-filter mode | fixed literal |
| 3 | `{main}` | primary filter chip | `p / m / c / s / i / v` |
| 4 | `{extra}` | lower-sheet tag ids | comma-joined raw ids, cross-group mixing works |
| 5 | `{year}` | year id | example `2025` |
| 6 | `{duration}` | duration bucket | example `lt-45` |
| 7 | `{month}` | month id | example `1` |

## Type Mapping

| Type | Label |
| --- | --- |
| `0` | `Censored` |
| `1` | `Uncensored` |
| `2` | `Western` |
| `3` | `FC2` |
| `4` | `Carton/Anime` |

## Main Values

| Value | Meaning |
| --- | --- |
| `p` | playable |
| `m` | downloadable |
| `c` | subtitle |
| `s` | single title |
| `i` | has preview images |
| `v` | has preview video |

## Safe Templates

Default first page:

```text
{type}:t:p::::
```

One extra chip:

```text
{type}:t:p:{tagId}:::
```

Cross-group extra chips:

```text
{type}:t:p:{tagIdA},{tagIdB}:::
```

Extra plus year:

```text
{type}:t:p:{tagId}:{year}::
```

Duration plus month:

```text
{type}:t:p:::{duration}:{month}
```

## Verified Behavior

### `extra`

- Accepts raw tag ids without group prefixes
- Supports cross-group mixing
- Order-insensitive in tested pairs
- Behaves like an intersection in tested pairs

Examples:

- `0:t:p:23,158:::`
- `0:t:p:17,18:::`
- `0:t:p:21,347:::`
- `1:t:p:117,96:::`
- `4:t:p:98,88:::`

### `year`

- Single value works
- Comma syntax is accepted
- Current live behavior suggests only the first value is honored

Examples:

- `0:t:p::2025::`
- `0:t:p::2025,2026::` -> behaves like `2025`
- `0:t:p::2026,2025::` -> behaves like `2026`

### `month`

- Single value works
- Comma syntax is accepted
- Current live behavior suggests only the first value is honored

Examples:

- `0:t:p::::1`
- `0:t:p::::1,12` -> behaves like `1`
- `0:t:p::::12,1` -> behaves like `12`

### `duration`

- Single values work
- Multi-value behavior is not stable enough to treat as fully decoded

Examples:

- `0:t:p:::lt-45:`
- `0:t:p:::45-90:`
- `0:t:p:::90-120:`
- `0:t:p:::lt-45,45-90:` -> mixed result, union-like in this ordering
- `0:t:p:::45-90,lt-45:` -> default-looking feed in current round

## Representative Group Map

### `type=0` / `Censored`

| `category_id` | Representative ids | Example |
| --- | --- | --- |
| `subject` | `23`, `51`, `52` | `0:t:p:23:::` |
| `role` | `158`, `67`, `115` | `0:t:p:158:::` |
| `cloth` | `58`, `84`, `106` | `0:t:p:58:::` |
| `body` | `17`, `65`, `91` | `0:t:p:17:::` |
| `behavior` | `18`, `24`, `72` | `0:t:p:18:::` |
| `play_method` | `21`, `60`, `66` | `0:t:p:21:::` |
| `category` | `347`, `345`, `28` | `0:t:p:347:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `0:t:p:::lt-45:` |

Representative combos:

- `0:t:p:21,347:::`
- `0:t:p:21,158:::`
- `0:t:p:21,17:::`

### `type=1` / `Uncensored`

| `category_id` | Representative ids | Example |
| --- | --- | --- |
| `subject` | `55`, `117`, `119` | `1:t:p:55:::` |
| `role` | `61`, `70`, `88` | `1:t:p:61:::` |
| `cloth` | `96`, `97`, `144` | `1:t:p:96:::` |
| `other` | `117`, `121`, `124` | `1:t:p:117:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `1:t:p:::lt-45:` |

Representative combos:

- `1:t:p:117,96:::`
- `1:t:p:117,61:::`
- `1:t:p:117,55:::`

### `type=2` / `Western`

| `category_id` | Representative ids | Example |
| --- | --- | --- |
| `subject` | `40`, `51`, `90` | `2:t:p:40:::` |
| `body` | `13`, `37`, `98` | `2:t:p:13:::` |
| `behavior` | `17`, `18`, `41` | `2:t:p:17:::` |
| `cloth` | `86`, `99`, `134` | `2:t:p:86:::` |
| `place` | `66`, `105`, `281` | `2:t:p:66:::` |
| `role` | `88`, `127`, `179` | `2:t:p:88:::` |
| `other` | `19`, `161`, `193` | `2:t:p:19:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `2:t:p:::45-90:` |

Known note:

- `type=2` dictionary is mapped, but cross-group combo verification is still lighter than `type=0/1/4`

### `type=3` / `FC2`

| `category_id` | Representative ids | Example |
| --- | --- | --- |
| `tag` | `42`, `47`, `49` | `3:t:p:42:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `3:t:p:::45-90:` |

Representative combos:

- `3:t:p:42,47:::`
- `3:t:p:42:2025::`

### `type=4` / `Carton/Anime`

| `category_id` | Representative ids | Example |
| --- | --- | --- |
| `subject` | `1`, `6`, `18` | `4:t:p:1:::` |
| `role` | `37`, `47`, `61` | `4:t:p:37:::` |
| `behavior` | `88`, `91`, `92` | `4:t:p:88:::` |
| `body` | `98`, `101`, `102` | `4:t:p:98:::` |
| `cloth` | `107`, `109`, `111` | `4:t:p:107:::` |
| `other` | `121`, `122` | `4:t:p:121:::` |
| `duration` | `lt-45`, `45-90`, `90-120`, `gt-120` | `4:t:p:::lt-45:` |

Representative combos:

- `4:t:p:98,88:::`
- `4:t:p:121,107:::`
- `4:t:p:1,37:::`

## Implementation Guidance

- Prefer one `year` and one `month`
- Prefer one `duration` until multi-value duration is decoded further
- Treat empty result sets as valid no-match outcomes
- Keep `sort_by=release`, `order_by=desc`, `page=1`, `limit=48` as the safest current defaults

## Canonical Query Example

```text
GET /api/v1/movies/tags?filter_by=1:t:p:117,96:::&sort_by=release&order_by=desc&page=1&limit=48
```

## Sort Buttons to API Params

Recovered category-page sorting maps the visible sort buttons onto `sort_by` and sometimes `order_by`.

| UI label | API params |
| --- | --- |
| `Sort by update` | `sort_by=update` |
| `Sort by release DESC` | `sort_by=release&order_by=desc` |
| `Sort by release ASC` | `sort_by=release&order_by=asc` |
| `Sort by score` | `sort_by=score` |
| `Sort by hit` | `sort_by=hit` |
| `Sort by want watch count` | `sort_by=want_watch_count` |
| `Sort by watched count` | `sort_by=watched_count` |

Notes:

- default category feed sort is `sort_by=release&order_by=desc`
- only the `release` sort exposes explicit `asc/desc` variants in the recovered category page
- the other recovered sort labels currently map to a single `sort_by` value, with no independently recovered `order_by` toggle in the same menu
