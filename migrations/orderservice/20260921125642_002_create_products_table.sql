-- +goose Up
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_cents int NOT NULL,
    description TEXT,
    CONSTRAINT price_check
        CHECK (price_cents >= 0)
);
-- +goose Down
DROP TABLE products;