-- +goose Up
-- IMPORTANT: After running this migration, you need to:
-- 1. Regenerate sqlc code: `cd go_backend && sqlc generate`
-- 2. Update any custom code that uses ApiCredential struct
-- 3. Restart the backend server: `make dev`
-- +goose StatementBegin

-- Drop existing triggers and indexes
DROP TRIGGER IF EXISTS update_api_credentials_updated_at;
DROP INDEX IF EXISTS idx_api_credentials_provider;

-- Create a backup of the existing table
CREATE TABLE api_credentials_backup (
    id TEXT NOT NULL DEFAULT 'default_user',
    provider TEXT NOT NULL,
    api_key TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Copy existing data to backup table
INSERT INTO api_credentials_backup SELECT * FROM api_credentials;

-- Drop the existing table
DROP TABLE api_credentials;

-- Create new table with auto-increment user_id
CREATE TABLE api_credentials (
    user_id INTEGER PRIMARY KEY AUTOINCREMENT,  -- Auto-increment primary key
    id TEXT NOT NULL DEFAULT 'default_user',    -- User identifier (keep default_user for backward compatibility)
    provider TEXT NOT NULL,                     -- Provider name (anthropic, openai, etc.)
    api_key TEXT,                               -- Encrypted API key
    created_at INTEGER NOT NULL,                -- Unix timestamp in milliseconds
    updated_at INTEGER NOT NULL,                -- Unix timestamp in milliseconds
    UNIQUE(id, provider)                        -- Allow multiple providers per user
);

-- Copy data back from backup (SQLite will auto-assign user_id values)
INSERT INTO api_credentials (id, provider, api_key, created_at, updated_at)
SELECT id, provider, api_key, created_at, updated_at FROM api_credentials_backup;

-- Drop backup table
DROP TABLE api_credentials_backup;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_api_credentials_provider ON api_credentials(provider);
CREATE INDEX IF NOT EXISTS idx_api_credentials_user_lookup ON api_credentials(id);

-- Create updated trigger for timestamps
CREATE TRIGGER IF NOT EXISTS update_api_credentials_updated_at
AFTER UPDATE ON api_credentials
BEGIN
    UPDATE api_credentials SET updated_at = strftime('%s', 'now') * 1000
    WHERE user_id = new.user_id;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers and indexes
DROP TRIGGER IF EXISTS update_api_credentials_updated_at;
DROP INDEX IF EXISTS idx_api_credentials_provider;
DROP INDEX IF EXISTS idx_api_credentials_user_lookup;

-- Create backup table with original schema
CREATE TABLE api_credentials_backup (
    id TEXT PRIMARY KEY DEFAULT 'default_user',
    provider TEXT NOT NULL,
    api_key TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(id, provider)
);

-- Copy data back (will only keep one row per unique id due to PRIMARY KEY constraint)
INSERT OR IGNORE INTO api_credentials_backup (id, provider, api_key, created_at, updated_at)
SELECT id, provider, api_key, created_at, updated_at FROM api_credentials;

-- Drop the current table
DROP TABLE api_credentials;

-- Recreate original table
CREATE TABLE api_credentials (
    id TEXT PRIMARY KEY DEFAULT 'default_user',
    provider TEXT NOT NULL,
    api_key TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(id, provider)
);

-- Copy data from backup
INSERT INTO api_credentials 
SELECT * FROM api_credentials_backup;

-- Drop backup table
DROP TABLE api_credentials_backup;

-- Recreate original indexes and triggers
CREATE INDEX IF NOT EXISTS idx_api_credentials_provider ON api_credentials(provider);

CREATE TRIGGER IF NOT EXISTS update_api_credentials_updated_at
AFTER UPDATE ON api_credentials
BEGIN
    UPDATE api_credentials SET updated_at = strftime('%s', 'now') * 1000
    WHERE id = new.id AND provider = new.provider;
END;

-- +goose StatementEnd