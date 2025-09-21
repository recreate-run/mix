-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN working_directory TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN working_directory;
-- +goose StatementEnd