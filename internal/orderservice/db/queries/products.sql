-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: AddProduct :one
INSERT INTO products (name, price_cents, description) VALUES ($1, $2, $3) RETURNING *;

-- name: GetProducts :many
SELECT * FROM products;