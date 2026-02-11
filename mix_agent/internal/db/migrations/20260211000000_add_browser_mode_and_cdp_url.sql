-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN browser_mode TEXT NOT NULL DEFAULT 'local-browser-service'
    CHECK (browser_mode IN ('electron-embedded-browser', 'local-browser-service', 'remote-cdp-websocket'));

ALTER TABLE sessions ADD COLUMN cdp_url TEXT;

-- Constraint: cdp_url must be set when browser_mode is 'remote-cdp-websocket'
-- This is enforced at application level via validation
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN cdp_url;
ALTER TABLE sessions DROP COLUMN browser_mode;
-- +goose StatementEnd
