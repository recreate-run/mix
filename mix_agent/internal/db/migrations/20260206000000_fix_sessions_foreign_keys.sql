-- +goose Up
-- +goose StatementBegin

-- Fix broken foreign key references to sessions_old
-- This happened because migration 20251117000000 recreated sessions table
-- but didn't update dependent tables

PRAGMA foreign_keys = OFF;

-- Recreate messages table with correct foreign key
CREATE TABLE messages_new (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,
    parts TEXT NOT NULL default '[]',
    model TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    finished_at INTEGER,
    FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE
);

-- Copy data
INSERT INTO messages_new SELECT * FROM messages;

-- Replace table
DROP TABLE messages;
ALTER TABLE messages_new RENAME TO messages;

-- Recreate indexes
CREATE INDEX idx_messages_session_id ON messages (session_id);
CREATE INDEX idx_messages_role_created_at ON messages (role, created_at DESC);

-- Recreate triggers
CREATE TRIGGER update_messages_updated_at
AFTER UPDATE ON messages
BEGIN
    UPDATE messages SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;

-- Recreate files table with correct foreign key
CREATE TABLE files_new (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    path TEXT NOT NULL,
    content TEXT NOT NULL,
    version TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE,
    UNIQUE(path, session_id, version)
);

-- Copy data
INSERT INTO files_new SELECT * FROM files;

-- Replace table
DROP TABLE files;
ALTER TABLE files_new RENAME TO files;

-- Recreate indexes
CREATE INDEX idx_files_session_id ON files (session_id);
CREATE INDEX idx_files_path ON files (path);

-- Recreate triggers
CREATE TRIGGER update_files_updated_at
AFTER UPDATE ON files
BEGIN
    UPDATE files SET updated_at = strftime('%s', 'now')
    WHERE id = new.id;
END;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- This is intentionally a no-op as reverting would break existing data
-- The broken state shouldn't be restored
SELECT 1;
-- +goose StatementEnd
