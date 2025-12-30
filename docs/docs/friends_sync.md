Friends Sync API (Android + Web)
===============================

Overview
--------
The backend exposes two ways to fetch friend data:
1) A structured overview (friends + incoming/outgoing requests)
2) A flattened connections list (single array with status fields)

Android can use either. The **connections** endpoint is recommended because it
matches typical mobile list UIs.

Authentication
--------------
- Cookie session auth (`mtg_session`)
- Android should use a CookieJar and send cookies with every request
- 401 if missing/invalid cookie

Endpoints
---------

GET /v1/friends
  - Returns a FriendsOverview object:
    - `friends`: array of UserSummary
    - `incoming_requests`: array of FriendRequest
    - `outgoing_requests`: array of FriendRequest

Example:
```
GET /v1/friends

200 OK
{
  "friends": [
    {
      "id": "user-2",
      "username": "alice",
      "display_name": "Alice",
      "avatar_path": "user-2-1717246496789000000.jpg",
      "avatar_updated_at": "2024-06-01T12:34:56.789Z"
    }
  ],
  "incoming_requests": [
    {
      "id": "req-1",
      "user": {
        "id": "user-3",
        "username": "bob",
        "display_name": "",
        "avatar_path": "",
        "avatar_updated_at": null
      },
      "created_at": "2024-06-02T09:10:11.123Z"
    }
  ],
  "outgoing_requests": []
}
```

GET /v1/friends/connections
  - Returns a single array of FriendConnection items:
    - `user`: UserSummary
    - `status`: "accepted" | "incoming" | "outgoing"
    - `request_id`: present for incoming/outgoing
    - `created_at`: present for incoming/outgoing

Example:
```
GET /v1/friends/connections

200 OK
[
  {
    "user": {
      "id": "user-2",
      "username": "alice",
      "display_name": "Alice",
      "avatar_path": "user-2-1717246496789000000.jpg",
      "avatar_updated_at": "2024-06-01T12:34:56.789Z"
    },
    "status": "accepted"
  },
  {
    "user": {
      "id": "user-3",
      "username": "bob",
      "display_name": "",
      "avatar_path": "",
      "avatar_updated_at": null
    },
    "status": "incoming",
    "request_id": "req-1",
    "created_at": "2024-06-02T09:10:11.123Z"
  }
]
```

POST /v1/friends/requests
  - Create a friend request.
  - Request JSON:
```
{ "username": "targetUsername" }
```
  - Response (201):
```
{
  "id": "req-123",
  "user": {
    "id": "user-2",
    "username": "alice",
    "display_name": "Alice",
    "avatar_path": "user-2-1717246496789000000.jpg",
    "avatar_updated_at": "2024-06-01T12:34:56.789Z"
  },
  "created_at": "2024-06-02T09:10:11.123Z"
}
```

POST /v1/friends/requests/{id}/accept
  - Accept a pending request.
  - Response: 204 No Content

POST /v1/friends/requests/{id}/decline
  - Decline a pending request.
  - Response: 204 No Content

POST /v1/friends/requests/{id}/cancel
  - Cancel a pending request sent by the current user.
  - Response: 204 No Content

UserSummary fields
------------------
- `id` (string)
- `username` (string)
- `display_name` (string, may be empty)
- `avatar_path` (string, may be empty)
- `avatar_updated_at` (string RFC3339 with milliseconds or null)

FriendRequest fields
--------------------
- `id` (string)
- `user` (UserSummary)
- `created_at` (string RFC3339 with milliseconds)

FriendConnection fields
-----------------------
- `user` (UserSummary)
- `status` ("accepted" | "incoming" | "outgoing")
- `request_id` (string, only for incoming/outgoing)
- `created_at` (string RFC3339 with milliseconds, only for incoming/outgoing)

Error responses
---------------
All errors use:
```
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

Common codes:
- `unauthorized` (401)
- `validation_error` (400)
- `not_found` (404)
- `friendship_exists` (409)

Android integration notes
-------------------------
1) Recommended: call `GET /v1/friends/connections` and map:
   - `status=accepted` → friend list
   - `status=incoming` → accept/decline actions
   - `status=outgoing` → show pending + cancel action
2) For avatars, prefer `avatar_url` from `/v1/users/me` for the current user.
   For friends, build URLs as `/app/avatars/{avatar_path}?v={avatar_updated_at_unix}`
   or display a default avatar if `avatar_path` is empty.
3) On accept/decline/cancel, refresh the connections list to stay in sync.
