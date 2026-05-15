-- +goose Up
CREATE TABLE IF NOT EXISTS device_commands (
                                               id VARCHAR(36) PRIMARY KEY,
    user_id INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL,
    payload JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    ack_status VARCHAR(50),
    ack_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delivered_at TIMESTAMP WITH TIME ZONE,
    acked_at TIMESTAMP WITH TIME ZONE
                           );

CREATE INDEX IF NOT EXISTS idx_device_cmd_user_status ON device_commands (user_id, status);
CREATE INDEX IF NOT EXISTS idx_device_cmd_status_created ON device_commands (status, created_at);

CREATE TABLE IF NOT EXISTS device_reports (
                                              id SERIAL PRIMARY KEY,
                                              user_id INTEGER NOT NULL,
                                              report JSONB NOT NULL,
                                              issues JSONB,
                                              app_version VARCHAR(50),
    platform VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
                             );

CREATE INDEX IF NOT EXISTS idx_device_reports_user_created ON device_reports (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS device_reports;
DROP TABLE IF EXISTS device_commands;
