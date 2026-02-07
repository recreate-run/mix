package tools

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test LoadToolDescription function
func TestLoadToolDescription(t *testing.T) {
	tests := []struct {
		name           string
		toolName       string
		expectError    bool
		expectedSubstr string
	}{
		{
			name:           "load existing tool description - glob",
			toolName:       "glob",
			expectError:    false,
			expectedSubstr: "Fast file pattern matching tool",
		},
		{
			name:           "load existing tool description - bash_output",
			toolName:       "bash_output",
			expectError:    false,
			expectedSubstr: "", // Just check it doesn't error
		},
		{
			name:           "load existing tool description - web_search",
			toolName:       "web_search",
			expectError:    false,
			expectedSubstr: "", // Just check it doesn't error
		},
		{
			name:           "non-existent tool description",
			toolName:       "non_existent_tool",
			expectError:    true,
			expectedSubstr: "Error: failed to load embedded tool description 'non_existent_tool'",
		},
		{
			name:           "empty tool name",
			toolName:       "",
			expectError:    true,
			expectedSubstr: "Error: failed to load embedded tool description ''",
		},
		{
			name:           "tool name with special characters",
			toolName:       "tool/../../../etc/passwd",
			expectError:    true,
			expectedSubstr: "Error: failed to load embedded tool description 'tool/../../../etc/passwd'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LoadToolDescription(tt.toolName)

			if tt.expectError {
				assert.Contains(t, result, tt.expectedSubstr)
				assert.Contains(t, result, "Error:")
			} else {
				assert.NotContains(t, result, "Error:")
				if tt.expectedSubstr != "" {
					assert.Contains(t, result, tt.expectedSubstr)
				}
				// Verify content is not empty for successful loads
				assert.NotEmpty(t, result)
			}
		})
	}
}

// Test that LoadToolDescription trims whitespace
func TestLoadToolDescriptionTrimsWhitespace(t *testing.T) {
	// Load a known description and verify it's trimmed
	result := LoadToolDescription("glob")

	// Should not have leading or trailing whitespace
	assert.Equal(t, strings.TrimSpace(result), result)

	// Should have actual content
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Fast file pattern matching tool")
}

// Test path construction with different tool names
func TestLoadToolDescriptionPathConstruction(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
	}{
		{
			name:     "simple tool name",
			toolName: "glob",
		},
		{
			name:     "tool name with underscore",
			toolName: "bash_output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LoadToolDescription(tt.toolName)

			// Should successfully load without error
			assert.NotContains(t, result, "Error:")
			assert.NotEmpty(t, result)
		})
	}
}

// Test error message format
func TestLoadToolDescriptionErrorFormat(t *testing.T) {
	toolName := "definitely_does_not_exist"
	result := LoadToolDescription(toolName)

	// Should contain specific error format
	assert.Contains(t, result, "Error: failed to load embedded tool description")
	assert.Contains(t, result, toolName)
	assert.Contains(t, result, ":")

	// Should be a single line error message
	lines := strings.Split(result, "\n")
	assert.Len(t, lines, 1)
}

// Test with various edge cases
func TestLoadToolDescriptionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected bool // true if should succeed, false if should error
	}{
		{
			name:     "normal alphanumeric name",
			toolName: "read_text",
			expected: true,
		},
		{
			name:     "name with dots",
			toolName: "tool.with.dots",
			expected: false, // likely doesn't exist
		},
		{
			name:     "very long name",
			toolName: strings.Repeat("a", 100),
			expected: false,
		},
		{
			name:     "single character name",
			toolName: "a",
			expected: false, // likely doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LoadToolDescription(tt.toolName)

			if tt.expected {
				assert.NotContains(t, result, "Error:")
				assert.NotEmpty(t, result)
			} else {
				assert.Contains(t, result, "Error:")
			}
		})
	}
}
