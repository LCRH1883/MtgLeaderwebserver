# MtgLeaderwebserver

Go backend for an MTG app (auth → friends → matches → stats).

## Local dev

### Prereqs
- Go (see `scripts/go` for the repo-local toolchain wrapper)
- Postgres
- SQL migrations are `goose`-compatible (`migrations/`)

### Run
Set env vars (minimum for auth):
- `APP_ADDR` (default: `127.0.0.1:8080`)
- `APP_DB_DSN` (Postgres DSN)
- `APP_COOKIE_SECRET` (recommend 32+ bytes)

Then:
```bash
cp .env.example .env
scripts/go test ./...
scripts/go run ./cmd/server
```

### Endpoints
- `GET /healthz` → `ok`
- `POST /v1/auth/register`
- `POST /v1/auth/login`
- `POST /v1/auth/logout`
- `GET /v1/users/me`
