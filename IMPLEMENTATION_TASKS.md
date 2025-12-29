# MtgLeaderwebserver: Implementation Tasks (Auth → Friends → Matches)

This is a production-minded, step-by-step task guide to build the backend described in your design, starting with **user auth** (register/login/logout/me) and then extending to **friends**, **matches**, and **stats**.

The intent is: complete each step in order, with clear “done” checks, and with references to the other modules that must exist for the step to work end-to-end.

---

## Conventions (apply to all steps)

### API basics
- Base path: `/v1`
- JSON only (request/response), except `/healthz`.
- Errors use a consistent JSON envelope (defined in **Step 5**).
- Production base URL: `https://mtgleader.xyz`.

### Auth approach (MVP)
- **Server-side sessions in Postgres**.
- Cookie name: `mtg_session`
- Cookie attributes (prod behind HTTPS): `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`
- Session TTL (example): 30 days

### Package/layout target
You currently have only `main.go`. As you implement, migrate to this layout (created in **Step 1**):

```
cmd/
  server/
    main.go
internal/
  config/
  domain/
  httpapi/
  auth/
  store/
    postgres/
  service/
migrations/
```

Layering rules:
- `internal/httpapi`: HTTP parsing/validation/status codes only.
- `internal/service`: business rules (permissions + invariants).
- `internal/store/postgres`: SQL + persistence only.
- `internal/domain`: shared types + sentinel errors.
- `internal/auth`: password hashing + session helpers.

---

## Phase 0 — Local prerequisites (one-time)

### 0.1 Install local tools (dev machine/server)
Tasks:
- Install Go (pick a stable version and keep it consistent across dev + CI).
- Install Postgres locally (or ensure reachable DSN).
- Pick a migration tool: `goose` recommended (CLI + SQL migrations).

Done when:
- You can run `go version`.
- You can connect to Postgres using a DSN you control.
- You can run the migration tool from your shell.

Notes / references:
- Migration scripts are created in **Step 3** and executed in **Step 4**.

---

## Phase 1 — Repository scaffolding (no business logic yet)

### 1.1 Create the package layout
Tasks:
- Move the current `main.go` into `cmd/server/main.go`.
- Add minimal `internal/` folders (empty packages for now).
- Add `.gitignore` (at least: binaries, local env files, IDE folders).

Done when:
- `go run ./cmd/server` builds and runs (even if it just serves `/healthz` later).

References:
- `/cmd/server/main.go` will call config loader (**Step 2**) and HTTP router (**Step 6**).

### 1.2 Decide initial dependency set (keep minimal)
Tasks:
- Add router: `github.com/go-chi/chi/v5`
- Add Postgres driver/pool: `github.com/jackc/pgx/v5/pgxpool`
- (Optional) Add UUID library if you don’t want to rely on DB-generated UUIDs.

Done when:
- `go mod tidy` completes cleanly.

References:
- Router used in **Step 6**.
- Database connection used in **Step 4**.

---

## Phase 2 — Configuration and process setup

### 2.1 Define config contract (environment variables)
Create `internal/config/config.go`.

Tasks:
- Define a `Config` struct with fields at minimum:
  - `Env` (dev/prod)
  - `Addr` (e.g., `127.0.0.1:8080`)
  - `PublicURL` (used for cookie/security decisions)
  - `DBDSN`
  - `CookieSecret` (32+ random bytes)
  - `SessionTTL`
  - `LogLevel`
- Implement `Load()` that reads env vars, validates them, and returns a typed config.

Done when:
- Starting the server without required env vars returns a clear error.
- Starting with valid env vars prints a single startup line showing env + addr (no secrets).

References:
- `cmd/server/main.go` loads config and passes it into:
  - DB init (**Step 4**)
  - HTTP router and middleware (**Step 6/7**)
  - Session cookie signing/encryption decisions (**Step 7**)

### 2.2 Add a `/healthz` endpoint early
Tasks:
- Implement `GET /healthz` that returns `200 OK` with plain text `ok`.
- Optionally add DB ping in `healthz` once DB is wired (**Step 4**).

Done when:
- `curl https://mtgleader.xyz/healthz` returns `ok` (local dev: `http://127.0.0.1:8080/healthz`).

References:
- Used by systemd/Nginx later; also helps you verify routing is correct before auth work.

---

## Phase 3 — Database schema and migrations (auth tables first)

### 3.1 Create initial migrations (users + sessions)
Create SQL migration(s) under `migrations/`.

Minimum tables for auth:
- `users`
  - `id` UUID PK
  - `email` (unique, nullable or required—choose now)
  - `username` (unique, required)
  - `password_hash` (text, required)
  - timestamps: `created_at`, `updated_at`, `last_login_at` (nullable)
  - `status` (`active` / `disabled`)
- `sessions`
  - `id` UUID PK (cookie value)
  - `user_id` FK -> users
  - `created_at`, `expires_at`
  - `revoked_at` nullable
  - optional audit: `ip`, `user_agent`

Constraints and indexes:
- Unique indexes on `users.email` (if used) and `users.username`.
- Index `sessions.user_id`, `sessions.expires_at`.

Decisions to make (write them into the migration and document here):
- Where UUIDs are generated:
  - Prefer `gen_random_uuid()` (requires `pgcrypto`) OR `uuid-ossp`.
- Email policy:
  - MVP option: allow null email, require unique username.

Done when:
- Migration applies cleanly to an empty database.
- Migration is idempotent only via your migration tool (don’t hand-roll “IF EXISTS” everywhere unless you have a strong reason).

References:
- `internal/store/postgres/users.go` and `internal/store/postgres/sessions.go` (created in **Step 4**) must match these columns exactly.

### 3.2 Add “friends + matches” tables later (do not block auth)
Tasks:
- Defer `friendships`, `matches`, `match_players` until after auth endpoints exist.

Done when:
- You can register/login/logout/me end-to-end before adding non-auth migrations.

References:
- Friends and matches start in **Phase 6**.

---

## Phase 4 — Postgres connection + store layer (auth only)

### 4.1 Create Postgres pool initialization
Create `internal/store/postgres/db.go`.

Tasks:
- Implement `Open(ctx, dsn) (*pgxpool.Pool, error)`
- Configure reasonable pool sizing defaults (safe MVP: use pgx defaults unless you know your load).
- Add a `Ping` method used by `/healthz` once DB is integrated (**Step 2.2**).

Done when:
- Server starts, connects to DB, and `/healthz` can optionally verify DB connectivity.

References:
- `cmd/server/main.go` owns process startup and passes pool to services/handlers (**Step 6/7/8**).

### 4.2 Implement `users` store
Create `internal/store/postgres/users.go`.

Tasks:
- Create functions like:
  - `CreateUser(ctx, email, username, passwordHash) (User, error)`
  - `GetUserByID(ctx, id) (User, error)`
  - `GetUserByLogin(ctx, login) (User, error)` where login can be username or email
  - `SetLastLogin(ctx, userID, when)`
- Map DB uniqueness violations to domain errors (see **Step 5.2**).

Done when:
- You can create a user and retrieve it by username/email in a small local test harness (or endpoint later).

References:
- `internal/service/usersvc.go` uses these functions in **Step 8**.

### 4.3 Implement `sessions` store
Create `internal/store/postgres/sessions.go`.

Tasks:
- Create functions like:
  - `CreateSession(ctx, userID, expiresAt, ip, userAgent) (sessionID, error)`
  - `GetSession(ctx, sessionID) (Session, error)` (must reject revoked/expired)
  - `RevokeSession(ctx, sessionID, when)`
  - `RevokeUserSessions(ctx, userID, when)` (optional: “logout everywhere” later)

Done when:
- A session can be created, fetched, and revoked, with correct expiry semantics.

References:
- `internal/httpapi/middleware.go` auth middleware calls `GetSession` (**Step 7.3**).
- `POST /v1/auth/logout` revokes session (**Step 8.3**).

---

## Phase 5 — Domain types, errors, and response contract

### 5.1 Define domain models used across layers
Create `internal/domain/models.go`.

Tasks:
- Define `User`, `Session` structs with only the fields you want to expose/use.
- Make sure you separate:
  - internal DB fields (e.g., `password_hash`) from
  - API response fields (never return password hashes).

Done when:
- Store layer returns domain models (or internal equivalents) without leaking sensitive fields.

References:
- Used by store (**Phase 4**), service (**Phase 7/8**), and handlers (**Phase 8**).

### 5.2 Define canonical errors
Create `internal/domain/errors.go`.

Tasks:
- Define errors like:
  - `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`
  - `ErrUsernameTaken`, `ErrEmailTaken`
  - `ErrInvalidCredentials`
  - `ErrUserDisabled`
  - `ErrValidation`
- Decide how errors map to HTTP status codes and error codes.

Done when:
- HTTP handlers can reliably map domain errors to status codes without string matching.

References:
- HTTP error responses in **Step 6.3** and auth handlers in **Phase 8**.

### 5.3 Define API error response format
Implement in `internal/httpapi/` (file name up to you, e.g. `responses.go`).

Tasks:
- Standardize error JSON shape, for example:
  - `{ "error": { "code": "...", "message": "..."} }`
- Ensure sensitive messages aren’t leaked (e.g., “user not found” vs “wrong password”).

Done when:
- All endpoints return consistent JSON errors for invalid input and auth failures.

References:
- Used by every handler in **Phase 8** and later features.

---

## Phase 6 — HTTP API skeleton (router + middleware)

### 6.1 Router wiring
Create `internal/httpapi/router.go`.

Tasks:
- Create `NewRouter(deps...) http.Handler` that mounts:
  - `GET /healthz`
  - `POST /v1/auth/register`
  - `POST /v1/auth/login`
  - `POST /v1/auth/logout`
  - `GET /v1/users/me`
- Keep handlers in separate files: `handlers_auth.go`, `handlers_users.go`.

Done when:
- `cmd/server/main.go` serves the router on `Config.Addr`.

References:
- Handlers are implemented in **Phase 7**.

### 6.2 Logging + request IDs + panic recovery
Create `internal/httpapi/middleware.go`.

Tasks:
- Add:
  - request ID middleware
  - structured logging (Go `log/slog` is fine)
  - panic recovery (return `500` JSON error; do not expose stack traces in prod)
- Log basic request metadata:
  - method, path, status, duration, request_id

Done when:
- Every request emits a single structured log line in dev.

References:
- Helps debug auth flows in **Phase 7** without guessing.

### 6.3 Auth middleware (session cookie → user context)
Tasks:
- Read `mtg_session` cookie.
- Look up session via `internal/store/postgres/sessions.go` (**Step 4.3**).
- Load user via `internal/store/postgres/users.go` (**Step 4.2**).
- Attach user to request context (e.g., `context.WithValue` or a typed context helper).
- If missing/invalid, return `401` for protected endpoints, but allow public endpoints.

Done when:
- `GET /v1/users/me` returns `401` if no cookie, `200` if valid cookie.

References:
- `GET /v1/users/me` handler in **Step 7.4** depends on this.

---

## Phase 7 — Auth internals (passwords + session cookie)

### 7.1 Implement password hashing (Argon2id)
Create `internal/auth/passwords.go`.

Tasks:
- Implement:
  - `HashPassword(plaintext) (hash string, err error)`
  - `VerifyPassword(hash, plaintext) (bool, err error)`
- Store hashes using a self-describing format (include parameters + salt).
- Enforce a minimum password length in the HTTP layer (see **Step 8.1**).

Done when:
- Unit tests confirm:
  - hashing is non-deterministic (salted)
  - verify succeeds for correct password, fails otherwise

References:
- Register handler hashes passwords (**Step 8.1**).
- Login handler verifies passwords (**Step 8.2**).

### 7.2 Session creation + cookie writing
Create `internal/auth/sessions.go` (or keep in service if you prefer).

Tasks:
- Decide what the cookie stores:
  - MVP: session UUID only (random, opaque).
- Implement helper:
  - `SetSessionCookie(w, sessionID, ttl, isSecure)`
  - `ClearSessionCookie(w)`
- Ensure cookie security settings are consistent with your deployment:
  - In prod behind Nginx TLS: `Secure=true`
  - In local dev over HTTP: allow `Secure=false` (guarded by `APP_ENV`)

Done when:
- Register/login responses set the cookie.
- Logout clears cookie and revokes server-side session.

References:
- Used by auth handlers in **Phase 8**.

### 7.3 Add basic rate limiting (MVP-friendly)
Tasks:
- Add rate limiting to `POST /v1/auth/login` at minimum (per IP and per login identifier).
- Keep it in memory initially (single server), with clear constants.

Done when:
- Repeated failed logins get throttled and return `429` consistently.

References:
- Implemented as middleware or inside the login handler (**Step 8.2**).

---

## Phase 8 — Auth endpoints (the first “vertical slice”)

### 8.1 `POST /v1/auth/register`
Create `internal/httpapi/handlers_auth.go`.

Request:
```json
{ "email": "optional@example.com", "username": "name", "password": "..." }
```

Tasks:
- Validate input:
  - normalize username (trim)
  - enforce password length (e.g., >= 10–12)
  - enforce username rules (allowed chars, min/max length)
  - optionally validate email format
- Hash password using `internal/auth/passwords.go` (**Step 7.1**).
- Insert user using `internal/store/postgres/users.go` (**Step 4.2**).
- Create session using `internal/store/postgres/sessions.go` (**Step 4.3**).
- Set cookie using `internal/auth/sessions.go` (**Step 7.2**).
- Return user profile JSON (never password hash).

Done when:
- `curl` register returns `200/201` and includes `Set-Cookie: mtg_session=...`.
- Re-register with same username returns `409` with a stable error code.

References:
- Error mapping uses `internal/domain/errors.go` (**Step 5.2**) and response format (**Step 5.3**).

### 8.2 `POST /v1/auth/login`
Request:
```json
{ "login": "emailOrUsername", "password": "..." }
```

Tasks:
- Lookup user by login (`GetUserByLogin`) (**Step 4.2**).
- If user is disabled, return `403` (don’t create a session).
- Verify password using `internal/auth/passwords.go` (**Step 7.1**).
- Create session + set cookie (**Steps 4.3, 7.2**).
- Update `last_login_at` (**Step 4.2**).
- Apply rate limiting (**Step 7.3**).

Done when:
- Correct credentials set cookie and return `200`.
- Wrong credentials return `401` with generic message.
- Excessive attempts return `429`.

### 8.3 `POST /v1/auth/logout`
Tasks:
- Require auth (session cookie).
- Revoke current session in DB (`RevokeSession`) (**Step 4.3**).
- Clear cookie (**Step 7.2**).

Done when:
- After logout, `GET /v1/users/me` returns `401`.

### 8.4 `GET /v1/users/me`
Tasks:
- Require auth middleware (**Step 6.3**).
- Return current user’s public profile JSON.

Done when:
- With a valid cookie, returns `200` and the correct user ID/username.

---

## Phase 9 — Local dev workflow + verification

### 9.1 Add a “how to run locally” section (README or docs)
Tasks:
- Document required env vars and example values.
- Document migration commands.
- Document example curl flows:
  - register → me → logout → me (401)

Done when:
- A new dev can run the service locally in <10 minutes.

References:
- Must match config contract in **Step 2.1** and migrations in **Phase 3**.

### 9.2 Add tests where they give leverage (don’t overbuild)
Tasks:
- Unit tests:
  - password hashing/verification (**Step 7.1**)
  - session cookie helpers (**Step 7.2**)
- Optional integration tests (if you’re comfortable):
  - spin up a test DB and test register/login/me flows

Done when:
- `go test ./...` is green.

---

## Phase 10 — Deployment foundations (after auth works)

### 10.1 systemd unit + env file (single-server deployment)
Tasks:
- Create an example unit file and env file template (don’t include secrets in git).
- Ensure the service binds to `127.0.0.1:8080` (private) and is proxied by Nginx for `https://mtgleader.xyz`.

Done when:
- Server starts on boot and is reachable via Nginx reverse proxy.

References:
- Cookie `Secure` behavior depends on TLS termination (see **Step 7.2**).

### 10.2 Nginx reverse proxy
Tasks:
- Add minimal Nginx config example (server block).
- Forward `X-Forwarded-Proto` and `X-Forwarded-For`.

Done when:
- HTTPS works end-to-end and cookies are set correctly on `https://mtgleader.xyz`.

---

## Phase 11 — Next features (once auth is stable)

### 11.1 Friendships (request/accept/list)
Tasks:
- Add `friendships` migration.
- Implement store/service/handlers:
  - list friends + pending
  - create request
  - accept/decline

Critical references:
- Matches in **Phase 11.2** depend on “accepted friend” checks in the service layer.

### 11.2 Matches (create/list) + “wins between friends”
Tasks:
- Add `matches` + `match_players` migrations.
- Enforce in `matchsvc`:
  - creator authenticated
  - >=2 players
  - all players are creator or accepted friends (MVP invariant)
  - winner constraints

### 11.3 Stats (summary + head-to-head)
Tasks:
- Implement query endpoints and SQL optimized with indexes.

---

## Suggested next step (what we do next together)

Start Phase 1 + Phase 2 (layout + config + `/healthz`), then Phase 3/4 (migrations + DB pool), then Phase 8 (auth endpoints). That gets your Android app unblocked quickly.
