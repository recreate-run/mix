-- +goose Up
-- +goose StatementBegin
-- Create OAuth credentials table to store encrypted OAuth tokens
-- Consolidates OAuth credential storage into the database alongside API keys
CREATE TABLE IF NOT EXISTS oauth_credentials (
    user_id TEXT NOT NULL DEFAULT 'default_user',  -- Single user system
    provider TEXT NOT NULL,                         -- anthropic, openai, etc.
    access_token TEXT,                              -- Encrypted OAuth access token
    refresh_token TEXT,                             -- Encrypted OAuth refresh token
    id_token TEXT,                                  -- Encrypted ID token (for OpenAI)
    api_key TEXT,                                   -- Encrypted generated API key (for OpenAI)
    account_id TEXT,                                -- Account ID (for OpenAI)
    client_id TEXT NOT NULL,                        -- OAuth client ID
    expires_at INTEGER,                             -- Token expiry (Unix timestamp)
    last_refresh TEXT,                              -- Last refresh timestamp (ISO format)
    created_at INTEGER NOT NULL,                    -- Unix timestamp in milliseconds
    updated_at INTEGER NOT NULL,                    -- Unix timestamp in milliseconds
    UNIQUE(user_id, provider)                       -- One OAuth credential per provider per user
);

-- Index for faster provider lookups
CREATE INDEX IF NOT EXISTS idx_oauth_credentials_provider ON oauth_credentials(provider);

-- Index for token expiry checks
CREATE INDEX IF NOT EXISTS idx_oauth_credentials_expires_at ON oauth_credentials(expires_at);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_oauth_credentials_updated_at
AFTER UPDATE ON oauth_credentials
BEGIN
UPDATE oauth_credentials SET updated_at = strftime('%s', 'now') * 1000
WHERE user_id = new.user_id AND provider = new.provider;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove OAuth credentials table and related objects
DROP TRIGGER IF EXISTS update_oauth_credentials_updated_at;
DROP INDEX IF EXISTS idx_oauth_credentials_expires_at;
DROP INDEX IF EXISTS idx_oauth_credentials_provider;
DROP TABLE IF EXISTS oauth_credentials;
-- +goose StatementEnd