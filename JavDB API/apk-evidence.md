# APK Evidence

## Scope

- This file records APK static analysis evidence only.
- It does not mean any route is live verified.
- Live results belong in `public.md` and `authenticated.md`.
- Unverified routes belong in `pending.md`.
- If a value was not recovered from the current APK scan, it is marked as `Unknown source`.

## Evidence Legend

### Evidence Type

- Route string
- Presenter call
- Service call
- queryParameters
- FormData.fromMap
- Header construction
- Native constant
- UI route reference

### Confidence

- High: route directly called in request code
- Medium: route path and params appear in the same static context, but the full call chain is incomplete
- Low: string-only or UI-only reference

## Signature Evidence

### jdsignature

| Item | Value / Pattern | Source | Confidence | Notes |
| --- | --- | --- | --- | --- |
| Header key | `jdsignature` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Header name recovered from Flutter AOT string pool only |
| Timestamp param injection | `timestamp` inserted into a request param map before URL assembly | `F:\codx\adrev\decoded\jdb_official_v1.9.35_apktool\smali\e3\a.smali:160-174` | Medium | Generic `HttpURLConnection` POST helper; not route-specific |
| MD5 helper | `MD5` via `MessageDigest.getInstance("MD5")` | `F:\codx\adrev\decoded\jdb_official_v1.9.35_apktool\smali\s5\c.smali:8-29` | Medium | Java-side MD5 helper present; direct relation to `jdsignature` not proven in this scan |
| Native MD5 symbols | `MD5Init`, `MD5Update`, `MD5Final`, `MD5Transform` | `F:\codx\adrev\decoded\jdb_official_v1.9.35_apktool\lib\arm64-v8a\libsecurity.so`, `...\lib\armeabi-v7a\libsecurity.so` | Medium | Confirms native MD5 implementation exists |
| Authorization header key | `authorization` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Header key only; no route binding recovered |
| Proxy auth header key | `proxy-authorization` | `libapp.so` arm64 extracted ASCII strings | Low | Likely library/network stack string, not app API specific |
| Middle constant | Unknown source | Unknown source | Low | Current APK scan did not recover the middle token as a plain string |
| Suffix constant | Unknown source | Unknown source | Low | Current APK scan did not recover the suffix token as a plain string |

## Route Evidence Index

### Startup / App Config

## `{METHOD?} /api/v1/startup`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/startup` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `jdsignature` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Header construction | Low | Shared header string |
| `app_channel`, `app_version`, `app_version_number` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Parameter names recovered as standalone strings |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/startup`
- Query params: `app_channel`, `app_version`, `app_version_number`
- Form fields: None recovered
- Headers: `jdsignature`
- Body style: Unknown
- Related UI page: App bootstrap
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm HTTP method.
2. Confirm whether `platform` and version metadata are query params or body fields.
3. Confirm whether the route accepts signed-only access.

## `{METHOD?} /api/v1/debug_logging`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/debug_logging` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | No call-site recovered |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/debug_logging`
- Query params: None recovered
- Form fields: None recovered
- Headers: Unknown
- Body style: Unknown
- Related UI page: Unknown
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm HTTP method.
2. Confirm whether this is still active.
3. Confirm auth/signature requirements.

### Auth / Sessions

## `{METHOD?} /api/v1/sessions`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/sessions` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Login route string present |
| `FormData.fromMap` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | FormData.fromMap | Medium | Flutter Dio multipart/form helper exists in same AOT bundle |
| `username`, `password`, `device_uuid`, `device_name`, `device_model`, `system_version`, `app_channel`, `app_version`, `app_version_number` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate login/request metadata fields |
| `authorization` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Header construction | Low | Header key is present in the app bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/sessions`
- Query params: None recovered
- Form fields: `username`, `password`, `device_uuid`, `device_name`, `device_model`, `system_version`, `app_channel`, `app_version`, `app_version_number`
- Headers: `jdsignature`, `authorization`
- Body style: Form-based / Multipart candidate
- Related UI page: Login flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm whether body is multipart or x-www-form-urlencoded.
2. Confirm exact required device metadata.
3. Confirm whether `authorization` is absent on login.

## `{METHOD?} /api/v1/users/forgot_password`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/users/forgot_password` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `forgotPassword` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Flow symbol only |
| `email`, `username`, `password` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate related fields only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/users/forgot_password`
- Query params: None recovered
- Form fields: `email` or `username`
- Headers: `jdsignature`
- Body style: Unknown
- Related UI page: Forgot password flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm body field names.
2. Confirm request content type.
3. Confirm whether email and username are interchangeable.

## `{METHOD?} /api/v1/users/reset_password`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/users/reset_password` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `resetPassword` and `package:astarte/mine/presenter/reset_password_presenter.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Flow symbol only |
| `password`, `password_confirmation`, `email`, `code` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate related fields only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/users/reset_password`
- Query params: None recovered
- Form fields: `email`, `code`, `password`, `password_confirmation`
- Headers: `jdsignature`
- Body style: Unknown
- Related UI page: Reset password flow
- Related presenter/service: `reset_password_presenter.dart`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm exact body shape.
2. Confirm whether multipart is used.
3. Confirm code validation rules.

## `{METHOD?} /api/v1/users/change_username`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/users/change_username` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `package:astarte/mine/presenter/update_username_presenter.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Presenter symbol only |
| `username`, `current_password`, `new_username` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate related fields only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/users/change_username`
- Query params: None recovered
- Form fields: `username`, `current_password`
- Headers: `authorization`, `jdsignature`
- Body style: Unknown
- Related UI page: Update username page
- Related presenter/service: `update_username_presenter.dart`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm whether `username` or `new_username` is sent.
2. Confirm content type.
3. Confirm auth requirement.

## `{METHOD?} /api/v1/users/change_password`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/users/change_password` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `package:astarte/mine/presenter/update_password_presenter.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Presenter symbol only |
| `old_password`, `password`, `password_confirmation` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate related fields only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/users/change_password`
- Query params: None recovered
- Form fields: `old_password`, `password`, `password_confirmation`
- Headers: `authorization`, `jdsignature`
- Body style: Unknown
- Related UI page: Update password page
- Related presenter/service: `update_password_presenter.dart`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm exact field names.
2. Confirm content type.
3. Confirm whether password confirmation is mandatory.

### Movies

## `{METHOD?} /api/v1/movies/tags`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/movies/tags` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `filter_by`, `movie_filter_by`, `filter_by_tags`, `sort_by`, `order_by` | `libapp.so` arm64 extracted ASCII strings | queryParameters | Medium | Route-related filter keys are present in the same AOT bundle |
| `queryParameters`, `get:queryParameters`, `init:queryParameters`, `queryParametersAll` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Confirms Dio query-based request helpers exist |
| `HomePagePresenter`, `CategoryListPresenter`, `ActorInfoNewPagePresenter`, `MakerDetailPagePresenter`, `MyListsDetailPagePresenter` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Presenter symbols recovered, but direct call chain not reconstructed |
| `package:astarte/category/provider/category_page_provider.dart`, `package:astarte/category/presenter/category_presenter.dart`, `package:astarte/category/models/tags_entity.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Category/tag flow symbols |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/movies/tags`
- Query params: `filter_by`, `filter_by_tags`, `sort_by`, `order_by`
- Form fields: None recovered
- Headers: `jdsignature`
- Body style: Query-based
- Related UI page: Category page, actor info page, maker detail page, list detail page
- Related presenter/service: `HomePagePresenter`, `CategoryListPresenter`, `ActorInfoNewPagePresenter`, `MakerDetailPagePresenter`, `MyListsDetailPagePresenter`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm exact `filter_by` values accepted by backend.
2. Confirm whether `page` and `limit` are sent for all tabs.
3. Confirm whether `filter_by_tags` is actor-only or shared across scenes.

## `{METHOD?} /api/v1/movies/%s/play`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/movies/%s/play` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Playback route recovered from AOT string pool |
| `source_id`, `resolution`, `fromRankings`, `operation` | `libapp.so` arm64 extracted ASCII strings | queryParameters | Medium | Candidate playback query keys present in same AOT bundle |
| `queryParameters` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Query helper exists in the same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/movies/%s/play`
- Query params: `source_id`, `resolution`, `fromRankings`, `operation`
- Form fields: None recovered
- Headers: `authorization`, `jdsignature`
- Body style: Query-based
- Related UI page: Playback flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm final method.
2. Confirm required playback query keys.
3. Confirm auth gate and VIP/payment behavior.

## `{METHOD?} /api/v1/movies/%s/resume_play`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/movies/%s/resume_play` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Resume playback route recovered from AOT string pool |
| `source_id`, `episode`, `resolution` | `libapp.so` arm64 extracted ASCII strings | queryParameters | Medium | Candidate resume-play query keys present in same AOT bundle |
| `queryParameters` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Query helper exists in the same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/movies/%s/resume_play`
- Query params: `source_id`, `episode`, `resolution`
- Form fields: None recovered
- Headers: `authorization`, `jdsignature`
- Body style: Query-based
- Related UI page: Playback resume flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm whether `platform=android` is always sent.
2. Confirm required `source_id`.
3. Confirm failure mode for unpaid sources.

### Search

## `{METHOD?} /api/v2/search`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v2/search` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Search route recovered from AOT string pool |
| `queryParameters` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Query helper exists in the same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v2/search`
- Query params: Unknown
- Form fields: None recovered
- Headers: `jdsignature`
- Body style: Query-based candidate
- Related UI page: Search flow
- Related presenter/service: `search_page_new_provider.dart`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm required query key.
2. Confirm pagination params.
3. Confirm signed-only vs anonymous access.

## `{METHOD?} /api/v1/search_magnet`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/search_magnet` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Magnet search route recovered from AOT string pool |
| `package:astarte/home/search_magnet_page.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Search-magnet UI symbol only |
| `queryParameters` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Query helper exists in the same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/search_magnet`
- Query params: Unknown
- Form fields: None recovered
- Headers: `jdsignature`
- Body style: Query-based candidate
- Related UI page: `search_magnet_page.dart`
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm required search keyword field.
2. Confirm pagination support.
3. Confirm whether route is public signed.

## `{METHOD?} /api/v2/search_image`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v2/search_image` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `SearchImagePage`, `SearchImageResultPage`, `search_image_movie_item`, `getImageSearchResult` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Medium | Multiple image-search flow symbols appear in same AOT bundle |
| `FormData.fromMap`, `FormData` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | FormData.fromMap | Medium | Multipart/form helper is present in same bundle |
| `queryParameters`, `queryParametersAll` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Search-image flow likely mixes query and upload metadata |

Observed static request hints:
- Method: Unknown
- Path: `/api/v2/search_image`
- Query params: Unknown
- Form fields: Unknown
- Headers: `jdsignature`
- Body style: Form-based / Multipart candidate
- Related UI page: `search_image_page.dart`, `search_image_result_page.dart`
- Related presenter/service: `getImageSearchResult`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm whether an image file is required.
2. Confirm final HTTP method.
3. Confirm whether query params are sent alongside multipart body.

### Reviews / Comments

## `{METHOD?} /api/v1/reviews/hotly`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/reviews/hotly` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `queryParameters` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Query helper exists in the same bundle |
| `package:astarte/home/hot_comment_page.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Hot-comment UI symbol only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/reviews/hotly`
- Query params: Unknown
- Form fields: None recovered
- Headers: `jdsignature`
- Body style: Query-based candidate
- Related UI page: `hot_comment_page.dart`
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm required query fields.
2. Confirm ordering/pagination behavior.
3. Confirm signed-only vs anonymous access.

### Users

## `{METHOD?} /api/v1/users/transaction_logs`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/users/transaction_logs` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `queryParameters` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | queryParameters | Medium | Query helper exists in the same bundle |
| `WithdrawRecordPagePresenter` and `rebate_record_page.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Transaction/wallet UI symbols nearby in bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/users/transaction_logs`
- Query params: Unknown
- Form fields: None recovered
- Headers: `authorization`, `jdsignature`
- Body style: Query-based candidate
- Related UI page: Wallet/transaction pages
- Related presenter/service: `WithdrawRecordPagePresenter`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm pagination params.
2. Confirm filter keys.
3. Confirm auth requirement.

## `{METHOD?} /api/v1/users/feedback`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/users/feedback` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `content`, `email`, `contact`, `category` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate related fields only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/users/feedback`
- Query params: None recovered
- Form fields: `content`, `contact`, `category`
- Headers: `authorization`, `jdsignature`
- Body style: Unknown
- Related UI page: Feedback flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm exact required fields.
2. Confirm whether contact defaults to registered email.
3. Confirm content type.

### Wallets

## `{METHOD?} /api/v1/wallets/verify_email`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/wallets/verify_email` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `_verifyEmail@1017285803`, `Verify email` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Medium | Verify-email flow symbols appear in the same bundle |
| `code`, `email` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate related fields only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/wallets/verify_email`
- Query params: None recovered
- Form fields: `code`, `email`
- Headers: `authorization`, `jdsignature`
- Body style: Unknown
- Related UI page: Verify email flow
- Related presenter/service: `_verifyEmail`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm whether `email` is posted or inferred from account.
2. Confirm content type.
3. Confirm auth requirement.

## `{METHOD?} /api/v1/wallets/send_verification_email`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/wallets/send_verification_email` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `email` and verify-email UI strings | `libapp.so` arm64 extracted ASCII strings | UI route reference | Low | Mail verification flow context only |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/wallets/send_verification_email`
- Query params: None recovered
- Form fields: Unknown
- Headers: `authorization`, `jdsignature`
- Body style: Unknown
- Related UI page: Verify email flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm whether body is empty.
2. Confirm rate-limit behavior.
3. Confirm auth requirement.

## `{METHOD?} /api/v1/wallets/bind_withdraw_account`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v1/wallets/bind_withdraw_account` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `package:astarte/mine/presenter/withdraw_bind_presenter.dart`, `withdraw_bind_page.dart`, `withdraw_choose_bank_page.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Medium | Withdraw binding flow symbols appear in the same bundle |
| `withdraw_type`, `withdraw_account_id`, `bank_name`, `code`, `email` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate related fields only |
| `FormData.fromMap` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | FormData.fromMap | Medium | Form helper exists in same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v1/wallets/bind_withdraw_account`
- Query params: None recovered
- Form fields: `withdraw_type`, `code`, plus bank/account-related fields
- Headers: `authorization`, `jdsignature`
- Body style: Form-based / Multipart candidate
- Related UI page: Withdraw bind / choose bank pages
- Related presenter/service: `withdraw_bind_presenter.dart`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm exact field names for bank vs crypto branches.
2. Confirm whether multipart is used.
3. Confirm verification-code requirement.

## `{METHOD?} /api/v2/wallets/withdraw`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v2/wallets/withdraw` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `package:astarte/mine/presenter/withdraw_presenter.dart`, `withdraw_page.dart`, `withdraw_verify_presenter.dart` | `libapp.so` arm64 extracted ASCII strings | UI route reference | Medium | Withdraw flow symbols appear in the same bundle |
| `withdraw_account_id` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate field only |
| `FormData.fromMap` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | FormData.fromMap | Medium | Form helper exists in same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v2/wallets/withdraw`
- Query params: None recovered
- Form fields: `withdraw_account_id`
- Headers: `authorization`, `jdsignature`
- Body style: Form-based / Multipart candidate
- Related UI page: Withdraw page
- Related presenter/service: `withdraw_presenter.dart`

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm required amount/code fields.
2. Confirm content type.
3. Confirm bank/USDT branch differences.

### Plans / Payment

## `{METHOD?} /api/v2/plans/payment_order`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v2/plans/payment_order` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `plan_id`, `platform_id`, `method_id` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate payment-order fields only |
| `FormData.fromMap` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | FormData.fromMap | Medium | Form helper exists in same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v2/plans/payment_order`
- Query params: None recovered
- Form fields: `plan_id`, `platform_id`, `method_id`
- Headers: `authorization`, `jdsignature`
- Body style: Form-based / Multipart candidate
- Related UI page: Payment flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm complete field matrix.
2. Confirm whether channel/price IDs are required.
3. Confirm anonymous vs authenticated behavior.

## `{METHOD?} /api/v3/plans/payment_order`

Status:
- Static inference only

Evidence:
| Evidence | Source file / symbol | Type | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `/api/v3/plans/payment_order` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Route string | Low | Route recovered from AOT string pool |
| `plan_id`, `platform_id`, `method_id` | `libapp.so` arm64 extracted ASCII strings | Route string | Low | Candidate payment-order fields only |
| `FormData.fromMap` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | FormData.fromMap | Medium | Form helper exists in same bundle |

Observed static request hints:
- Method: Unknown
- Path: `/api/v3/plans/payment_order`
- Query params: None recovered
- Form fields: `plan_id`, `platform_id`, `method_id`
- Headers: `authorization`, `jdsignature`
- Body style: Form-based / Multipart candidate
- Related UI page: Payment flow
- Related presenter/service: Unknown

Verification status:
- Not live verified in this file
- Check `public.md` / `authenticated.md` for live status

To verify:
1. Confirm structural differences vs v2.
2. Confirm complete field matrix.
3. Confirm auth requirement.

## Special Section: `/api/v1/movies/tags`

### filter_by Static Evidence

| Candidate | UI Scene | Source | Construction Pattern | Confidence | Notes |
| --- | --- | --- | --- | --- | --- |
| `filter_by` | Generic movie-tag list | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Literal query key | Medium | Stronger than route-only because `queryParameters` helpers are also present |
| `filter_by_tags` | Actor / filtered movie list | `libapp.so` arm64 extracted ASCII strings | Literal query key | Medium | Suggests additional tag filter chaining |
| `main:m` | Category page synthetic main filter | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\provider\category_page_provider.dart` | Direct object construction | High | `TagItem` is created with `id="m"`, localized name `Download`, and `group="main"` |
| `main:p` | Category page synthetic main filter | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\provider\category_page_provider.dart` | Direct object construction | High | `TagItem` is created with `id="p"`, localized name `Playable`, and `group="main"` |
| `:t:m::::` | Category page default download filter branch | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\category_page.dart` | Direct string interpolation before `getCategoryMovies()` | High | `CategoryMovieListState` checks `SpUtil::getBool("movie_default_filter_download", true)` and interpolates `":t:m::::"` into the `filter_by` position passed to `/api/v1/movies/tags` |
| `:t:p::::` | Category page default non-download filter branch | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\category_page.dart` | Direct string interpolation before `getCategoryMovies()` | High | In the false branch of `SpUtil::getBool("movie_default_filter_download", true)`, `CategoryMovieListState` interpolates `":t:p::::"` into the `filter_by` position passed to `/api/v1/movies/tags` |
| `0:l:{list_id}:` | My list detail page derived filter | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\mine\presenter\my_lists_detail_presenter.dart` | Direct string interpolation | High | Presenter builds `filter_by` from literal `\"0:l:\"` + `list_id` + `\":\"` |
| `0:l:{list_id}:p` | My list detail page derived filter | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\mine\presenter\my_lists_detail_presenter.dart` | Direct string interpolation | High | Presenter builds `filter_by` from literal `\"0:l:\"` + `list_id` + `\":p\"` |
| `0:l:{list_id}:m` | My list detail page derived filter | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\mine\presenter\my_lists_detail_presenter.dart` | Direct string interpolation | High | Presenter builds `filter_by` from literal `\"0:l:\"` + `list_id` + `\":m\"` |
| `0:l:{list_id}:c` | My list detail page derived filter | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\mine\presenter\my_lists_detail_presenter.dart` | Direct string interpolation | High | Presenter builds `filter_by` from literal `\"0:l:\"` + `list_id` + `\":c\"` |
| `main:f` | Unknown source | Unknown source | Not recovered | Low | Current APK scan did not recover this literal token as plain string |
| `main:fc2` | Unknown source | Unknown source | Not recovered | Low | Current APK scan did not recover this literal token as plain string |
| `western` | `homeTabWestern`, `actorPageTab_Western_Male`, `actorPageTab_Western_Female` | `libapp.so` arm64 extracted ASCII strings | UI tab string | Low | UI evidence exists, but no direct backend token recovered |
| `uncensored` | `Uncensored`, `uncensored` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | UI/tab string | Low | UI evidence exists, but no direct backend token recovered |
| `censored` | Unknown source | Unknown source | Not recovered | Low | Current APK scan did not recover this literal token as plain string |
| `fc2` | `homeTabFC2`, `get:homeTabFC2` | `libapp.so` arm64 extracted ASCII strings | UI tab string | Low | UI evidence exists, but no direct backend token recovered |
| `category` | Category page flow | `libapp.so` arm64 extracted ASCII strings | Query fragment strings like `?category=d`, `&category=m` | Medium | Category route fragments are present, but not directly bound to `/movies/tags` |
| `page` | Unknown source | Unknown source | Not cleanly attributable | Low | `page` appears widely in Flutter/UI strings; no route-bound API evidence recovered |
| `limit` | Unknown source | Unknown source | Not cleanly attributable | Low | `limit` appears widely in library/UI strings; no route-bound API evidence recovered |

### Recovered Flutter Call Sites

| Call site | Source | Query keys seen | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `CategoryListPresenter::getCategoryMovies` | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\presenter\category_presenter.dart` | `filter_by`, `sort_by`, `order_by`, `page`, `limit` | High | Direct `/api/v1/movies/tags` request builder |
| `HomePagePresenter::getFocusMovies` | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\home\presenter\home_presenter.dart` | `filter_by`, `sort_by`, `page`, `limit` | High | Direct `/api/v1/movies/tags` request builder |
| `ActorInfoNewPagePresenter::getActorMovies` | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\actor\presenter\actor_info_new_presenter.dart` | `filter_by`, `filter_by_tags`, `sort_by`, `order_by`, `page`, `limit` | High | Actor detail flow includes secondary tag chaining |
| `MakerDetailPagePresenter::getDetailMovies` | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\actor\presenter\maker_detail_presenter.dart` | `filter_by`, `sort_by`, `order_by`, `page`, `limit` | High | Presenter interpolates a string containing `":"` before assigning it to `filter_by` |
| `MyListsDetailPagePresenter::getListsMovies` | `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\mine\presenter\my_lists_detail_presenter.dart` | `type=0`, `filter_by`, `sort_by`, `page`, `limit` | High | Request builder is direct; `filter_by` is statically built as `0:l:{list_id}{suffix}` |

### Detail-flow filter construction notes

- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\actor\presenter\maker_detail_presenter.dart` builds a `filter_by` string through `_StringBase::_interpolate` with a literal `":"` before the `/api/v1/movies/tags` request is sent.
- The same presenter's `getMakerInfo` branch selector uses one-letter type codes:
  - `m -> /api/v1/makers/%s`
  - `s -> /api/v1/series/%s`
  - `d -> /api/v1/directors/%s`
  - `p -> /api/v1/publishers/%s`
  - `c -> /api/v1/codes/%s`
- This confirms that detail pages also call `/api/v1/movies/tags`, but these detail-page ids are not currently treated as valid `filter_by` candidate sources for verification.

### My-list filter construction notes

- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\mine\presenter\my_lists_detail_presenter.dart` statically builds `filter_by` through `_StringBase::_interpolate`.
- The recovered literals in that builder are:
  - `"0:l:"`
  - `":"`
  - `":p"`
  - `":m"`
  - `":c"`
- This yields the following APK-backed candidate patterns:
  - `0:l:{list_id}:`
  - `0:l:{list_id}:p`
  - `0:l:{list_id}:m`
  - `0:l:{list_id}:c`
- The same `0:l:` + suffix construction also appears in page-side helpers in `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\mine\my_lists_detail_page.dart`.
- The exact user-facing meaning of the suffixes is still not written as fact here, because this pass only confirmed the string construction, not the final live semantics.

### Category default-download filter notes

- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\category_page.dart` contains a `CategoryMovieListState` async closure that checks `SpUtil::getBool("movie_default_filter_download", true)` before calling `CategoryListPresenter::getCategoryMovies`.
- In the truthy branch, the code interpolates the literal `":t:m::::"` and passes that interpolated value as the first positional argument to `getCategoryMovies`.
- In the false branch, the code interpolates the literal `":t:p::::"` and passes that interpolated value as the first positional argument to `getCategoryMovies`.
- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\presenter\category_presenter.dart` confirms the first positional argument of `getCategoryMovies` is assigned to the `filter_by` query key for `/api/v1/movies/tags`.
- This confirms both `":t:m::::"` and `":t:p::::"` are real APK-backed `filter_by` candidates for the category-page default download toggle branch.
- Subsequent static tracing plus live category verification now supports binding the category type prefix to the visible top-row tabs as `0=Censored`, `1=Uncensored`, `2=Western`, `3=FC2`, `4=Carton/Anime`.

### Lower-sheet slot order notes

- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\provider\category_page_provider.dart` maintains explicit state buckets for `main`, `year`, `month`, `duration`, plus a catch-all list for additional lower-sheet tags.
- The category-page request builder in `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\category_page.dart` appends those buckets in this order:
  - `main`
  - catch-all extra tags
  - `year`
  - `duration`
  - `month`
- This yields the current best-supported `filter_by` grammar for category feeds:
  - `{type}:t:{main}:{extra}:{year}:{duration}:{month}`
- Live verification anchors on 2026-04-26:
  - `0:t:p:23:::` returned a filtered list, proving slot 4 is the generic `extra` slot
  - `0:t:p::2025::` returned titles released in `2025`, proving slot 5 is `year`
  - `0:t:p:::lt-45:` returned short titles such as `34`, `36`, and `38` minutes, proving slot 6 is `duration`
  - `0:t:p::::1` returned January releases, proving slot 7 is `month`

### Top-row tab transport notes

- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\category\category_page.dart` builds a five-item top-row tab list with labels `Mosaic`, `Uncensored`, `Western`, `FC2`, and `Carton`.
- The selected top-row tab is tracked through `field_3b` as a page index and is fed into `TabController::animateTo` / `PageController::jumpToPage`.
- The same category-page flow then calls `CategoryListPresenter::getCategoryTags`, which sends a numeric `type` query parameter to `/api/v2/tags`.
- Cross-page APK evidence from `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\home\presenter\search_new_presenter.dart` shows the shared top-tab `typeIndex` is converted to `movie_type` by subtracting `1`, which yields the stable order `0,1,2,3,4` across the visible tab strip.
- Runtime anchor on 2026-04-25:
  - `/api/v2/tags?type=3` returned the same FC2-style lower filter set seen in the real app screenshot, including `tag` items such as `家庭主婦`, `美少女`, and `自拍`
  - `/api/v2/tags?type=4` returned anime/cartoon-style filter groups, including `subject` items such as `動作/戰鬥`, `冒險`, and `機器人`
- Current best-supported top-row `/api/v2/tags?type` mapping:

| Top tab label | `type` | Evidence | Confidence | Notes |
| --- | --- | --- | --- | --- |
| `Mosaic` | `0` | `CategoryPageState::build` creates five `CategoryMovieList` pages and stores the PageView index into `CategoryMovieList.field_b`; `CategoryMovieListState` then passes `widget.field_b` directly to `getCategoryTags`, which requests `/api/v2/tags?type=<index>`; runtime `/api/v2/tags?type=0` groups match the `有碼` screenshot | High | Direct category-page transport proof plus screenshot-backed group match |
| `Uncensored` | `1` | `CategoryPageState::build` creates five `CategoryMovieList` pages and stores the PageView index into `CategoryMovieList.field_b`; `CategoryMovieListState` then passes `widget.field_b` directly to `getCategoryTags`, which requests `/api/v2/tags?type=<index>`; runtime `/api/v2/tags?type=1` groups match the `無碼` screenshot | High | Direct category-page transport proof plus screenshot-backed group match |
| `Western` | `2` | `CategoryPageState::build` creates five `CategoryMovieList` pages and stores the PageView index into `CategoryMovieList.field_b`; `CategoryMovieListState` then passes `widget.field_b` directly to `getCategoryTags`, which requests `/api/v2/tags?type=<index>`; runtime `/api/v2/tags?type=2` groups match the `歐美` screenshot | High | Direct category-page transport proof plus screenshot-backed group match |
| `FC2` | `3` | `CategoryPageState::build` creates five `CategoryMovieList` pages and stores the PageView index into `CategoryMovieList.field_b`; `CategoryMovieListState` then passes `widget.field_b` directly to `getCategoryTags`, which requests `/api/v2/tags?type=<index>`; runtime `/api/v2/tags?type=3` groups match the FC2 screenshot | High | Direct category-page transport proof plus screenshot-backed group match |
| `Carton` | `4` | `CategoryPageState::build` creates five `CategoryMovieList` pages and stores the PageView index into `CategoryMovieList.field_b`; `CategoryMovieListState` then passes `widget.field_b` directly to `getCategoryTags`, which requests `/api/v2/tags?type=<index>`; runtime `/api/v2/tags?type=4` groups match the `動漫` screenshot | High | APK label is `Carton`; screenshot label is `動漫` |
- Static and screenshot-backed evidence therefore supports this narrower interpretation:
  - the visible top-row tabs definitely drive page selection and a category `type`
  - the direct category-page transport chain is: top-row `PageView.builder` index -> `CategoryMovieList.field_b` -> `CategoryMovieListState::getCategoryTags(widget.field_b)` -> `/api/v2/tags?type=<index>`
  - the category-page `/api/v2/tags?type` mapping can now be treated as `有碼 -> 0`, `無碼 -> 1`, `歐美 -> 2`, `FC2 -> 3`, `動漫 -> 4`
  - but this scan still has not recovered a direct `/api/v1/movies/tags` backend token for `Uncensored / Western / FC2 / Carton`
  - those tab labels should remain UI-only evidence until a direct token or call-site mapping is recovered

### Screenshot-backed UI interpretation notes

- Real app screenshots of the category page show a two-layer filtering UI:
  - a top-row primary category strip such as `有碼 / 無碼 / 歐美 / FC2 / 動漫`
  - a lower filter sheet with chip groups such as `基本`, `年份`, `月份`, `標籤`, and `時長`
- This UI structure supports treating the top-row category strip as the highest-priority source for `filter_by` candidates.
- The lower chip groups should not be assumed to reuse the same simple `filter_by=...` transport until the APK shows their exact request encoding.
- This interpretation is used only to prioritize verification and candidate grouping; it does not by itself prove any backend token.

### Lower-filter transport notes

- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\home\presenter\latest_presenter.dart` maps a small filter enum to `all`, `magnets`, `can_play`, and `subtitle`, and sends that value as `filter_by` to `/api/v1/movies/latest`.
- `F:\codx\javdbapkre\recovered_flutter_source\lib\astarte\home\presenter\search_new_presenter.dart` maps a similar filter enum to `all`, `can_play`, `magnets`, `subtitle`, and `single`, and sends that value as `movie_filter_by` to `/api/v2/search`.
- This is strong APK evidence that the lower filter-sheet chips such as `可播放 / 可下載 / 含字幕 / 單體影片` are part of a broader filter vocabulary reused across other movie-list/search endpoints.
- Because those values are already proven on `/api/v1/movies/latest` and `/api/v2/search`, they should not automatically be assumed to be top-row category tokens for `/api/v1/movies/tags`.

### Live-validation caution on 2026-04-25

- Static APK evidence strongly supports `main:m` and `main:p` as category-page synthetic filters.
- Actor info pages, maker detail pages, and list detail pages also call the endpoint, but those page-level call sites should not be conflated with authoritative `filter_by` candidate sources.

## Special Section: FormData APIs

| Route | Source | Fields | Likely body style | Confidence |
| --- | --- | --- | --- | --- |
| `/api/v1/sessions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` + login field strings | `username`, `password`, `device_uuid`, `device_name`, `device_model`, `system_version`, `app_channel`, `app_version`, `app_version_number` | Form-based / Multipart candidate | Medium |
| `/api/v2/search_image` | `libapp.so` arm64/armeabi-v7a route string + `SearchImagePage` flow + `FormData.fromMap` | Unknown image/file field | Form-based / Multipart candidate | Medium |
| `/api/v1/wallets/bind_withdraw_account` | `libapp.so` arm64/armeabi-v7a route string + withdraw bind flow + `FormData.fromMap` | `withdraw_type`, `code`, bank/account-related fields | Form-based / Multipart candidate | Medium |
| `/api/v2/wallets/withdraw` | `libapp.so` arm64/armeabi-v7a route string + withdraw flow + `FormData.fromMap` | `withdraw_account_id` | Form-based / Multipart candidate | Medium |
| `/api/v2/plans/payment_order` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `plan_id`, `platform_id`, `method_id` | Form-based / Multipart candidate | Medium |
| `/api/v3/plans/payment_order` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `plan_id`, `platform_id`, `method_id` | Form-based / Multipart candidate | Medium |
| `/api/v1/actors/%s/collect_actions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `name` candidate only | Form-based / Multipart candidate | Low |
| `/api/v1/directors/%s/collect_actions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `name` candidate only | Form-based / Multipart candidate | Low |
| `/api/v1/makers/%s/collect_actions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `name` candidate only | Form-based / Multipart candidate | Low |
| `/api/v1/series/%s/collect_actions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `name` candidate only | Form-based / Multipart candidate | Low |
| `/api/v1/lists/%s/collect_actions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `name` candidate only | Form-based / Multipart candidate | Low |
| `/api/v1/lists/%s/movie_actions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `movie_id`, `name` | Form-based / Multipart candidate | Low |
| `/api/v1/codes/%s/collect_actions` | `libapp.so` arm64/armeabi-v7a route string + `FormData.fromMap` | `name` candidate only | Form-based / Multipart candidate | Low |

## Special Section: Query APIs

| Route | Source | Query params | Confidence |
| --- | --- | --- | --- |
| `/api/v1/movies/tags` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` + `filter_by` family strings | `filter_by`, `filter_by_tags`, `sort_by`, `order_by` | Medium |
| `/api/v1/movies/%s/play` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` + playback field strings | `source_id`, `resolution`, `fromRankings`, `operation` | Medium |
| `/api/v1/movies/%s/resume_play` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` + playback field strings | `source_id`, `episode`, `resolution` | Medium |
| `/api/v2/search` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` | Unknown | Low |
| `/api/v1/search_magnet` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` | Unknown | Low |
| `/api/v1/reviews/hotly` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` | Unknown | Low |
| `/api/v1/users/transaction_logs` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` | Unknown | Low |
| `/api/v1/rankings` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` | Unknown | Low |
| `/api/v1/rankings/actors` | `libapp.so` arm64/armeabi-v7a route string + `queryParameters` | Unknown | Low |

## Unknown / Orphan Route Strings

| Route | Source | Confidence | Next search |
| --- | --- | --- | --- |
| `/api/v2/tags` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search for tag catalog presenter/provider bindings |
| `/api/v1/rankings/playbackP` | `libapp.so` arm64 extracted ASCII strings | Low | Search older build / other ABI for playback ranking presenter bindings |
| `/api/v1/actors/batch_uncollection` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search actor collection management flow |
| `/api/v1/users/unpaid_tickets` | `libapp.so` arm64 extracted ASCII strings | Low | Search payment/order presenter and unpaid-order pages |
| `/api/v1/following_tags/%s/sort` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search tag-manage presenter call chain |
| `/api/v1/following_tags/batch_destroy` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search tag-manage presenter call chain |
| `/api/v1/following_tags/batch_push` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search tag-manage presenter call chain |
| `/api/v2/users/%s/reviews` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search user-review presenter flow |
| `/api/v2/users/review_movies` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search review-movie list provider |
| `/api/v1/users/collected_actors` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search collection page provider bindings |
| `/api/v1/users/collected_makers` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search collection page provider bindings |
| `/api/v1/users/collected_directors` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search collection page provider bindings |
| `/api/v1/users/collected_series` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search collection page provider bindings |
| `/api/v1/users/collected_codes` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search collection page provider bindings |
| `/api/v1/users/collected_lists` | `libapp.so` arm64/armeabi-v7a extracted ASCII strings | Low | Search collection page provider bindings |
