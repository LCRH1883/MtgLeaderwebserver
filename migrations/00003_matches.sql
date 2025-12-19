-- +goose Up
-- +goose StatementBegin

CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  played_at TIMESTAMPTZ NULL,
  winner_id UUID NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX matches_created_by_created_at_idx ON matches (created_by, created_at DESC);
CREATE INDEX matches_winner_id_idx ON matches (winner_id);

CREATE TABLE match_players (
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY (match_id, user_id)
);

CREATE INDEX match_players_user_id_idx ON match_players (user_id);
CREATE INDEX match_players_match_id_idx ON match_players (match_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE match_players;
DROP TABLE matches;

-- +goose StatementEnd
