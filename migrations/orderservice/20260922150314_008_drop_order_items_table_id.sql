-- +goose Up
ALTER TABLE order_items
    DROP CONSTRAINT order_items_pkey;

ALTER TABLE order_items
    DROP COLUMN id;

ALTER TABLE order_items
    DROP CONSTRAINT order_product_unique;

ALTER TABLE order_items
    ADD CONSTRAINT order_items_pkey
        PRIMARY KEY (order_id, product_id);

-- +goose Down
ALTER TABLE order_items
    DROP CONSTRAINT order_items_pkey;

ALTER TABLE order_items
    ADD COLUMN id SERIAL PRIMARY KEY;

ALTER TABLE order_items
    ADD CONSTRAINT order_product_unique
        UNIQUE (order_id, product_id);
