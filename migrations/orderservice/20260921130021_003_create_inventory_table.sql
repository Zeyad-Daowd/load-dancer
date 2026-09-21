-- +goose Up
CREATE TABLE inventory (
    product_id INTEGER PRIMARY KEY REFERENCES products(id),
    stock_count INTEGER NOT NULL,
    CONSTRAINT stock_count_check
        CHECK (stock_count >= 0)
);
-- +goose Down
DROP TABLE inventory;