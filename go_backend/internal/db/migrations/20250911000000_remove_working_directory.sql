-- Drop working_directory column from sessions table
-- This migration removes the arbitrary per-session working directory concept
-- in favor of centralized session-based storage folders

-- Migration: Remove working_directory column
-- +goose Up
-- This migration removes the working_directory column from existing sessions tables
-- For fresh databases, this is essentially a no-op

-- For existing databases with working_directory column, we need to recreate the table
-- For fresh databases, this migration will do nothing (which is correct)

SELECT 1; -- Simple no-op migration for fresh databases

-- +goose Down
ALTER TABLE sessions ADD COLUMN working_directory TEXT;