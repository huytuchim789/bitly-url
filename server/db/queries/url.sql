-- name: CreateURL :one
INSERT INTO urls (id, original, short, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetURLByID :one
SELECT * FROM urls WHERE id = $1;

-- name: GetURLByShort :one
SELECT * FROM urls WHERE short = $1;

-- name: ListURLs :many
SELECT * FROM urls ORDER BY created_at DESC;

-- name: DeleteURL :exec
DELETE FROM urls WHERE id = $1;
