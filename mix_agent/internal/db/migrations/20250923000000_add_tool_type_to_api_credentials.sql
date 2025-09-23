-- +goose Up
-- +goose StatementBegin
-- Add tool_type column to api_credentials table to support tools/agents
-- tool_type can be: 'provider' (default for existing records), 'web_search', 'multimodal_analyzer'
ALTER TABLE api_credentials ADD COLUMN tool_type TEXT DEFAULT 'provider';

-- Update the unique constraint to include tool_type
-- First drop the existing constraint by recreating the table with new constraint
CREATE TABLE api_credentials_new (
    user_id INTEGER PRIMARY KEY AUTOINCREMENT,  -- Auto-increment primary key
    id TEXT NOT NULL DEFAULT 'default_user',    -- Single user system
    provider TEXT NOT NULL,                     -- anthropic, openai, gemini, groq, openrouter, brave, etc.
    api_key TEXT,                              -- Encrypted API key
    tool_type TEXT NOT NULL DEFAULT 'provider', -- 'provider', 'web_search', 'multimodal_analyzer'
    created_at INTEGER NOT NULL,               -- Unix timestamp in milliseconds
    updated_at INTEGER NOT NULL,               -- Unix timestamp in milliseconds
    UNIQUE(id, provider, tool_type)            -- One credential per provider/tool combination per user
);

-- Copy existing data
INSERT INTO api_credentials_new (user_id, id, provider, api_key, tool_type, created_at, updated_at)
SELECT user_id, id, provider, api_key, 'provider', created_at, updated_at
FROM api_credentials;

-- Drop old table and rename new one
DROP TABLE api_credentials;
ALTER TABLE api_credentials_new RENAME TO api_credentials;

-- Recreate index for faster provider lookups
CREATE INDEX IF NOT EXISTS idx_api_credentials_provider ON api_credentials(provider);

-- Create index for tool_type lookups
CREATE INDEX IF NOT EXISTS idx_api_credentials_tool_type ON api_credentials(tool_type);

-- Recreate trigger to automatically update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_api_credentials_updated_at
AFTER UPDATE ON api_credentials
BEGIN
UPDATE api_credentials SET updated_at = strftime('%s', 'now') * 1000
WHERE user_id = new.user_id;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert to original schema without tool_type
CREATE TABLE api_credentials_old (
    user_id INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT NOT NULL DEFAULT 'default_user',
    provider TEXT NOT NULL,
    api_key TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(id, provider)
);

-- Copy data back (excluding tool-specific entries)
INSERT INTO api_credentials_old (user_id, id, provider, api_key, created_at, updated_at)
SELECT user_id, id, provider, api_key, created_at, updated_at
FROM api_credentials
WHERE tool_type = 'provider';

-- Drop current table and rename old one
DROP TABLE api_credentials;
ALTER TABLE api_credentials_old RENAME TO api_credentials;

-- Recreate original index
CREATE INDEX IF NOT EXISTS idx_api_credentials_provider ON api_credentials(provider);

-- Recreate original trigger
CREATE TRIGGER IF NOT EXISTS update_api_credentials_updated_at
AFTER UPDATE ON api_credentials
BEGIN
UPDATE api_credentials SET updated_at = strftime('%s', 'now') * 1000
WHERE id = new.id AND provider = new.provider;
END;

-- +goose StatementEnd