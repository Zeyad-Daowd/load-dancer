-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    role VARCHAR(20) NOT NULL,
    CONSTRAINT role_check
        CHECK (role IN ('admin', 'user'))
);
-- +goose Down
DROP TABLE users;
