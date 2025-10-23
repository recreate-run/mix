-- Remove summary_message_id column from sessions table
-- This migration removes the summary feature that is no longer needed

-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN summary_message_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN summary_message_id TEXT;
-- +goose StatementEnd
