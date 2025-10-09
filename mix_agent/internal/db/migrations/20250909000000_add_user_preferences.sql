-- +goose Up
-- +goose StatementBegin
-- Create user preferences table to store model and provider preferences
-- Replaces agent configurations from .mix.json file
CREATE TABLE IF NOT EXISTS user_preferences (
    id TEXT PRIMARY KEY DEFAULT 'default_user',  -- Single user system
    preferred_provider TEXT,                     -- anthropic, openai, gemini, etc.
    main_agent_model TEXT,                      -- claude-sonnet-4-5, gpt-4, etc.
    main_agent_max_tokens INTEGER,
    main_agent_reasoning_effort TEXT,           -- low, medium, high
    sub_agent_model TEXT,
    sub_agent_max_tokens INTEGER,
    sub_agent_reasoning_effort TEXT,
    created_at INTEGER NOT NULL,               -- Unix timestamp in milliseconds
    updated_at INTEGER NOT NULL                -- Unix timestamp in milliseconds
);

-- Insert default preferences (matching current defaults from config.go)
INSERT OR IGNORE INTO user_preferences (
    id,
    preferred_provider,
    main_agent_model,
    main_agent_max_tokens,
    main_agent_reasoning_effort,
    sub_agent_model,
    sub_agent_max_tokens,
    sub_agent_reasoning_effort,
    created_at,
    updated_at
) VALUES (
    'default_user',
    'anthropic',
    'claude-sonnet-4-5',
    4096,
    '',
    'claude-sonnet-4-5', 
    2048,
    '',
    strftime('%s', 'now') * 1000,
    strftime('%s', 'now') * 1000
);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_user_preferences_updated_at
AFTER UPDATE ON user_preferences
BEGIN
UPDATE user_preferences SET updated_at = strftime('%s', 'now') * 1000
WHERE id = new.id;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove user preferences table and trigger
DROP TRIGGER IF EXISTS update_user_preferences_updated_at;
DROP TABLE IF EXISTS user_preferences;
-- +goose StatementEnd