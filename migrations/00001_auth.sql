-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NULL,
  username TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_login_at TIMESTAMPTZ NULL,
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX users_username_uq ON users (username);
CREATE UNIQUE INDEX users_email_uq ON users (email) WHERE email IS NOT NULL;

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ NULL,
  ip TEXT NULL,
  user_agent TEXT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE sessions;
DROP TABLE users;

-- +goose StatementEnd
