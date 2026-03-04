-- name: CreatePingHistory :one
INSERT INTO ping_history (payload)
VALUES ($1)
RETURNING id, payload, created_at;

-- name: ListRecentPingHistory :many
SELECT id, payload, created_at
FROM ping_history
ORDER BY created_at DESC, id DESC
LIMIT $1;
