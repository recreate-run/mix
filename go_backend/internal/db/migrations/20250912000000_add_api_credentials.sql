-- +goose Up
-- +goose StatementBegin
-- Create API credentials table to store encrypted API keys for providers
-- Replaces dependency on environment variables for API key authentication
CREATE TABLE IF NOT EXISTS api_credentials (
    id TEXT PRIMARY KEY DEFAULT 'default_user',  -- Single user system
    provider TEXT NOT NULL,                      -- anthropic, openai, gemini, groq, openrouter, etc.
    api_key TEXT,                               -- Encrypted API key
    created_at INTEGER NOT NULL,               -- Unix timestamp in milliseconds
    updated_at INTEGER NOT NULL,               -- Unix timestamp in milliseconds
    UNIQUE(id, provider)                       -- One credential per provider per user
);

-- Index for faster provider lookups
CREATE INDEX IF NOT EXISTS idx_api_credentials_provider ON api_credentials(provider);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_api_credentials_updated_at
AFTER UPDATE ON api_credentials
BEGIN
UPDATE api_credentials SET updated_at = strftime('%s', 'now') * 1000
WHERE id = new.id AND provider = new.provider;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove API credentials table and related objects
DROP TRIGGER IF EXISTS update_api_credentials_updated_at;
DROP INDEX IF EXISTS idx_api_credentials_provider;
DROP TABLE IF EXISTS api_credentials;
-- +goose StatementEnd