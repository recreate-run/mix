-- +goose Up
-- +goose StatementBegin

-- Add CHECK constraint to enforce maximum title length of 20 characters
-- SQLite doesn't support ALTER TABLE ADD CONSTRAINT, so we need to recreate the table

-- Create new sessions table with the CHECK constraint
CREATE TABLE sessions_new (
    id TEXT PRIMARY KEY,
    parent_session_id TEXT,
    parent_tool_call_id TEXT,
    title TEXT NOT NULL CHECK(LENGTH(title) <= 20),
    custom_system_prompt TEXT,
    prompt_mode TEXT NOT NULL DEFAULT 'default',
    callbacks TEXT DEFAULT '[]',
    session_type TEXT NOT NULL DEFAULT 'main',
    subagent_type TEXT,
    prompt_tokens INTEGER NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    completion_tokens INTEGER NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
    cost REAL NOT NULL DEFAULT 0.0 CHECK (cost >= 0.0),
    updated_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (parent_session_id) REFERENCES sessions(id) ON DELETE SET NULL
);

-- Copy data from old table (truncating titles that exceed 20 characters)
INSERT INTO sessions_new
SELECT
    id,
    parent_session_id,
    parent_tool_call_id,
    SUBSTR(title, 1, 20) as title,
    custom_system_prompt,
    prompt_mode,
    callbacks,
    session_type,
    subagent_type,
    prompt_tokens,
    completion_tokens,
    cost,
    updated_at,
    created_at
FROM sessions;

-- Drop old table
DROP TABLE sessions;

-- Rename new table
ALTER TABLE sessions_new RENAME TO sessions;

-- Recreate triggers
CREATE TRIGGER IF NOT EXISTS update_sessions_updated_at
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Recreate table without CHECK constraint
CREATE TABLE sessions_new (
    id TEXT PRIMARY KEY,
    parent_session_id TEXT,
    parent_tool_call_id TEXT,
    title TEXT NOT NULL,
    custom_system_prompt TEXT,
    prompt_mode TEXT NOT NULL DEFAULT 'default',
    callbacks TEXT DEFAULT '[]',
    session_type TEXT NOT NULL DEFAULT 'main',
    subagent_type TEXT,
    prompt_tokens INTEGER NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    completion_tokens INTEGER NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
    cost REAL NOT NULL DEFAULT 0.0 CHECK (cost >= 0.0),
    updated_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (parent_session_id) REFERENCES sessions(id) ON DELETE SET NULL
);

-- Copy data back
INSERT INTO sessions_new
SELECT * FROM sessions;

-- Drop constrained table
DROP TABLE sessions;

-- Rename
ALTER TABLE sessions_new RENAME TO sessions;

-- Recreate triggers
CREATE TRIGGER IF NOT EXISTS update_sessions_updated_at
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;

-- +goose StatementEnd
