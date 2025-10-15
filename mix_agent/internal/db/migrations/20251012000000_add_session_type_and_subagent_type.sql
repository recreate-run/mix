-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN session_type TEXT NOT NULL DEFAULT 'main' CHECK (session_type IN ('main', 'subagent', 'forked'));
ALTER TABLE sessions ADD COLUMN subagent_type TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN session_type;
ALTER TABLE sessions DROP COLUMN subagent_type;
-- +goose StatementEnd
