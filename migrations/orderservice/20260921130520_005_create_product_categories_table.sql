-- +goose Up
CREATE TABLE product_categories (
    product_id INTEGER REFERENCES products(id),
    category_id INTEGER REFERENCES categories(id),
    PRIMARY KEY (product_id, category_id)
);
-- +goose Down
DROP TABLE product_categories;