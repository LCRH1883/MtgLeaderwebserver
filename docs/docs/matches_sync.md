Matches Sync API
================

Overview
--------
The matches endpoint supports offline-first clients using:
- Optional `updated_at` timestamp (RFC3339 UTC with milliseconds)
- Client-side idempotency via `client_match_id`

Endpoint
--------

POST /v1/matches
  - Creates a match (or returns the existing match if the client id already exists).

Request JSON:
```
{
  "client_match_id": "device-generated-uuid",
  "updated_at": "2025-12-29T20:00:00.000Z",
  "started_at": "2025-12-29T18:00:00Z",
  "ended_at": "2025-12-29T20:00:00Z",
  "format": "commander",
  "total_duration_seconds": 5400,
  "turn_count": 12,
  "players": [
    {
      "seat_index": 0,
      "user_id": "USER_1",
      "display_name": "Player One",
      "place": 1,
      "eliminated_turn_number": null,
      "eliminated_during_seat_index": null,
      "total_turn_time_ms": 123456,
      "turns_taken": 15
    },
    {
      "seat_index": 1,
      "guest_name": "Guest",
      "display_name": "Guest",
      "place": 2,
      "eliminated_turn_number": 10,
      "eliminated_during_seat_index": 0
    }
  ]
}
```

Notes:
- `updated_at` is optional; if provided it must be RFC3339 UTC with milliseconds.
- `client_match_id` is recommended for idempotency.
- `client_ref` is accepted as a legacy alias for `client_match_id`.
- Each player must include exactly one of `user_id` or `guest_name`.
- `place` must be >= 1 and exactly one player must have `place = 1`.
- `user_id` players must be the creator or an accepted friend.

Success response (201):
```
{
  "match_id": "MATCH_ID",
  "match": { "...": "..." },
  "stats_summary": { "...": "..." }
}
```

Idempotent response (200):
```
{
  "match_id": "MATCH_ID",
  "match": { "...": "..." }
}
```

Idempotency semantics:
- If a match already exists for the same `(created_by, client_match_id)`, the server returns 200 with the existing match.
- Matches are immutable; newer `updated_at` values still return the existing match.

Legacy payloads
---------------
Older clients may still use `player_ids` + `winner_id` or `results` with `rank` fields.
