-- +goose Up
-- +goose StatementBegin

ALTER TABLE matches
  ADD COLUMN started_at TIMESTAMPTZ NULL,
  ADD COLUMN ended_at TIMESTAMPTZ NULL;

UPDATE matches
SET
  started_at = COALESCE(started_at, played_at),
  ended_at = COALESCE(ended_at, played_at)
WHERE played_at IS NOT NULL;

CREATE TABLE match_participants (
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  seat_index INT NOT NULL CHECK (seat_index >= 0),
  user_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
  guest_name TEXT NULL,
  display_name TEXT NOT NULL DEFAULT '',
  place INT NOT NULL CHECK (place >= 1),
  eliminated_turn_number INT NULL CHECK (eliminated_turn_number >= 0),
  eliminated_during_seat_index INT NULL CHECK (eliminated_during_seat_index >= 0),
  total_turn_time_ms BIGINT NULL CHECK (total_turn_time_ms >= 0),
  turns_taken INT NULL CHECK (turns_taken >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (match_id, seat_index),
  CHECK (
    (user_id IS NOT NULL AND guest_name IS NULL)
    OR (user_id IS NULL AND guest_name IS NOT NULL)
  )
);

CREATE INDEX match_participants_match_id_idx ON match_participants (match_id);
CREATE INDEX match_participants_user_id_idx ON match_participants (user_id);
CREATE INDEX match_participants_guest_name_idx ON match_participants (guest_name);
CREATE UNIQUE INDEX match_participants_match_user_uq ON match_participants (match_id, user_id) WHERE user_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS match_participants_match_user_uq;
DROP INDEX IF EXISTS match_participants_guest_name_idx;
DROP INDEX IF EXISTS match_participants_user_id_idx;
DROP INDEX IF EXISTS match_participants_match_id_idx;
DROP TABLE match_participants;

ALTER TABLE matches
  DROP COLUMN ended_at,
  DROP COLUMN started_at;

-- +goose StatementEnd
