-- +goose Up
CREATE TABLE IF NOT EXISTS locations (
                                         id SERIAL PRIMARY KEY,
                                         user_id INTEGER NOT NULL,
                                         latitude DOUBLE PRECISION NOT NULL,
                                         longitude DOUBLE PRECISION NOT NULL,
                                         created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                         updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                         CONSTRAINT fk_location_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
    );

-- +goose Down
DROP TABLE IF EXISTS locations;