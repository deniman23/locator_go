-- +goose Up
CREATE TABLE IF NOT EXISTS location_requests (
                                                 id VARCHAR(36) PRIMARY KEY,
    user_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
                               );

CREATE INDEX IF NOT EXISTS idx_location_requests_user_status ON location_requests (user_id, status);
CREATE INDEX IF NOT EXISTS idx_location_requests_status_created ON location_requests (status, created_at);

-- +goose Down
DROP TABLE IF EXISTS location_requests;
