-- +goose Up
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    idempotency_key TEXT NOT NULL UNIQUE,
    total_cents INTEGER NOT NULL,
    CONSTRAINT status_check
        CHECK (status IN ('started', 'pending', 'completed', 'cancelled')),
    CONSTRAINT total_cents_check
        CHECK (total_cents >= 0)
);
-- +goose Down
DROP TABLE orders;