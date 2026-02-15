-- +goose Up
-- +goose StatementBegin
-- Update existing users to have medium reasoning effort as default
-- This migration ensures all existing users get the new medium default
UPDATE user_preferences
SET
    main_agent_reasoning_effort = 'medium',
    sub_agent_reasoning_effort = 'medium'
WHERE
    (main_agent_reasoning_effort = '' OR main_agent_reasoning_effort IS NULL)
    OR (sub_agent_reasoning_effort = '' OR sub_agent_reasoning_effort IS NULL);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert to empty string (no reasoning)
UPDATE user_preferences
SET
    main_agent_reasoning_effort = '',
    sub_agent_reasoning_effort = ''
WHERE id = 'default_user';
-- +goose StatementEnd
