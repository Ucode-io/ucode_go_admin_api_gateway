# Meta Ads Analytics

The gateway reads Meta Marketing API data on every authenticated dashboard request. It does not persist advertising data in PostgreSQL.

## Configuration

```env
META_GRAPH_BASE_URL=https://graph.facebook.com
META_GRAPH_VERSION=v26.0
META_AD_ACCOUNT_ID=843587364827107
META_ACCESS_TOKEN=<SYSTEM_USER_TOKEN>
META_ADS_CACHE_TTL_SEC=1800
META_ADS_REQUEST_TIMEOUT_SEC=30
META_ADS_MAX_RANGE_DAYS=90
META_LEAD_ACTION_TYPES=lead
META_ATTRIBUTION_WINDOWS=
```

`META_ACCESS_TOKEN` must be stored in backend secret storage. Do not add it to Git, frontend variables, request parameters, or logs. The account ID may be supplied with or without the `act_` prefix.

`META_LEAD_ACTION_TYPES` is an allowlist. Compare `observed_action_types` from a real 7–30 day response with Ads Manager before changing the production allowlist. Leaving `META_ATTRIBUTION_WINDOWS` empty uses the ad account attribution setting.

## Endpoints

Both endpoints require the existing gateway authentication middleware.

```http
GET /v1/meta-ads/account
GET /v1/meta-ads/dashboard?since=2026-08-01&until=2026-08-20
```

Dashboard filters:

- `campaign_id`, `ad_set_id`, `ad_id`
- `objective`, `effective_status`
- `age`, `gender`, `country`, `region`
- `publisher_platform`, `platform_position`, `device_platform`
- `breakdowns=age_gender,country,region,placement,device`

Comma-separated values are accepted for every filter except object IDs. The default date range is the latest seven UTC dates, the maximum is controlled by `META_ADS_MAX_RANGE_DAYS`, and all supported breakdowns are returned by default.

## Freshness and fallback

The handler checks Redis for the last successful response, requests fresh data directly from Meta, and overwrites Redis after success. A successful fresh response contains:

```json
{"source":"meta","stale":false}
```

If Meta returns a transient error and a previous response exists, the gateway returns it as an explicit fallback:

```json
{"source":"cache","stale":true,"warning":"Meta API is temporarily unavailable; returning the last successful response"}
```

Redis is not the primary data source and is never used to skip the fresh Meta request. Cache keys contain only the public ad account ID and normalized dashboard filters; they never contain the access token.

## Response sections

- Account currency and timezone
- KPI totals: spend, impressions, reach, frequency, clicks, link clicks, landing page views, CTR, CPC, CPM, leads, and nullable CPL
- Daily trends
- Campaign → ad set → ad hierarchy with creative metadata
- Age/gender, country, region, placement, and device breakdowns
- Observed Meta action types for lead allowlist verification

Meta frequently returns numbers as strings. The gateway converts counts to integers, values to floating-point JSON numbers, and returns `cpl: null` when leads are zero. Personal Instant Form lead fields are outside this endpoint and require the existing Leads integration permissions.
