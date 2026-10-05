-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN IF NOT EXISTS key_lookup varchar(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS disabled_at timestamptz;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_key_lookup ON users (key_lookup) WHERE key_lookup IS NOT NULL AND key_lookup <> '';

ALTER TABLE device_commands ADD COLUMN IF NOT EXISTS created_by integer;
ALTER TABLE device_commands ADD COLUMN IF NOT EXISTS lease_until timestamptz;

ALTER TABLE checkpoints ADD COLUMN IF NOT EXISTS archived_at timestamptz;

ALTER TABLE locations ADD COLUMN IF NOT EXISTS request_id varchar(36);
ALTER TABLE locations ADD COLUMN IF NOT EXISTS source varchar(20);
-- +goose StatementEnd

-- +goose Down
-- Columns stay. Dropping them would destroy lookup and disable state.
SELECT 1;
