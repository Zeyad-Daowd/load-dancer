-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: AddProduct :one
INSERT INTO products (name, price_cents, description) VALUES ($1, $2, $3) RETURNING *;

-- name: GetProducts :many
SELECT * FROM products
WHERE (sqlc.narg(category_id)::int IS NULL or EXISTS (select 1 from product_categories where category_id = sqlc.narg(category_id) and product_id = products.id))
AND (sqlc.narg(min_price)::int IS NULL or price_cents >= sqlc.narg(min_price))
AND (sqlc.narg(max_price)::int IS NULL or price_cents <= sqlc.narg(max_price))
LIMIT sqlc.arg(limit_products) OFFSET sqlc.arg(offset_products);

-- name: GetProductsSortedByPriceAsc :many
SELECT * FROM products
WHERE (sqlc.narg(category_id)::int IS NULL or EXISTS (select 1 from product_categories where category_id = sqlc.narg(category_id) and product_id = products.id))
AND (sqlc.narg(min_price)::int IS NULL or price_cents >= sqlc.narg(min_price))
AND (sqlc.narg(max_price)::int IS NULL or price_cents <= sqlc.narg(max_price))
ORDER BY price_cents ASC, id ASC
LIMIT sqlc.arg(limit_products) OFFSET sqlc.arg(offset_products);

-- name: GetProductsSortedByPriceDesc :many
SELECT * FROM products
WHERE (sqlc.narg(category_id)::int IS NULL or EXISTS (select 1 from product_categories where category_id = sqlc.narg(category_id) and product_id = products.id))
AND (sqlc.narg(min_price)::int IS NULL or price_cents >= sqlc.narg(min_price))
AND (sqlc.narg(max_price)::int IS NULL or price_cents <= sqlc.narg(max_price))
ORDER BY price_cents DESC, id ASC
LIMIT sqlc.arg(limit_products) OFFSET sqlc.arg(offset_products);

-- name: InsertInventory :one
INSERT INTO inventory (product_id, stock_count) VALUES ($1, $2) RETURNING *;

-- name: IncreaseProductStock :one
UPDATE inventory SET stock_count = inventory.stock_count + $2 WHERE product_id = $1 RETURNING *;

-- name: GetStock :one
SELECT stock_count from inventory WHERE product_id = $1;