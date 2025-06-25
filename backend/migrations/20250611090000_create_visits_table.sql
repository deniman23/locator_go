-- +goose Up
CREATE TABLE IF NOT EXISTS visits (
                                      id BIGSERIAL PRIMARY KEY,
                                      user_id INTEGER NOT NULL,
                                      checkpoint_id INTEGER NOT NULL,
                                      start_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
                                      end_at TIMESTAMP WITHOUT TIME ZONE,
                                      duration INTEGER NOT NULL,
                                      created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                      updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                      CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_checkpoint FOREIGN KEY (checkpoint_id) REFERENCES checkpoints (id) ON DELETE CASCADE
    );

-- +goose Down
DROP TABLE IF EXISTS visits;