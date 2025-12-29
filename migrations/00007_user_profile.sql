-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
  ADD COLUMN display_name TEXT NULL,
  ADD COLUMN avatar_path TEXT NULL,
  ADD COLUMN avatar_updated_at TIMESTAMPTZ NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
  DROP COLUMN avatar_updated_at,
  DROP COLUMN avatar_path,
  DROP COLUMN display_name;

-- +goose StatementEnd
