-- +goose Up
-- +goose StatementBegin
-- Remove stored message count complexity and compute counts on-demand
-- This eliminates trigger complexity, state drift, and synchronization bugs

-- Drop existing message count triggers
DROP TRIGGER IF EXISTS update_session_message_count_on_insert;
DROP TRIGGER IF EXISTS update_session_message_count_on_delete;

-- Remove stored message_count column (we'll compute counts dynamically)
ALTER TABLE sessions DROP COLUMN message_count;

-- +goose StatementEnd

-- +goose Down  
-- +goose StatementBegin
-- Restore message_count column and triggers for rollback
ALTER TABLE sessions ADD COLUMN message_count INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0);

-- Populate message_count from actual message data
UPDATE sessions SET message_count = (
    SELECT COUNT(*) FROM messages WHERE session_id = sessions.id
);

-- Recreate triggers
CREATE TRIGGER IF NOT EXISTS update_session_message_count_on_insert
AFTER INSERT ON messages
BEGIN
UPDATE sessions SET
    message_count = message_count + 1
WHERE id = new.session_id;
END;

CREATE TRIGGER IF NOT EXISTS update_session_message_count_on_delete
AFTER DELETE ON messages
BEGIN
UPDATE sessions SET
    message_count = message_count - 1
WHERE id = old.session_id;
END;

-- +goose StatementEnd