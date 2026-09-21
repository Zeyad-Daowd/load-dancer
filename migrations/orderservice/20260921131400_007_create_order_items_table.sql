-- +goose Up
CREATE TABLE order_items (
    ID BIGSERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id),
    order_id BIGINT NOT NULL REFERENCES orders(id),
    quantity INTEGER NOT NULL,
    unit_price_cents INTEGER NOT NULL,
    CONSTRAINT quantity_check
        CHECK (quantity > 0),
    CONSTRAINT unit_price_cents_check
        CHECK (unit_price_cents >= 0)
);
-- +goose Down
DROP TABLE order_items;