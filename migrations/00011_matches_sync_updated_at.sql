-- +goose Up
-- +goose StatementBegin

ALTER TABLE matches
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT date_trunc('milliseconds', now()),
  ALTER COLUMN created_at SET DEFAULT date_trunc('milliseconds', now());

UPDATE matches
SET
  created_at = date_trunc('milliseconds', created_at),
  updated_at = date_trunc('milliseconds', created_at);

DROP INDEX IF EXISTS matches_client_ref_uq;
CREATE UNIQUE INDEX matches_client_ref_uq ON matches (created_by, client_ref);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS matches_client_ref_uq;
CREATE UNIQUE INDEX matches_client_ref_uq ON matches (client_ref);

ALTER TABLE matches
  ALTER COLUMN created_at SET DEFAULT now(),
  DROP COLUMN updated_at;

-- +goose StatementEnd
