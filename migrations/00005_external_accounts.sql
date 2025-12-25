-- +goose Up
-- +goose StatementBegin

CREATE TABLE user_external_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  email TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT user_external_accounts_provider_provider_id_uq UNIQUE (provider, provider_id)
);

CREATE INDEX user_external_accounts_user_id_idx ON user_external_accounts (user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE user_external_accounts;

-- +goose StatementEnd
