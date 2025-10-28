package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test ParseCommand function
func TestParseCommand(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedName   string
		expectedArgs   string
		expectedRaw    string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "simple command",
			input:        "/help",
			expectedName: "help",
			expectedArgs: "",
			expectedRaw:  "/help",
			expectError:  false,
		},
		{
			name:         "command with arguments",
			input:        "/review pr123",
			expectedName: "review",
			expectedArgs: "pr123",
			expectedRaw:  "/review pr123",
			expectError:  false,
		},
		{
			name:         "command with multiple arguments",
			input:        "/search query with multiple words",
			expectedName: "search",
			expectedArgs: "query with multiple words",
			expectedRaw:  "/search query with multiple words",
			expectError:  false,
		},
		{
			name:         "command with extra whitespace",
			input:        "/test    arg1   arg2  ",
			expectedName: "test",
			expectedArgs: "arg1   arg2",
			expectedRaw:  "/test    arg1   arg2  ",
			expectError:  false,
		},
		{
			name:         "empty slash command",
			input:        "/",
			expectedName: "",
			expectedArgs: "",
			expectedRaw:  "/",
			expectError:  false,
		},
		{
			name:           "not a slash command",
			input:          "regular text",
			expectError:    true,
			expectedErrMsg: "input is not a slash command",
		},
		{
			name:           "empty input",
			input:          "",
			expectError:    true,
			expectedErrMsg: "input is not a slash command",
		},
		{
			name:         "slash with whitespace only",
			input:        "/   ",
			expectedName: "",
			expectedArgs: "",
			expectedRaw:  "/   ",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseCommand(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, parsed)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsed)
				assert.Equal(t, tt.expectedName, parsed.Name)
				assert.Equal(t, tt.expectedArgs, parsed.Arguments)
				assert.Equal(t, tt.expectedRaw, parsed.RawInput)
			}
		})
	}
}

// Test IsSlashCommand function
func TestIsSlashCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid slash command",
			input:    "/help",
			expected: true,
		},
		{
			name:     "slash command with args",
			input:    "/review pr123",
			expected: true,
		},
		{
			name:     "slash with whitespace",
			input:    "/   ",
			expected: true,
		},
		{
			name:     "slash only",
			input:    "/",
			expected: true,
		},
		{
			name:     "regular text",
			input:    "regular text",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: false,
		},
		{
			name:     "slash in middle",
			input:    "text /command",
			expected: false,
		},
		{
			name:     "slash command with leading whitespace",
			input:    "  /help",
			expected: true,
		},
		{
			name:     "URL that starts with slash",
			input:    "/path/to/file",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSlashCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test ParsedCommand struct
func TestParsedCommandFields(t *testing.T) {
	parsed := &ParsedCommand{
		Name:      "test-command",
		Arguments: "arg1 arg2",
		RawInput:  "/test-command arg1 arg2",
	}

	assert.Equal(t, "test-command", parsed.Name)
	assert.Equal(t, "arg1 arg2", parsed.Arguments)
	assert.Equal(t, "/test-command arg1 arg2", parsed.RawInput)
}

// Test edge cases
func TestParseCommandEdgeCases(t *testing.T) {
	// Command with only spaces as arguments
	parsed, err := ParseCommand("/cmd   ")
	assert.NoError(t, err)
	assert.Equal(t, "cmd", parsed.Name)
	assert.Equal(t, "", parsed.Arguments)

	// Command with newlines (should work)
	parsed, err = ParseCommand("/cmd with\nnewlines")
	assert.NoError(t, err)
	assert.Equal(t, "cmd", parsed.Name)
	assert.Equal(t, "with\nnewlines", parsed.Arguments)

	// Command with tabs (tabs are not treated as separators, only spaces)
	parsed, err = ParseCommand("/cmd\twith\ttabs")
	assert.NoError(t, err)
	assert.Equal(t, "cmd\twith\ttabs", parsed.Name)
	assert.Equal(t, "", parsed.Arguments)
}
