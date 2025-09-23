-- name: GetAPICredential :one
-- Get API credential for a specific provider
SELECT user_id, id, provider, api_key, tool_type, created_at, updated_at
FROM api_credentials 
WHERE id = 'default_user' AND provider = ? AND tool_type = COALESCE(?, 'provider');

-- name: ListAPICredentials :many
-- List all API credentials for the user
SELECT user_id, id, provider, api_key, tool_type, created_at, updated_at
FROM api_credentials 
WHERE id = 'default_user'
ORDER BY tool_type, provider;

-- name: CreateAPICredential :one
-- Store a new API credential for a provider
INSERT INTO api_credentials (
    id, provider, api_key, tool_type, created_at, updated_at
) VALUES (
    'default_user', ?, ?, COALESCE(?, 'provider'), strftime('%s', 'now') * 1000, strftime('%s', 'now') * 1000
) RETURNING user_id, id, provider, api_key, tool_type, created_at, updated_at;

-- name: UpdateAPICredential :one
-- Update existing API credential for a provider
UPDATE api_credentials 
SET api_key = ?, updated_at = strftime('%s', 'now') * 1000
WHERE id = 'default_user' AND provider = ? AND tool_type = COALESCE(?, 'provider')
RETURNING user_id, id, provider, api_key, tool_type, created_at, updated_at;

-- name: UpsertAPICredential :one
-- Insert or update API credential for a provider
INSERT INTO api_credentials (
    id, provider, api_key, tool_type, created_at, updated_at
) VALUES (
    'default_user', ?, ?, COALESCE(?, 'provider'), strftime('%s', 'now') * 1000, strftime('%s', 'now') * 1000
) ON CONFLICT(id, provider, tool_type) DO UPDATE SET
    api_key = excluded.api_key,
    updated_at = strftime('%s', 'now') * 1000
RETURNING user_id, id, provider, api_key, tool_type, created_at, updated_at;

-- name: DeleteAPICredential :exec
-- Delete API credential for a provider
DELETE FROM api_credentials 
WHERE id = 'default_user' AND provider = ? AND tool_type = COALESCE(?, 'provider');

-- name: DeleteAllAPICredentials :exec
-- Delete all API credentials for the user
DELETE FROM api_credentials 
WHERE id = 'default_user';

-- name: HasAPICredential :one
-- Check if user has API credential for a provider
SELECT COUNT(*) as count
FROM api_credentials 
WHERE id = 'default_user' AND provider = ? AND tool_type = COALESCE(?, 'provider') AND api_key IS NOT NULL AND api_key != '';

-- name: ListToolCredentials :many
-- List all tool credentials by tool type
SELECT user_id, id, provider, api_key, tool_type, created_at, updated_at
FROM api_credentials 
WHERE id = 'default_user' AND tool_type = ?
ORDER BY provider;

-- name: GetToolCredential :one
-- Get API credential for a specific tool
SELECT user_id, id, provider, api_key, tool_type, created_at, updated_at
FROM api_credentials 
WHERE id = 'default_user' AND provider = ? AND tool_type = ?;