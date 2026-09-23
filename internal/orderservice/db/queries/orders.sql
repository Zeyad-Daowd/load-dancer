-- name: AddOrder :one
WITH inserted as (
    INSERT INTO orders (user_id, idempotency_key, total_cents) VALUES ($1, $2, $3)
    ON CONFLICT (user_id, idempotency_key)
    DO NOTHING
    RETURNING *
)
SELECT * FROM inserted
UNION ALL
SELECT * FROM orders 
WHERE user_id = $1 AND idempotency_key = $2
AND NOT EXISTS (SELECT 1 FROM inserted);


-- name: AddProductOrderItem :one
INSERT INTO order_items (product_id, order_id, quantity, unit_price_cents)
    SELECT $1, $2, $3, price_cents
    FROM products
    WHERE id = $1
    RETURNING *;

-- name: EditProductQuantityOrderItem :one
UPDATE order_items SET quantity = $1 WHERE product_id = $2 and order_id = $3 RETURNING *;

-- name: RemoveProductQuantityOrderItem :one
DELETE FROM order_items WHERE product_id = $1 AND order_id = $2 RETURNING *;

-- name: GetOrderStatus :one
SELECT status from orders WHERE id = $1;

-- name: GetCheckoutItems :many
SELECT i.product_id, i.stock_count as available_stock, oi.quantity as requested_quantity FROM inventory i
JOIN order_items oi ON i.product_id = oi.product_id
WHERE oi.order_id = $1 ORDER BY i.product_id FOR UPDATE of i;

-- name: DecreaseInventoryStock :one
UPDATE inventory SET stock_count = stock_count - $2 WHERE product_id = $1 RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders SET status = 'pending' WHERE id = $1 and status = 'started' RETURNING *;

-- name: UpdateOrderTotal :exec
UPDATE orders
    SET total_cents = (
        SELECT COALESCE(SUM(quantity * unit_price_cents), 0)
        FROM order_items
        WHERE order_id = $1
    )
    WHERE id = $1;

-- name: GetUserOrders :many
SELECT * FROM orders WHERE user_id = $1;

-- name: GetOrder :many
SELECT o.*, oi.*, p.name 
FROM orders o JOIN order_items oi ON o.id = oi.order_id 
JOIN products p ON oi.product_id = p.id 
WHERE o.id = $1;

-- name: GetOrderItems :many
SELECT oi.*, p.name as product_name FROM order_items oi JOIN products p ON oi.product_id = p.id WHERE order_id = $1;