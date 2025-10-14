-- +goose Up
-- +goose NO TRANSACTION
-- Add foreign key constraint to parent_session_id for referential integrity
-- SQLite requires table recreation to add foreign keys to existing columns
-- NO TRANSACTION is required because PRAGMA foreign_keys only works outside transactions

PRAGMA foreign_keys = OFF;

BEGIN;

-- Rename existing table
ALTER TABLE sessions RENAME TO sessions_old;

-- Create new sessions table with foreign key constraint
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    parent_session_id TEXT,
    title TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    completion_tokens INTEGER NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
    cost REAL NOT NULL DEFAULT 0.0 CHECK (cost >= 0.0),
    updated_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    summary_message_id TEXT,
    custom_system_prompt TEXT,
    prompt_mode TEXT DEFAULT 'default',
    session_type TEXT NOT NULL DEFAULT 'main' CHECK (session_type IN ('main', 'subagent', 'forked')),
    subagent_type TEXT,
    FOREIGN KEY (parent_session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- Copy data from old table to new table
INSERT INTO sessions (
    id,
    parent_session_id,
    title,
    prompt_tokens,
    completion_tokens,
    cost,
    updated_at,
    created_at,
    summary_message_id,
    custom_system_prompt,
    prompt_mode,
    session_type,
    subagent_type
)
SELECT
    id,
    parent_session_id,
    title,
    prompt_tokens,
    completion_tokens,
    cost,
    updated_at,
    created_at,
    summary_message_id,
    custom_system_prompt,
    prompt_mode,
    session_type,
    subagent_type
FROM sessions_old;

-- Drop old table
DROP TABLE sessions_old;

COMMIT;

-- +goose StatementBegin
-- Recreate update trigger (must be outside transaction, wrapped to prevent semicolon splitting)
CREATE TRIGGER IF NOT EXISTS update_sessions_updated_at
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;
-- +goose StatementEnd

-- Create index on parent_session_id for query performance
CREATE INDEX IF NOT EXISTS idx_sessions_parent_id ON sessions(parent_session_id);

PRAGMA foreign_keys = ON;

-- +goose Down
-- +goose NO TRANSACTION
-- Remove foreign key constraint by recreating table without it

PRAGMA foreign_keys = OFF;

BEGIN;

-- Rename existing table
ALTER TABLE sessions RENAME TO sessions_old;

-- Create table without foreign key constraint (original structure)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    parent_session_id TEXT,
    title TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    completion_tokens INTEGER NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
    cost REAL NOT NULL DEFAULT 0.0 CHECK (cost >= 0.0),
    updated_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    summary_message_id TEXT,
    custom_system_prompt TEXT,
    prompt_mode TEXT DEFAULT 'default',
    session_type TEXT NOT NULL DEFAULT 'main' CHECK (session_type IN ('main', 'subagent', 'forked')),
    subagent_type TEXT
);

-- Copy data back
INSERT INTO sessions (
    id,
    parent_session_id,
    title,
    prompt_tokens,
    completion_tokens,
    cost,
    updated_at,
    created_at,
    summary_message_id,
    custom_system_prompt,
    prompt_mode,
    session_type,
    subagent_type
)
SELECT
    id,
    parent_session_id,
    title,
    prompt_tokens,
    completion_tokens,
    cost,
    updated_at,
    created_at,
    summary_message_id,
    custom_system_prompt,
    prompt_mode,
    session_type,
    subagent_type
FROM sessions_old;

-- Drop old table
DROP TABLE sessions_old;

COMMIT;

-- +goose StatementBegin
-- Recreate update trigger (must be outside transaction, wrapped to prevent semicolon splitting)
CREATE TRIGGER IF NOT EXISTS update_sessions_updated_at
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;
-- +goose StatementEnd

-- Drop index on parent_session_id
DROP INDEX IF EXISTS idx_sessions_parent_id;

PRAGMA foreign_keys = ON;
