-- name: CreateSession :one
INSERT INTO sessions (id, title, message_count, created_at, updated_at)
VALUES (?, ?, 0, ?, ?)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = ?;

-- name: ListSessions :many
SELECT * FROM sessions ORDER BY updated_at DESC;

-- name: UpdateSessionTitle :exec
UPDATE sessions SET title = ?, updated_at = ? WHERE id = ?;

-- name: UpdateSessionMessageCount :exec
UPDATE sessions SET message_count = message_count + 1, updated_at = ? WHERE id = ?;

-- name: DecrementSessionMessageCount :exec
UPDATE sessions SET message_count = CASE WHEN message_count > 0 THEN message_count - 1 ELSE 0 END, updated_at = ? WHERE id = ?;

-- name: SetSessionSummary :exec
UPDATE sessions SET summary_message_id = ?, updated_at = ? WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: SearchSessions :many
SELECT * FROM sessions
WHERE title LIKE '%' || ? || '%'
ORDER BY updated_at DESC;

-- name: ListSessionsWithPreview :many
SELECT
    s.id,
    s.title,
    s.message_count,
    s.summary_message_id,
    s.created_at,
    s.updated_at,
    COALESCE((SELECT m.parts FROM messages m WHERE m.session_id = s.id AND m.role = 'user' ORDER BY m.created_at ASC LIMIT 1), '') as first_message
FROM sessions s
ORDER BY s.updated_at DESC;

-- name: SearchSessionsWithPreview :many
SELECT
    s.id,
    s.title,
    s.message_count,
    s.summary_message_id,
    s.created_at,
    s.updated_at,
    COALESCE((SELECT m.parts FROM messages m WHERE m.session_id = s.id AND m.role = 'user' ORDER BY m.created_at ASC LIMIT 1), '') as first_message
FROM sessions s
WHERE s.title LIKE '%' || ? || '%'
ORDER BY s.updated_at DESC;
