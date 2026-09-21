-- name: AddOrder :one
INSERT INTO orders (user_id, idempotency_key, total_cents) VALUES ($1, $2, $3) RETURNING *;

-- name: AddProductOrderItem :one
INSERT INTO order_items (product_id, order_id, quantity, unit_price_cents) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: EditProductQuantityOrderItem :one
UPDATE order_items SET quantity = $1 WHERE product_id = $2 and order_id = $3 RETURNING *;

-- name: GetCheckoutItems :many
SELECT i.product_id, i.stock_count as available_stock, oi.quantity as requested_quantity FROM inventory i
JOIN order_items oi ON i.product_id = oi.product_id
WHERE oi.order_id = $1 ORDER BY i.product_id FOR UPDATE of i;

-- name: UpdateInventoryStock :one
UPDATE inventory SET stock_count = stock_count - $2 WHERE product_id = $1 RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders SET status = 'pending' WHERE id = $1 and status = 'started' RETURNING *;
