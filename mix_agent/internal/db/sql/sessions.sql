-- name: CreateSession :one
INSERT INTO sessions (
    id,
    parent_session_id,
    title,
    custom_system_prompt,
    prompt_mode,
    session_type,
    subagent_type,
    prompt_tokens,
    completion_tokens,
    cost,
    summary_message_id,
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
    ?,
    ?,
    ?,
    null,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING
    id,
    parent_session_id,
    title,
    custom_system_prompt,
    prompt_mode,
    session_type,
    subagent_type,
    prompt_tokens,
    completion_tokens,
    cost,
    created_at,
    updated_at,
    summary_message_id;

-- name: GetSessionByID :one
SELECT
    s.id,
    s.parent_session_id,
    s.title,
    s.custom_system_prompt,
    s.prompt_mode,
    s.session_type,
    s.subagent_type,
    s.prompt_tokens,
    s.completion_tokens,
    s.cost,
    s.created_at,
    s.updated_at,
    s.summary_message_id,
    COALESCE(counts.user_message_count, 0) as user_message_count,
    COALESCE(counts.assistant_message_count, 0) as assistant_message_count,
    COALESCE(counts.tool_call_count, 0) as tool_call_count
FROM sessions s
LEFT JOIN (
    SELECT session_id,
           COUNT(CASE WHEN role = 'user' THEN 1 END) as user_message_count,
           COUNT(CASE WHEN role = 'assistant' THEN 1 END) as assistant_message_count,
           COUNT(CASE WHEN role = 'tool' THEN 1 END) as tool_call_count
    FROM messages GROUP BY session_id
) counts ON s.id = counts.session_id
WHERE s.id = ? LIMIT 1;

-- name: ListSessionsMetadata :many
SELECT
    s.id,
    s.parent_session_id,
    s.title,
    s.custom_system_prompt,
    s.prompt_mode,
    s.session_type,
    s.subagent_type,
    s.prompt_tokens,
    s.completion_tokens,
    s.cost,
    s.created_at,
    s.updated_at,
    s.summary_message_id,
    COALESCE(counts.user_message_count, 0) as user_message_count,
    COALESCE(counts.assistant_message_count, 0) as assistant_message_count,
    COALESCE(counts.tool_call_count, 0) as tool_call_count
FROM sessions s
LEFT JOIN (
    SELECT session_id,
           COUNT(CASE WHEN role = 'user' THEN 1 END) as user_message_count,
           COUNT(CASE WHEN role = 'assistant' THEN 1 END) as assistant_message_count,
           COUNT(CASE WHEN role = 'tool' THEN 1 END) as tool_call_count
    FROM messages GROUP BY session_id
) counts ON s.id = counts.session_id
ORDER BY s.created_at DESC;

-- name: ListSessionsWithContent :many
SELECT
    s.id,
    s.parent_session_id,
    s.title,
    s.custom_system_prompt,
    s.prompt_mode,
    s.session_type,
    s.subagent_type,
    s.prompt_tokens,
    s.completion_tokens,
    s.cost,
    s.created_at,
    s.updated_at,
    s.summary_message_id,
    COALESCE(first_msg.parts, '') as first_user_message,
    COALESCE(counts.user_message_count, 0) as user_message_count,
    COALESCE(counts.assistant_message_count, 0) as assistant_message_count,
    COALESCE(counts.tool_call_count, 0) as tool_call_count
FROM sessions s
LEFT JOIN (
    SELECT
        session_id,
        parts,
        ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY created_at ASC) as rn
    FROM messages
    WHERE role = 'user'
) first_msg ON s.id = first_msg.session_id AND first_msg.rn = 1
LEFT JOIN (
    SELECT session_id,
           COUNT(CASE WHEN role = 'user' THEN 1 END) as user_message_count,
           COUNT(CASE WHEN role = 'assistant' THEN 1 END) as assistant_message_count,
           COUNT(CASE WHEN role = 'tool' THEN 1 END) as tool_call_count
    FROM messages GROUP BY session_id
) counts ON s.id = counts.session_id
ORDER BY s.created_at DESC;

-- name: UpdateSession :one
UPDATE sessions
SET
    title = ?,
    custom_system_prompt = ?,
    prompt_mode = ?,
    prompt_tokens = ?,
    completion_tokens = ?,
    summary_message_id = ?,
    cost = ?,
    updated_at = strftime('%s', 'now')
WHERE id = ?
RETURNING
    id,
    parent_session_id,
    title,
    custom_system_prompt,
    prompt_mode,
    session_type,
    subagent_type,
    prompt_tokens,
    completion_tokens,
    cost,
    created_at,
    updated_at,
    summary_message_id;

-- name: IncrementSessionCost :exec
UPDATE sessions
SET cost = cost + ?,
    updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;
