-- +goose Up
-- +goose StatementBegin

ALTER TABLE friendships
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT date_trunc('milliseconds', now()),
  ALTER COLUMN created_at SET DEFAULT date_trunc('milliseconds', now());

UPDATE friendships
SET
  created_at = date_trunc('milliseconds', created_at),
  responded_at = CASE
    WHEN responded_at IS NULL THEN NULL
    ELSE date_trunc('milliseconds', responded_at)
  END,
  updated_at = date_trunc('milliseconds', COALESCE(responded_at, created_at));

CREATE INDEX friendships_updated_at_idx ON friendships (updated_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS friendships_updated_at_idx;

ALTER TABLE friendships
  ALTER COLUMN created_at SET DEFAULT now(),
  DROP COLUMN updated_at;

-- +goose StatementEnd
