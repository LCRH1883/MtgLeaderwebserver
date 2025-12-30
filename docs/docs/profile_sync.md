Profile Sync API (Mobile + Web)
===============================

Overview
--------
User profiles are synced using a timestamp-based conflict rule. Clients must send
`updated_at` with every profile update. If the incoming timestamp is not newer
than the stored value, the update is rejected with `409` and the current user
payload so the client can refresh and retry.

Timestamp contract
------------------
- Field name: `updated_at`
- Format: RFC3339 UTC with milliseconds, e.g. `2024-06-01T12:34:56.789Z`
- Required on every profile update (display name + avatar)
- Returned in all profile responses:
  - `POST /v1/auth/register`
  - `POST /v1/auth/login`
  - `POST /v1/auth/google`
  - `POST /v1/auth/apple`
  - `GET /v1/users/me`
  - `PATCH /v1/users/me`
  - `POST /v1/users/me/avatar`
  - `PATCH /v1/users/me/avatar`

Endpoints
---------

GET /v1/users/me
  - Optional: `include_stats=true` to include `stats_summary`.
  - Response (200):
    - id, email, username, display_name, avatar, avatar_path, avatar_url
    - created_at, updated_at
    - stats_summary (optional)

Example (include_stats=true):
```
GET /v1/users/me?include_stats=true

200 OK
{
  "id": "user-123",
  "email": "player@example.com",
  "username": "player1",
  "display_name": "Player One",
  "avatar": "user-123-1717246496789000000.jpg",
  "avatar_path": "user-123-1717246496789000000.jpg",
  "avatar_updated_at": "2024-06-01T12:34:56.789Z",
  "avatar_url": "/app/avatars/user-123-1717246496789000000.jpg?v=1717246496",
  "created_at": "2024-05-01T09:15:00Z",
  "updated_at": "2024-06-01T12:34:56.789Z",
  "stats_summary": {
    "matches_played": 12,
    "wins": 5,
    "losses": 7,
    "avg_turn_seconds": 110
  }
}
```

PATCH /v1/users/me
PUT /v1/users/me
POST /v1/users/me
  - Request JSON:
    - `display_name`: string (empty string clears the name)
    - `updated_at`: RFC3339 with milliseconds
  - Response:
    - 200 + full user payload on success
    - 409 + full user payload if update is stale

Example:
```
PATCH /v1/users/me
{
  "display_name": "Player One",
  "updated_at": "2024-06-01T12:34:56.789Z"
}

200 OK
{ ...full user payload... }
```

Conflict example:
```
409 Conflict
{ ...current user payload... }
```

POST /v1/users/me/avatar
PATCH /v1/users/me/avatar
PUT /v1/users/me/avatar
  - Multipart form:
    - `avatar`: 96x96 image file (jpeg/png/gif accepted by server)
    - `updated_at`: RFC3339 with milliseconds
  - Response:
    - 200 + full user payload on success
    - 409 + full user payload if update is stale

Example:
```
POST /v1/users/me/avatar
Content-Type: multipart/form-data

avatar=@avatar.jpg
updated_at=2024-06-01T12:34:56.789Z
```

Conflict responses
------------------
If `updated_at` is older or equal to the stored value, the update does not apply.
The server responds with `409 Conflict` and the current user payload so the
client can refresh local state.

Invalid requests
----------------
- `invalid_updated_at` (400): missing or malformed timestamp.
- `validation_error` (400): invalid display name.
- `invalid_avatar` (400): missing/invalid avatar or wrong dimensions.

Error envelope example:
```
400 Bad Request
{
  "error": {
    "code": "invalid_updated_at",
    "message": "updated_at must be RFC3339 UTC with milliseconds"
  }
}
```

Caching and invalidation
------------------------
- Profile responses return `ETag` based on `updated_at`.
- `GET /v1/users/me` sets:
  - `Cache-Control: private, max-age=0` (without stats)
  - `Cache-Control: no-store` (with stats)
- Avatar URLs are versioned and served with long-lived caching:
  - `/app/avatars/*` sets `Cache-Control: public, max-age=31536000, immutable`
  - Avatar URL changes whenever `updated_at` changes.

Android integration notes
-------------------------
1) On login or app start, call `GET /v1/users/me?include_stats=true` and store
   the full profile payload.
2) When updating display name, generate a NEW `updated_at` value (UTC with
   milliseconds) and send it with the update.
   - If 200: replace local profile with response.
   - If 409: replace local profile with response and prompt user to retry with
     a fresh timestamp.
3) When uploading avatar, include a NEW `updated_at` as a form field (same
   format as above).
   - If 200: replace local profile with response.
   - If 409: refresh local profile and re-render.
4) Use `avatar_url` as the cache-busted image source.
5) Optional: store `ETag` from `GET /v1/users/me` and send `If-None-Match`
   on subsequent syncs (only when `include_stats=false`).
