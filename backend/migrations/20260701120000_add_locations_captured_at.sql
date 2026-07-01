-- +goose Up
ALTER TABLE locations
    ADD COLUMN IF NOT EXISTS captured_at TIMESTAMP WITHOUT TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_locations_user_captured_at
    ON locations (user_id, captured_at);

-- +goose Down
DROP INDEX IF EXISTS idx_locations_user_captured_at;
ALTER TABLE locations DROP COLUMN IF EXISTS captured_at;
