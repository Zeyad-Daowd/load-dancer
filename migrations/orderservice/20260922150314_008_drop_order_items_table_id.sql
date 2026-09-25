-- +goose Up
ALTER TABLE order_items
    DROP CONSTRAINT order_items_pkey,
    DROP COLUMN id,
    DROP CONSTRAINT order_product_unique,
    ADD CONSTRAINT order_items_pkey
        PRIMARY KEY (order_id, product_id);

-- +goose Down
ALTER TABLE order_items
    DROP CONSTRAINT order_items_pkey,
    ADD COLUMN id SERIAL PRIMARY KEY,
    ADD CONSTRAINT order_product_unique
        UNIQUE (order_id, product_id);
