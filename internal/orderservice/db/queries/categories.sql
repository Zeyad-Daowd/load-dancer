-- name: InsertCategory :one
INSERT INTO categories (name) VALUES ($1) RETURNING *;

-- name: AddProductCategory :one
INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2) RETURNING *;