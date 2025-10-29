package commands

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test error definitions
func TestErrorDefinitions(t *testing.T) {
	// Test that all expected errors are defined
	require.Error(t, ErrNotSlashCommand)
	require.Error(t, ErrEmptyCommand)
	require.Error(t, ErrCommandNotFound)
	require.Error(t, ErrCommandFailed)
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrNotSlashCommand",
			err:         ErrNotSlashCommand,
			expectedMsg: "input is not a slash command",
		},
		{
			name:        "ErrEmptyCommand",
			err:         ErrEmptyCommand,
			expectedMsg: "command cannot be empty",
		},
		{
			name:        "ErrCommandNotFound",
			err:         ErrCommandNotFound,
			expectedMsg: "command not found",
		},
		{
			name:        "ErrCommandFailed",
			err:         ErrCommandFailed,
			expectedMsg: "command execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMsg, tt.err.Error())
		})
	}
}

// Test error usage with errors.Is
func TestErrorIsComparison(t *testing.T) {
	// Test that different errors are not equal
	require.NotErrorIs(t, ErrNotSlashCommand, ErrEmptyCommand)
	require.NotErrorIs(t, ErrCommandNotFound, ErrCommandFailed)
}

// Test wrapped errors
func TestWrappedErrors(t *testing.T) {
	// Test wrapped command not found error
	wrappedNotFound := errors.New("command not found: test-cmd")
	require.NotErrorIs(t, wrappedNotFound, ErrCommandNotFound)

	// But actual wrapping should work
	actualWrapped := errors.Join(ErrCommandNotFound, errors.New("test-cmd"))
	require.ErrorIs(t, actualWrapped, ErrCommandNotFound)
}

// Test error types are consistent
func TestErrorTypes(t *testing.T) {
	// All our errors should be standard error type
	var err error

	err = ErrNotSlashCommand
	assert.Implements(t, (*error)(nil), err)

	err = ErrEmptyCommand
	assert.Implements(t, (*error)(nil), err)

	err = ErrCommandNotFound
	assert.Implements(t, (*error)(nil), err)

	err = ErrCommandFailed
	assert.Implements(t, (*error)(nil), err)
}
