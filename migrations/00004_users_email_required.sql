-- +goose Up
-- +goose StatementBegin

-- Normalize existing values.
UPDATE users SET email = lower(email) WHERE email IS NOT NULL;

-- Drop old partial unique index (from 00001) and replace with a strict unique index.
DROP INDEX IF EXISTS users_email_uq;

ALTER TABLE users
  ALTER COLUMN email SET NOT NULL;

CREATE UNIQUE INDEX users_email_uq ON users (email);

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_email_lower_chk;

ALTER TABLE users
  ADD CONSTRAINT users_email_lower_chk CHECK (email = lower(email));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_email_lower_chk;

DROP INDEX IF EXISTS users_email_uq;

ALTER TABLE users
  ALTER COLUMN email DROP NOT NULL;

CREATE UNIQUE INDEX users_email_uq ON users (email) WHERE email IS NOT NULL;

-- +goose StatementEnd

