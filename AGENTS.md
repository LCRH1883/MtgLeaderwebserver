# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server/main.go` is the API entry point.
- `internal/` holds application packages: `auth`, `httpapi` (handlers/middleware), `service` (business logic), `store/postgres` (DB access), `domain` (models/errors), `config`, and `adminui`.
- `migrations/` contains Goose-compatible SQL migrations.
- `scripts/` provides repo-local tooling (`scripts/go`, `scripts/migrate`).
- `deploy/` includes systemd and nginx examples for single-server deploys.

## Build, Test, and Development Commands
- `scripts/go test ./...` runs all Go tests using the repo-local toolchain wrapper.
- `scripts/go run ./cmd/server` starts the API server.
- `scripts/migrate up` applies DB migrations (requires `APP_DB_DSN`).
- `scripts/migrate status` shows the current migration state.

## Coding Style & Naming Conventions
- Follow standard Go style and run `gofmt` on all `.go` files.
- Use tabs for indentation (Go default).
- Package names are short, lower-case, and no underscores.
- Test files use the `*_test.go` suffix; keep test names descriptive (e.g., `TestSessionsCreate`).

## Testing Guidelines
- Framework: Go `testing` package.
- Run the full suite with `scripts/go test ./...` before opening a PR.
- Add tests alongside the package being changed (e.g., `internal/auth` tests live in `internal/auth`).

## Commit & Pull Request Guidelines
- Commit messages follow short, imperative summaries (e.g., “Adds admin UI”, “Implements match stats”).
- PRs should include a concise description, testing notes, and linked issues.
- Include screenshots or GIFs for admin UI changes in `internal/adminui`.
- Call out schema changes and include migration names when applicable.

## Configuration & Security Tips
- Use `.env.example` as a starting point and set required env vars: `APP_ADDR`, `APP_DB_DSN`, `APP_COOKIE_SECRET`.
- Admin UI access is gated by `APP_ADMIN_EMAILS`; keep this list tight in shared environments.

## Postgres Setup (Required)
- Create a dedicated DB role and database for this app:
  - `sudo -u postgres psql -d postgres -c "CREATE ROLE mtgleader WITH LOGIN PASSWORD '8yr3jctj';"`
  - `sudo -u postgres createdb -O mtgleader mtg`
- If they already exist, reset ownership/password:
  - `sudo -u postgres psql -d postgres -c "ALTER ROLE mtgleader WITH PASSWORD '8yr3jctj';"`
  - `sudo -u postgres psql -d postgres -c "ALTER DATABASE mtg OWNER TO mtgleader;"`
- DSN (local dev): `postgres://mtgleader:8yr3jctj@127.0.0.1:5432/mtg?sslmode=disable`
- DSN (prod): `postgres://mtgleader:8yr3jctj@127.0.0.1:5432/mtg?sslmode=require`
- Example prod env file: `deploy/mtgleaderwebserver.env.example`
- If you access over HTTP, set `APP_PUBLIC_URL` to `http://127.0.0.1:8080` or login cookies will not persist.

## Reverse Proxy (Required for HTTPS -> HTTP)
- If using Nginx Proxy Manager or nginx with HTTPS termination, set `APP_PUBLIC_URL` to the external HTTPS URL.
- Forward `Host` and `X-Forwarded-Proto` headers so cookies are issued for the correct domain.

## Admin Email and Password Resets
- Configure SMTP in the admin UI at `https://mtgleader.xyz/admin/email`.
- Password reset links are sent from configured SMTP aliases and handled at `/app/reset`.

## Canonical URLs
- Production base URL: `https://mtgleader.xyz` (use full URLs in docs and responses).
- User UI: `https://mtgleader.xyz/app/login` and `https://mtgleader.xyz/app/register`.
- Admin UI: `https://mtgleader.xyz/admin/login`.
