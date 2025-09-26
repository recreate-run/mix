-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN custom_system_prompt TEXT;
ALTER TABLE sessions ADD COLUMN prompt_mode TEXT DEFAULT 'default';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN custom_system_prompt;
ALTER TABLE sessions DROP COLUMN prompt_mode;
-- +goose StatementEnd