# MtgLeaderwebserver

Go backend for an MTG app (auth → friends → matches → stats).

## Local dev

### Prereqs
- Go (see `scripts/go` for the repo-local toolchain wrapper)
- Postgres
- SQL migrations are `goose`-compatible (`migrations/`) and runnable via `scripts/migrate`

### Run
Set env vars (minimum for auth):
- `APP_ADDR` (default: `127.0.0.1:8080`)
- `APP_DB_DSN` (Postgres DSN)
- `APP_COOKIE_SECRET` (recommend 32+ bytes)
- `email` is required for all users (login is email/password)

Then:
```bash
cp .env.example .env
export $(grep -v '^#' .env | xargs)  # or set env vars in GoLand Run config

scripts/migrate up
scripts/go test ./...
scripts/go run ./cmd/server
```

### Endpoints
- `GET /healthz` → `ok`
- `POST /v1/auth/register`
- `POST /v1/auth/login` (email + password)
- `POST /v1/auth/logout`
- `GET /v1/users/me`
- `GET /v1/friends`
- `POST /v1/friends/requests`
- `POST /v1/friends/requests/{id}/accept`
- `POST /v1/friends/requests/{id}/decline`
- `POST /v1/matches`
- `GET /v1/matches`
- `GET /v1/stats/summary`
- `GET /v1/stats/head-to-head/{id}`

### Example curl flow
```bash
curl -i -sS -X POST http://127.0.0.1:8080/v1/auth/register \
  -H 'content-type: application/json' \
  -d '{"email":"alice@example.com","username":"alice","password":"correct horse battery staple"}' \
  -c cookies.txt

curl -i -sS http://127.0.0.1:8080/v1/users/me -b cookies.txt
curl -i -sS http://127.0.0.1:8080/v1/stats/summary -b cookies.txt
```

## Deployment (single server)
- systemd example: `deploy/systemd/mtg-leaderwebserver.service.example`
- nginx example: `deploy/nginx/mtg-leaderwebserver.conf.example`
