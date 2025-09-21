-- name: GetUserPreferences :one
SELECT * FROM user_preferences WHERE id = 'default_user' LIMIT 1;

-- name: CreateUserPreferences :one
INSERT INTO user_preferences (
    id,
    preferred_provider,
    main_agent_model,
    main_agent_max_tokens,
    main_agent_reasoning_effort,
    sub_agent_model,
    sub_agent_max_tokens,
    sub_agent_reasoning_effort,
    created_at,
    updated_at
) VALUES (
    'default_user',
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    strftime('%s', 'now') * 1000,
    strftime('%s', 'now') * 1000
)
RETURNING *;

-- name: UpdateUserPreferences :one
UPDATE user_preferences 
SET 
    preferred_provider = ?,
    main_agent_model = ?,
    main_agent_max_tokens = ?,
    main_agent_reasoning_effort = ?,
    sub_agent_model = ?,
    sub_agent_max_tokens = ?,
    sub_agent_reasoning_effort = ?
WHERE id = 'default_user'
RETURNING *;

-- name: UpdateUserPreferredProvider :one
UPDATE user_preferences 
SET preferred_provider = ?
WHERE id = 'default_user'
RETURNING *;

-- name: UpdateMainAgentModel :one
UPDATE user_preferences 
SET 
    main_agent_model = ?,
    main_agent_max_tokens = ?,
    main_agent_reasoning_effort = ?
WHERE id = 'default_user'
RETURNING *;

-- name: UpdateSubAgentModel :one
UPDATE user_preferences 
SET 
    sub_agent_model = ?,
    sub_agent_max_tokens = ?,
    sub_agent_reasoning_effort = ?
WHERE id = 'default_user'
RETURNING *;

-- name: ResetUserPreferencesToDefaults :one
UPDATE user_preferences 
SET 
    preferred_provider = 'anthropic',
    main_agent_model = 'claude-4-sonnet',
    main_agent_max_tokens = 4096,
    main_agent_reasoning_effort = '',
    sub_agent_model = 'claude-4-sonnet',
    sub_agent_max_tokens = 2048,
    sub_agent_reasoning_effort = ''
WHERE id = 'default_user'
RETURNING *;