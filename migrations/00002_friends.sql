-- +goose Up
-- +goose StatementBegin

CREATE TABLE friendships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  addressee_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'declined')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  responded_at TIMESTAMPTZ NULL,
  CHECK (requester_id <> addressee_id)
);

-- Ensure only one row per unordered user pair.
CREATE UNIQUE INDEX friendships_pair_uq ON friendships (
  LEAST(requester_id, addressee_id),
  GREATEST(requester_id, addressee_id)
);

CREATE INDEX friendships_requester_status_idx ON friendships (requester_id, status);
CREATE INDEX friendships_addressee_status_idx ON friendships (addressee_id, status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE friendships;

-- +goose StatementEnd
