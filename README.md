# MtgLeaderwebserver

Go backend for an MTG app (auth → friends → matches → stats).

## Production URLs
- Base URL: `https://mtgleader.xyz`
- User login: `https://mtgleader.xyz/app/login`
- User signup: `https://mtgleader.xyz/app/register`
- Admin login: `https://mtgleader.xyz/admin/login`

## Local dev

### Prereqs
- Go (see `scripts/go` for the repo-local toolchain wrapper)
- Postgres
- SQL migrations are `goose`-compatible (`migrations/`) and runnable via `scripts/migrate`
  - Optional admin UI: set `APP_ADMIN_EMAILS` (comma-separated allowlist)
  - Optional admin bootstrap user: set `APP_ADMIN_BOOTSTRAP_EMAIL` + `APP_ADMIN_BOOTSTRAP_PASSWORD` (defaults to `lh@intagri.io` / `admin`)

## Database setup (required)
This app uses a dedicated Postgres role and database:
- Role: `mtgleader`
- Database: `mtg`
- Password: `8yr3jctj`
- Do not reuse other app databases (e.g. `beauty_db`).

Create the role and database:
```bash
sudo -u postgres psql -d postgres -c "CREATE ROLE mtgleader WITH LOGIN PASSWORD '8yr3jctj';"
sudo -u postgres createdb -O mtgleader mtg
```

If they already exist, reset ownership/password:
```bash
sudo -u postgres psql -d postgres -c "ALTER ROLE mtgleader WITH PASSWORD '8yr3jctj';"
sudo -u postgres psql -d postgres -c "ALTER DATABASE mtg OWNER TO mtgleader;"
```

DSN examples:
```bash
# Local dev
APP_DB_DSN=postgres://mtgleader:8yr3jctj@127.0.0.1:5432/mtg?sslmode=disable

# Production (local socket or remote host)
APP_DB_DSN=postgres://mtgleader:8yr3jctj@127.0.0.1:5432/mtg?sslmode=require
```

### Run
Set env vars (minimum for auth):
- `APP_ADDR` (default: `127.0.0.1:8080`)
- `APP_DB_DSN` (Postgres DSN)
- `APP_COOKIE_SECRET` (recommend 32+ bytes)
- `APP_PUBLIC_URL` (prod: `https://mtgleader.xyz`, local dev: `http://127.0.0.1:8080`)
- `email` is required for all users (login is email/password)
- Optional external sign-in:
  - `GOOGLE_WEB_CLIENT_ID`
  - `APPLE_SERVICE_ID`

Then:
```bash
cp .env.example .env
export $(grep -v '^#' .env | xargs)  # or set env vars in GoLand Run config

scripts/migrate up
scripts/go test ./...
scripts/go run ./cmd/server
```

If you access the app over plain HTTP, `APP_PUBLIC_URL` must also be HTTP or login will loop (secure cookies are only sent over HTTPS).

### Endpoints
- `GET /healthz` → `ok`
- `POST /v1/auth/register`
- `POST /v1/auth/login` (email + password)
- `POST /v1/auth/google`
- `POST /v1/auth/apple`
- `POST /v1/auth/logout`
- `GET /v1/users/me`
- `GET /v1/users/search?q=...`
- `GET /v1/friends`
- `POST /v1/friends/requests`
- `POST /v1/friends/requests/{id}/accept`
- `POST /v1/friends/requests/{id}/decline`
- `POST /v1/matches`
- `GET /v1/matches`
- `GET /v1/stats/summary`
- `GET /v1/stats/head-to-head/{id}`
- Admin UI (only when `APP_ADMIN_EMAILS` is set):
  - `GET /admin/`
  - `GET /admin/users`

### Example curl flow
```bash
BASE_URL=https://mtgleader.xyz
# For local dev: BASE_URL=http://127.0.0.1:8080

curl -i -sS -X POST "$BASE_URL/v1/auth/register" \
  -H 'content-type: application/json' \
  -d '{"email":"alice@example.com","username":"alice","password":"correct horse battery staple"}' \
  -c cookies.txt

curl -i -sS "$BASE_URL/v1/users/me" -b cookies.txt
curl -i -sS "$BASE_URL/v1/stats/summary" -b cookies.txt
```

## Deployment (single server)
- systemd example: `deploy/systemd/mtg-leaderwebserver.service.example`
- nginx example: `deploy/nginx/mtg-leaderwebserver.conf.example`
- environment example: `deploy/mtgleaderwebserver.env.example`

### Reverse proxy (Nginx Proxy Manager or nginx)
When proxying HTTPS -> HTTP (e.g. `mtgleader.xyz` -> `http://192.168.2.209:80`), you must:
- Set `APP_PUBLIC_URL=https://mtgleader.xyz` so cookies are marked `Secure`.
- Forward the original host and scheme to the app, or cookies will be set for the wrong host and cause login loops.

Nginx headers (also works in NPM "Custom Nginx Configuration"):
```nginx
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```
