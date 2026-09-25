-- +goose Up
ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_idempotency_key_key,
    ADD CONSTRAINT orders_user_idempotency_unique UNIQUE (user_id, idempotency_key);

-- +goose Down
ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_user_idempotency_unique,
    ADD CONSTRAINT orders_idempotency_key_key UNIQUE (idempotency_key);