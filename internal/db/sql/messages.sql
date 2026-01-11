-- name: CreateMessage :one
INSERT INTO messages (id, session_id, role, parts, model, provider, is_summary, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMessage :one
SELECT * FROM messages WHERE id = ?;

-- name: GetSessionMessages :many
SELECT * FROM messages
WHERE session_id = ?
ORDER BY created_at ASC;

-- name: GetSessionMessagesWithLimit :many
SELECT * FROM messages
WHERE session_id = ?
ORDER BY created_at ASC
LIMIT ?;

-- name: GetMessagesFromID :many
SELECT m.* FROM messages m
WHERE m.session_id = ? AND m.created_at >= (SELECT m2.created_at FROM messages m2 WHERE m2.id = ?)
ORDER BY m.created_at ASC;

-- name: GetSummaryMessage :one
SELECT * FROM messages
WHERE session_id = ? AND is_summary = 1
ORDER BY created_at DESC
LIMIT 1;

-- name: CountSessionMessages :one
SELECT COUNT(*) FROM messages WHERE session_id = ?;

-- name: UpdateMessageParts :exec
UPDATE messages SET parts = ?, updated_at = ? WHERE id = ?;

-- name: DeleteMessage :exec
DELETE FROM messages WHERE id = ?;

-- name: DeleteSessionMessages :exec
DELETE FROM messages WHERE session_id = ?;

-- name: DeleteOldMessages :exec
DELETE FROM messages AS outer_msg
WHERE outer_msg.session_id = ?
AND outer_msg.id NOT IN (
    SELECT inner_msg.id FROM messages AS inner_msg
    WHERE inner_msg.session_id = outer_msg.session_id
    ORDER BY inner_msg.created_at DESC
    LIMIT ?
);
