-- +goose Up
-- +goose StatementBegin

CREATE TABLE notification_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token TEXT NOT NULL,
  platform TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT date_trunc('milliseconds', now()),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT date_trunc('milliseconds', now())
);

CREATE UNIQUE INDEX notification_tokens_token_uq ON notification_tokens (token);
CREATE INDEX notification_tokens_user_id_idx ON notification_tokens (user_id);
CREATE INDEX notification_tokens_updated_at_idx ON notification_tokens (updated_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE notification_tokens;

-- +goose StatementEnd
