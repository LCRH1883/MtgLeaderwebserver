-- +goose Up
-- +goose StatementBegin

CREATE TABLE smtp_settings (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  host TEXT NOT NULL,
  port INTEGER NOT NULL CHECK (port > 0 AND port < 65536),
  username TEXT NOT NULL,
  password TEXT NOT NULL,
  tls_mode TEXT NOT NULL DEFAULT 'starttls' CHECK (tls_mode IN ('none', 'starttls', 'tls')),
  from_name TEXT NOT NULL,
  from_email TEXT NOT NULL,
  alias_emails TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  sent_to_email TEXT NOT NULL,
  created_by_admin_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ NULL
);

CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);
CREATE INDEX password_reset_tokens_token_hash_idx ON password_reset_tokens (token_hash);
CREATE INDEX password_reset_tokens_expires_idx ON password_reset_tokens (expires_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE password_reset_tokens;
DROP TABLE smtp_settings;

-- +goose StatementEnd
