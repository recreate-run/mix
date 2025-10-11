-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN callbacks TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN callbacks;
-- +goose StatementEnd
