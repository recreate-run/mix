-- name: CreateSession :one
INSERT INTO sessions (
    id,
    parent_session_id,
    parent_tool_call_id,
    title,
    custom_system_prompt,
    prompt_mode,
    callbacks,
    session_type,
    subagent_type,
    browser_mode,
    cdp_url,
    prompt_tokens,
    completion_tokens,
    cost,
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
    ?,
    ?,
    ?,
    ?,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING
    id,
    parent_session_id,
    parent_tool_call_id,
    title,
    custom_system_prompt,
    prompt_mode,
    callbacks,
    session_type,
    subagent_type,
    browser_mode,
    cdp_url,
    prompt_tokens,
    completion_tokens,
    cost,
    created_at,
    updated_at;

-- name: GetSessionByID :one
SELECT
    s.id,
    s.parent_session_id,
    s.parent_tool_call_id,
    s.title,
    s.custom_system_prompt,
    s.prompt_mode,
    s.callbacks,
    s.session_type,
    s.subagent_type,
    s.browser_mode,
    s.cdp_url,
    s.prompt_tokens,
    s.completion_tokens,
    s.cost,
    s.created_at,
    s.updated_at,
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
    s.parent_tool_call_id,
    s.title,
    s.custom_system_prompt,
    s.prompt_mode,
    s.callbacks,
    s.session_type,
    s.subagent_type,
    s.browser_mode,
    s.cdp_url,
    s.prompt_tokens,
    s.completion_tokens,
    s.cost,
    s.created_at,
    s.updated_at,
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
ORDER BY s.created_at DESC
LIMIT 20;

-- name: ListSessionsWithContent :many
SELECT
    s.id,
    s.parent_session_id,
    s.parent_tool_call_id,
    s.title,
    s.custom_system_prompt,
    s.prompt_mode,
    s.callbacks,
    s.session_type,
    s.subagent_type,
    s.browser_mode,
    s.cdp_url,
    s.prompt_tokens,
    s.completion_tokens,
    s.cost,
    s.created_at,
    s.updated_at,
    0 as user_message_count,
    0 as assistant_message_count,
    0 as tool_call_count
FROM sessions s
ORDER BY s.created_at DESC
LIMIT 20;

-- name: UpdateSession :one
UPDATE sessions
SET
    title = ?,
    custom_system_prompt = ?,
    prompt_mode = ?,
    callbacks = ?,
    prompt_tokens = ?,
    completion_tokens = ?,
    cost = ?,
    updated_at = strftime('%s', 'now')
WHERE id = ?
RETURNING
    id,
    parent_session_id,
    parent_tool_call_id,
    title,
    custom_system_prompt,
    prompt_mode,
    callbacks,
    session_type,
    subagent_type,
    prompt_tokens,
    completion_tokens,
    cost,
    created_at,
    updated_at;

-- name: IncrementSessionCost :exec
UPDATE sessions
SET cost = cost + ?,
    updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;
