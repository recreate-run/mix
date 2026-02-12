-- +goose Up
-- +goose StatementBegin
-- Add token usage tracking columns to messages table
ALTER TABLE messages ADD COLUMN input_tokens INTEGER DEFAULT 0 CHECK (input_tokens >= 0);
ALTER TABLE messages ADD COLUMN output_tokens INTEGER DEFAULT 0 CHECK (output_tokens >= 0);
ALTER TABLE messages ADD COLUMN cache_creation_tokens INTEGER DEFAULT 0 CHECK (cache_creation_tokens >= 0);
ALTER TABLE messages ADD COLUMN cache_read_tokens INTEGER DEFAULT 0 CHECK (cache_read_tokens >= 0);
ALTER TABLE messages ADD COLUMN cost REAL DEFAULT 0.0 CHECK (cost >= 0.0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- SQLite doesn't support DROP COLUMN in older versions, but this is for reference
-- In practice, rolling back requires recreating the table
ALTER TABLE messages DROP COLUMN input_tokens;
ALTER TABLE messages DROP COLUMN output_tokens;
ALTER TABLE messages DROP COLUMN cache_creation_tokens;
ALTER TABLE messages DROP COLUMN cache_read_tokens;
ALTER TABLE messages DROP COLUMN cost;
-- +goose StatementEnd
