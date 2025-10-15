-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN session_type TEXT NOT NULL DEFAULT 'main';
ALTER TABLE sessions ADD COLUMN subagent_type TEXT;
ALTER TABLE sessions ADD CONSTRAINT session_type_check
    CHECK (session_type IN ('main', 'subagent', 'forked'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP CONSTRAINT session_type_check;
ALTER TABLE sessions DROP COLUMN session_type;
ALTER TABLE sessions DROP COLUMN subagent_type;
-- +goose StatementEnd
