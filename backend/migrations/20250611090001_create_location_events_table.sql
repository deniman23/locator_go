CREATE TABLE location_events (
                                 id SERIAL PRIMARY KEY,
                                 user_id INTEGER NOT NULL,
                                 checkpoint_id INTEGER NOT NULL,
                                 latitude NUMERIC(10, 6) NOT NULL,
                                 longitude NUMERIC(10, 6) NOT NULL,
                                 occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                 payload JSONB,  -- дополнительные данные, если требуются
                                 processed BOOLEAN NOT NULL DEFAULT FALSE
);