-- name: CreateSession :one
INSERT INTO sessions (
    id,
    parent_session_id,
    title,
    message_count,
    prompt_tokens,
    completion_tokens,
    cost,
    summary_message_id,
    working_directory,
    updated_at,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    null,
    ?,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING *;

-- name: GetSessionByID :one
SELECT *
FROM sessions
WHERE id = ? LIMIT 1;

-- name: ListSessions :many
SELECT *
FROM sessions
WHERE parent_session_id is NULL
ORDER BY created_at DESC;

-- name: ListSessionsWithFirstMessage :many
SELECT 
    s.id, 
    s.parent_session_id,
    s.title, 
    s.message_count, 
    s.prompt_tokens, 
    s.completion_tokens, 
    s.cost, 
    s.created_at, 
    s.updated_at,
    s.summary_message_id,
    s.working_directory,
    COALESCE(m.parts, '') as first_user_message
FROM sessions s
LEFT JOIN (
    SELECT 
        session_id,
        parts,
        ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at ASC) as rn
    FROM messages 
    WHERE role = 'user'
) m ON s.id = m.session_id AND m.rn = 1
WHERE s.parent_session_id IS NULL
ORDER BY s.created_at DESC;

-- name: UpdateSession :one
UPDATE sessions
SET
    title = ?,
    prompt_tokens = ?,
    completion_tokens = ?,
    summary_message_id = ?,
    cost = ?
WHERE id = ?
RETURNING *;


-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;
