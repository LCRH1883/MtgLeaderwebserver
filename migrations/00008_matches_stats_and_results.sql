-- +goose Up
-- +goose StatementBegin

ALTER TABLE matches
  ADD COLUMN format TEXT NOT NULL DEFAULT 'commander',
  ADD COLUMN total_duration_seconds INT NOT NULL DEFAULT 0,
  ADD COLUMN turn_count INT NOT NULL DEFAULT 0,
  ADD COLUMN client_ref TEXT NULL;

ALTER TABLE matches
  ADD CONSTRAINT matches_format_chk CHECK (format IN ('commander', 'brawl', 'standard', 'modern'));

CREATE UNIQUE INDEX matches_client_ref_uq ON matches (client_ref);
CREATE INDEX matches_format_played_at_idx ON matches (format, played_at);

CREATE TABLE match_player_results (
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rank INT NOT NULL CHECK (rank >= 1),
  elimination_turn INT NULL CHECK (elimination_turn >= 0),
  elimination_batch INT NULL CHECK (elimination_batch >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (match_id, user_id)
);

CREATE INDEX match_player_results_user_id_idx ON match_player_results (user_id);
CREATE INDEX match_player_results_match_rank_idx ON match_player_results (match_id, rank);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS match_player_results_match_rank_idx;
DROP INDEX IF EXISTS match_player_results_user_id_idx;
DROP TABLE match_player_results;

DROP INDEX IF EXISTS matches_format_played_at_idx;
DROP INDEX IF EXISTS matches_client_ref_uq;
ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_format_chk;
ALTER TABLE matches
  DROP COLUMN client_ref,
  DROP COLUMN turn_count,
  DROP COLUMN total_duration_seconds,
  DROP COLUMN format;

-- +goose StatementEnd
