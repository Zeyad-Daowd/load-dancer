-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3) RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;