-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
  ALTER COLUMN created_at SET DEFAULT date_trunc('milliseconds', now()),
  ALTER COLUMN updated_at SET DEFAULT date_trunc('milliseconds', now());

UPDATE users
SET
  created_at = date_trunc('milliseconds', created_at),
  updated_at = date_trunc('milliseconds', updated_at),
  avatar_updated_at = CASE
    WHEN avatar_updated_at IS NULL THEN NULL
    ELSE date_trunc('milliseconds', avatar_updated_at)
  END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
  ALTER COLUMN created_at SET DEFAULT now(),
  ALTER COLUMN updated_at SET DEFAULT now();

-- +goose StatementEnd
