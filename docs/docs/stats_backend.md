# MTG Stats Backend: Formats, Placements, Friend Stats

## Purpose

This backend stores and computes statistics for a small friend group playing Magic: The Gathering across multiple formats. It supports:

- Recording completed games (“matches”) with:
  - format (commander/brawl/standard/modern)
  - who played
  - winner
  - placement/rank for each player (ties allowed)
  - played timestamp
  - total playtime (seconds)
  - turn count
- Computing:
  - overall stats per player
  - per-format stats per player
  - friend-vs-friend stats (wins/losses/co-losses)
  - “most often beat” and “most often beats you”
- Serving data for:
  - phone app (JSON API)
  - web UI (server-rendered pages)

This document defines the data model, ranking semantics (ties), API contract, and how stats are computed.

---

## Terminology

### Match / Game
A single session of play. The backend uses “match” in table and endpoint names, but conceptually it is a “game”.

### Format
A string enum:
- `commander`
- `brawl`
- `standard`
- `modern`

### Completed match
A match is considered “completed” if `matches.winner_id IS NOT NULL`.
Only completed matches count toward wins/losses and “games played” stats.

### Rank / Place
An integer where:
- `1` is winner / first place
- Higher numbers are worse placement
- Ties are allowed for simultaneous eliminations

---

## Tie and placement semantics

### What “tie” means
If multiple players lose simultaneously on the same player’s turn, they are treated as tied for that elimination moment.

### How ties appear in stored data
The backend stores a `rank` per player in `match_player_results.rank`. If players are tied, they share the same rank number.

**Important:** Ranks may have gaps when ties occur. This follows “competition ranking”.

### Competition ranking definition
If two players tie for 3rd, the next rank is 5th (rank 4 is skipped). Example:

- 1st: Player A
- 2nd: Player B
- 3rd: Player C
- 3rd: Player D
- 5th: Player E

This is valid and expected.

### Recommended rank assignment algorithm (client-side)
If the phone app tracks eliminations as events (batches), assign ranks like:

- Let `alive_before` be how many players are still alive just before an elimination event.
- Let `k` be how many players are eliminated in that event.
- After the event, `alive_after = alive_before - k`.
- All eliminated players get rank = `alive_after + 1`.

Examples:

#### Example 1: 5 players, single eliminations
Alive 5 → eliminate 1 → alive_after 4 → eliminated gets rank 5  
Alive 4 → eliminate 1 → alive_after 3 → eliminated gets rank 4  
Alive 3 → eliminate 1 → alive_after 2 → eliminated gets rank 3  
Alive 2 → eliminate 1 → alive_after 1 → eliminated gets rank 2  
Winner gets rank 1

Ranks: 1,2,3,4,5 (no ties)

#### Example 2: 5 players, first elimination is a tie (2 die same moment)
Alive 5 → eliminate 2 → alive_after 3 → both eliminated rank 4  
Alive 3 → eliminate 1 → alive_after 2 → eliminated rank 3  
Alive 2 → eliminate 1 → alive_after 1 → eliminated rank 2  
Winner rank 1

Ranks: 1,2,3,4,4 (note: no “5th” because of the tie)

---

## Database schema

This backend uses Postgres.

### `matches` table (extended)

Existing table. Extended with:

- `format TEXT NOT NULL`  
  Must be one of: commander/brawl/standard/modern

- `total_duration_seconds INT NOT NULL DEFAULT 0`  
  Total game playtime in seconds.

- `turn_count INT NOT NULL DEFAULT 0`  
  Total turns played. If unknown, keep 0.

- `client_ref TEXT NULL UNIQUE`  
  Optional idempotency key from the phone app to prevent duplicate uploads.

A match is “completed” when `winner_id` is non-null.

### `match_players` table

Associates participants to a match.

- `match_id`
- `user_id`

### `match_player_results` table (new)

Stores placements and optional elimination metadata.

Columns:
- `match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE`
- `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`
- `rank INT NOT NULL CHECK(rank >= 1)`
- `elimination_turn INT NULL CHECK(elimination_turn >= 0)`
- `elimination_batch INT NULL CHECK(elimination_batch >= 0)`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

Primary key:
- `(match_id, user_id)`

Indexes:
- `user_id` for fast per-user stats
- `(match_id, rank)` for fast ordering in match detail

---

## API design

### POST `/v1/matches` — create a match

#### Preferred payload (v2)
```json
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

Notes:
- `updated_at` is optional; if provided it must be RFC3339 UTC with milliseconds.
- `client_match_id` is strongly recommended for idempotency.
- `client_ref` is accepted as a legacy alias for `client_match_id` if needed.


Validation rules:

Must contain at least 2 unique players.

Exactly one place = 1 (single winner).

Seats must be unique and contiguous from 0..N-1.

Each player must include exactly one of user_id or guest_name.

The creator must be included in the player list.

The backend derives winner_id from place=1 and sets matches.winner_id.

Only the creator or accepted friends can be included as user_id entries.

Idempotency:

If (created_by, client_match_id) already exists, return 200 with the existing match record.
Matches are immutable; newer `updated_at` values still return the existing record.

Response:

```
{
  "match_id": "MATCH_UUID",
  "match": { "id": "MATCH_UUID", "...": "..." },
  "stats_summary": { "...": "..." }
}
```

Legacy payload (backward compatibility)

Older clients may send:

{
  "played_at": "2025-12-29T20:00:00Z",
  "winner_id": "USER_1",
  "player_ids": ["USER_2","USER_3"]
}


The backend still stores matches and match_players. Results/ranks may be absent.

GET /v1/matches — list matches for user

Response includes extended fields:

format, duration, turn_count

players with rank when available

GET /v1/matches/{id} — match detail

Return players ordered by:

rank ascending (1 first)

username as stable tie-breaker

GET /v1/stats/summary — user summary stats

Response (example shape):

{
  "matches_played": 42,
  "wins": 12,
  "losses": 30,
  "win_pct": 0.2857,
  "avg_turn_seconds": 75,
  "by_format": {
    "commander": {"matches_played": 20, "wins": 7, "losses": 13, "avg_turn_seconds": 80},
    "modern": {"matches_played": 22, "wins": 5, "losses": 17, "avg_turn_seconds": 70}
  },
  "most_often_beat": {"opponent":{"id":"...","username":"bob"}, "count": 6},
  "most_often_beats_you": {"opponent":{"id":"...","username":"alice"}, "count": 9},
  "guest_head_to_head": [{"guest_name":"Guest","wins":2,"losses":1}]
}


Definitions:

matches_played: completed matches (winner_id not null) where user participated

wins: completed matches where winner_id = user

losses: completed matches where winner_id != user

win_pct: wins / matches_played (0 when matches_played is 0)

avg_turn_seconds: weighted average across matches:

SUM(total_duration_seconds) / SUM(turn_count) for completed matches with turn_count > 0

guest_head_to_head: winner-only guest stats for the authenticated user

GET /v1/stats/head-to-head/{id} — stats vs a specific opponent

Response includes:

total: completed matches where both participated

wins: matches where user was winner and opponent participated

losses: matches where opponent was winner and user participated

co_losses: matches where both participated and winner was neither of them

Optional:

by_format breakdown of the same metrics

GET /v1/stats/friends — stats vs each friend

Returns an array:

[
  {"friend":{"id":"...","username":"bob"}, "total":10, "wins":4, "losses":3, "co_losses":3},
  {"friend":{"id":"...","username":"carol"}, "total":7, "wins":1, "losses":5, "co_losses":1}
]

How stats are computed (SQL-level definitions)
Completed matches participated

A match counts only if:

user is in match_players

matches.winner_id IS NOT NULL

Wins

A win is:

user participated AND matches.winner_id = user_id

Losses

A loss is:

user participated AND matches.winner_id IS NOT NULL AND matches.winner_id != user_id

Games played together (A,B)

A match counts as “together” if both exist in match_players for that match and winner_id IS NOT NULL.

A wins vs B

Count matches where:

both participated

winner_id = A

A losses vs B

Count matches where:

both participated

winner_id = B

Co-losses (both lost together)

Count matches where:

both participated

winner_id IS NOT NULL

winner_id != A AND winner_id != B

Security / Permissions

All /v1/* endpoints require an authenticated user unless explicitly public.

Match creation enforces the invariant:

all participants must be either the creator or confirmed friends of the creator

Stats endpoints only reveal:

the current user’s stats

friend/opponent stats where friendship/visibility rules permit (typically: only friends)

Web UI pages

The web UI is a server-rendered view over the same service layer used by /v1/*.

Routes:

/app/stats: renders /v1/stats/summary data

/app/matches: renders /v1/matches data

/app/matches/{id}: renders match detail

/app/friends: renders /v1/stats/friends data

Extensibility notes

Future enhancements can be added without breaking the core model:

per-turn timing breakdowns

“first eliminated” trophies

per-player per-turn contributions

deck tracking (commander identity, etc.)

match “status” field (completed/voided) if needed


---
