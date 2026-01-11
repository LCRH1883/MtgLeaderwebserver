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
- `APP_AVATAR_DIR` (optional; defaults to `data/avatars`)
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

### First Run (No Prompts)
- Ensure `.env` exists with `APP_DB_DSN` set (use `.env.example` as a template).
- `scripts/migrate up` reads `.env` automatically, so no extra export is required.

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
  - `GET /admin/email` (SMTP settings)
  - `POST /admin/users/reset` (send reset link / update email)
- User password reset:
  - `GET /app/reset`
  - `POST /app/reset`
- User profile:
  - `GET /app/profile`
  - `POST /app/profile`
  - `POST /app/profile/avatar`

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
  - If `APP_AVATAR_DIR` points outside the working directory, update `ReadWritePaths` in the systemd unit and ensure the folder is writable by the service user.

### Restarting the server
If you installed this as a systemd service, restart it with:
```bash
sudo systemctl restart mtgleaderwebserver
sudo systemctl status mtgleaderwebserver --no-pager
sudo journalctl -u mtgleaderwebserver -f
```

If you’re not sure what the unit is called:
```bash
sudo systemctl list-unit-files --type=service | grep -i mtg
```

If you run it manually (no systemd), stop the PID from `server.manual.pid` and start it again.

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

### Admin email settings
Configure SMTP in the admin UI at `https://mtgleader.xyz/admin/email`. Password reset links use `APP_PUBLIC_URL` to build the reset URL and are sent from the selected alias.
