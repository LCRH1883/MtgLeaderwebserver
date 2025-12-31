Matches Sync API
================

Overview
--------
The matches endpoint supports offline-first clients using:
- A required `updated_at` timestamp (RFC3339 UTC with milliseconds)
- Optional client-side idempotency via `client_match_id`

Endpoint
--------

POST /v1/matches
  - Creates a match (or returns a conflict if the client id already exists).

Request JSON:
```
{
  "updated_at": "2025-12-29T20:00:00.000Z",
  "client_match_id": "device-generated-uuid",
  "played_at": "2025-12-29T20:00:00Z",
  "format": "commander",
  "total_duration_seconds": 5400,
  "turn_count": 12,
  "results": [
    {"id":"USER_1","rank":1},
    {"id":"USER_2","rank":2}
  ]
}
```

Notes:
- `updated_at` is required and must be RFC3339 UTC with milliseconds.
- `client_match_id` is recommended for idempotency.
- `client_ref` is accepted as a legacy alias for `client_match_id`.

Success response (201):
```
{
  "match": { "...": "..." },
  "stats_summary": { "...": "..." }
}
```

Conflict response (409):
```
{
  "match": { "...": "..." }
}
```

Conflict semantics:
- If a match already exists for the same `(created_by, client_match_id)`, the server returns 409.
- Matches are immutable; even newer `updated_at` values return the existing match.
