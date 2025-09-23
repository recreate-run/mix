-- +goose Up
-- Migration to remove tool_type field from api_credentials table
-- This simplifies the schema to only use provider field

-- Create new table without tool_type
CREATE TABLE api_credentials_new (
    user_id INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT NOT NULL DEFAULT 'default_user',
    provider TEXT NOT NULL,
    api_key TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(id, provider)
);

-- Migrate data from old table (only provider type records)
-- For records with the same id+provider but different tool_types, keep the 'provider' one
INSERT INTO api_credentials_new (id, provider, api_key, created_at, updated_at)
SELECT DISTINCT id, provider, api_key, created_at, updated_at
FROM api_credentials
WHERE tool_type = 'provider'
ON CONFLICT(id, provider) DO UPDATE SET
    api_key = excluded.api_key,
    updated_at = excluded.updated_at;

-- For any records that don't have tool_type = 'provider', migrate them as regular providers
INSERT OR IGNORE INTO api_credentials_new (id, provider, api_key, created_at, updated_at)
SELECT DISTINCT id, provider, api_key, created_at, updated_at
FROM api_credentials
WHERE tool_type != 'provider';

-- Drop old table and rename new one
DROP TABLE api_credentials;
ALTER TABLE api_credentials_new RENAME TO api_credentials;

-- +goose Down
-- Restore the tool_type field
CREATE TABLE api_credentials_new (
    user_id INTEGER PRIMARY KEY AUTOINCREMENT,
    id TEXT NOT NULL DEFAULT 'default_user',
    provider TEXT NOT NULL,
    api_key TEXT,
    tool_type TEXT NOT NULL DEFAULT 'provider',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(id, provider, tool_type)
);

-- Migrate data back
INSERT INTO api_credentials_new (id, provider, api_key, tool_type, created_at, updated_at)
SELECT id, provider, api_key, 'provider', created_at, updated_at
FROM api_credentials;

-- Drop old table and rename new one
DROP TABLE api_credentials;
ALTER TABLE api_credentials_new RENAME TO api_credentials;