-- name: GetOAuthCredential :one
-- Get OAuth credential for a specific provider
SELECT user_id, provider, access_token, refresh_token, id_token, api_key, account_id, 
       client_id, expires_at, last_refresh, created_at, updated_at
FROM oauth_credentials 
WHERE user_id = 'default_user' AND provider = ?;

-- name: ListOAuthCredentials :many
-- List all OAuth credentials for the user
SELECT user_id, provider, access_token, refresh_token, id_token, api_key, account_id, 
       client_id, expires_at, last_refresh, created_at, updated_at
FROM oauth_credentials 
WHERE user_id = 'default_user'
ORDER BY provider;

-- name: CreateOAuthCredential :one
-- Store a new OAuth credential for a provider
INSERT INTO oauth_credentials (
    user_id, provider, access_token, refresh_token, id_token, api_key, account_id,
    client_id, expires_at, last_refresh, created_at, updated_at
) VALUES (
    'default_user', ?, ?, ?, ?, ?, ?, ?, ?, ?, 
    strftime('%s', 'now') * 1000, strftime('%s', 'now') * 1000
) RETURNING user_id, provider, access_token, refresh_token, id_token, api_key, account_id, 
           client_id, expires_at, last_refresh, created_at, updated_at;

-- name: UpsertOAuthCredential :one
-- Insert or update OAuth credential for a provider
INSERT INTO oauth_credentials (
    user_id, provider, access_token, refresh_token, id_token, api_key, account_id,
    client_id, expires_at, last_refresh, created_at, updated_at
) VALUES (
    'default_user', ?, ?, ?, ?, ?, ?, ?, ?, ?, 
    strftime('%s', 'now') * 1000, strftime('%s', 'now') * 1000
) ON CONFLICT(user_id, provider) DO UPDATE SET
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    id_token = excluded.id_token,
    api_key = excluded.api_key,
    account_id = excluded.account_id,
    client_id = excluded.client_id,
    expires_at = excluded.expires_at,
    last_refresh = excluded.last_refresh,
    updated_at = strftime('%s', 'now') * 1000
RETURNING user_id, provider, access_token, refresh_token, id_token, api_key, account_id, 
          client_id, expires_at, last_refresh, created_at, updated_at;

-- name: UpdateOAuthCredential :one
-- Update existing OAuth credential for a provider
UPDATE oauth_credentials 
SET access_token = ?, refresh_token = ?, id_token = ?, api_key = ?, account_id = ?,
    client_id = ?, expires_at = ?, last_refresh = ?, updated_at = strftime('%s', 'now') * 1000
WHERE user_id = 'default_user' AND provider = ?
RETURNING user_id, provider, access_token, refresh_token, id_token, api_key, account_id, 
          client_id, expires_at, last_refresh, created_at, updated_at;

-- name: DeleteOAuthCredential :exec
-- Delete OAuth credential for a provider
DELETE FROM oauth_credentials 
WHERE user_id = 'default_user' AND provider = ?;

-- name: DeleteAllOAuthCredentials :exec
-- Delete all OAuth credentials for the user
DELETE FROM oauth_credentials 
WHERE user_id = 'default_user';

-- name: HasOAuthCredential :one
-- Check if user has OAuth credential for a provider
SELECT COUNT(*) as count
FROM oauth_credentials 
WHERE user_id = 'default_user' AND provider = ? 
AND access_token IS NOT NULL AND access_token != '';

-- name: ListExpiredOAuthCredentials :many
-- Get OAuth credentials that are expired or will expire soon (35 minute buffer)
-- The 35-minute buffer ensures tokens are refreshed before they hit the 5-minute "truly expired" threshold
-- With 30-minute background checks + 5-minute safety margin = zero downtime
SELECT user_id, provider, access_token, refresh_token, id_token, api_key, account_id,
       client_id, expires_at, last_refresh, created_at, updated_at
FROM oauth_credentials
WHERE user_id = 'default_user'
AND expires_at IS NOT NULL
AND expires_at <= (strftime('%s', 'now') + 2100);