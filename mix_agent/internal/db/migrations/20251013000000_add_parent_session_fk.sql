-- +goose Up
-- Add foreign key constraint to parent_session_id for referential integrity
-- This migration drops and recreates the sessions table with the FK constraint
-- WARNING: This deletes all existing session data

DROP TABLE IF EXISTS sessions;

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

-- +goose StatementBegin
-- Recreate trigger for automatic updated_at management
CREATE TRIGGER update_sessions_updated_at
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;
-- +goose StatementEnd

-- Create index on parent_session_id for query performance
CREATE INDEX idx_sessions_parent_id ON sessions(parent_session_id);

-- +goose Down
-- Remove foreign key constraint by recreating table without it
-- WARNING: This deletes all existing session data

DROP TABLE IF EXISTS sessions;

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

-- +goose StatementBegin
-- Recreate trigger
CREATE TRIGGER update_sessions_updated_at
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;
-- +goose StatementEnd
