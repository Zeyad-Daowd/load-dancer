-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: AddProduct :one
INSERT INTO products (name, price_cents, description) VALUES ($1, $2, $3) RETURNING *;

-- name: GetProducts :many
SELECT * FROM products;

-- name: InsertInventory :one
INSERT INTO inventory (product_id, stock_count) VALUES ($1, $2) RETURNING *;

-- name: IncreaseProductStock :one
UPDATE inventory SET stock_count = inventory.stock_count + $2 WHERE product_id = $1 RETURNING *;

-- name: GetStock :one
SELECT stock_count from inventory WHERE product_id = $1;