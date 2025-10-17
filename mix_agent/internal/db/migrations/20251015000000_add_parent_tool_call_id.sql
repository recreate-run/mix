-- +goose Up
-- Add parent_tool_call_id column to track which tool call spawned a subagent session
-- This enables persistent UI nesting of subagent events under their parent task tool
ALTER TABLE sessions ADD COLUMN parent_tool_call_id TEXT;

-- Add index for query performance when filtering by parent tool call
CREATE INDEX IF NOT EXISTS idx_sessions_parent_tool_call_id ON sessions(parent_tool_call_id);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_parent_tool_call_id;
ALTER TABLE sessions DROP COLUMN parent_tool_call_id;
