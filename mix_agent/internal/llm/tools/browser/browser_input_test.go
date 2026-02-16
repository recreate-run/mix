package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateKeyboardInput(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "plain text",
			input:     "hello world",
			wantError: false,
		},
		{
			name:      "text with Enter key",
			input:     "search{Enter}",
			wantError: false,
		},
		{
			name:      "modifier key",
			input:     "{cmd+a}",
			wantError: false,
		},
		{
			name:      "multiple keys",
			input:     "{Backspace}{Backspace}{Delete}",
			wantError: false,
		},
		{
			name:      "mixed text and keys",
			input:     "hello{Tab}world{Enter}",
			wantError: false,
		},
		{
			name:      "escaped braces",
			input:     "code: {{example}}",
			wantError: false,
		},
		{
			name:      "unclosed brace",
			input:     "hello{Enter",
			wantError: true,
			errorMsg:  "unclosed brace",
		},
		{
			name:      "unknown key",
			input:     "{InvalidKey}",
			wantError: true,
			errorMsg:  "unknown key: InvalidKey",
		},
		{
			name:      "empty key sequence",
			input:     "hello{}",
			wantError: true,
			errorMsg:  "empty key sequence",
		},
		{
			name:      "unmatched closing brace",
			input:     "hello}",
			wantError: true,
			errorMsg:  "unmatched closing brace",
		},
		{
			name:      "all function keys",
			input:     "{F1}{F5}{F12}",
			wantError: false,
		},
		{
			name:      "arrow keys",
			input:     "{ArrowUp}{ArrowDown}{ArrowLeft}{ArrowRight}",
			wantError: false,
		},
		{
			name:      "navigation keys",
			input:     "{Home}{End}{PageUp}{PageDown}",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			err := validateKeyboardInput(tt.input)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseKeyboardInput(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		input    string
		expected []keyboardSegment
	}{
		{
			name:  "plain text only",
			input: "hello world",
			expected: []keyboardSegment{
				{isKey: false, value: "hello world"},
			},
		},
		{
			name:  "single key only",
			input: "{Enter}",
			expected: []keyboardSegment{
				{isKey: true, value: "Enter"},
			},
		},
		{
			name:  "text then key",
			input: "search{Enter}",
			expected: []keyboardSegment{
				{isKey: false, value: "search"},
				{isKey: true, value: "Enter"},
			},
		},
		{
			name:  "key then text",
			input: "{cmd+a}new text",
			expected: []keyboardSegment{
				{isKey: true, value: "cmd+a"},
				{isKey: false, value: "new text"},
			},
		},
		{
			name:  "multiple keys",
			input: "{Backspace}{Backspace}{Delete}",
			expected: []keyboardSegment{
				{isKey: true, value: "Backspace"},
				{isKey: true, value: "Backspace"},
				{isKey: true, value: "Delete"},
			},
		},
		{
			name:  "mixed text and keys",
			input: "hello{Tab}world{Enter}",
			expected: []keyboardSegment{
				{isKey: false, value: "hello"},
				{isKey: true, value: "Tab"},
				{isKey: false, value: "world"},
				{isKey: true, value: "Enter"},
			},
		},
		{
			name:  "escaped opening brace",
			input: "code: {{example",
			expected: []keyboardSegment{
				{isKey: false, value: "code: {example"},
			},
		},
		{
			name:  "escaped closing brace",
			input: "example}}",
			expected: []keyboardSegment{
				{isKey: false, value: "example}"},
			},
		},
		{
			name:  "complex with escapes",
			input: "format: {{key}}{Enter}",
			expected: []keyboardSegment{
				{isKey: false, value: "format: {key}"},
				{isKey: true, value: "Enter"},
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			result := parseKeyboardInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidKey(t *testing.T) {
	t.Helper()

	validKeys := []string{
		"Enter", "Tab", "Escape", "Space",
		"ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight",
		"Home", "End", "PageUp", "PageDown",
		"Backspace", "Delete", "Insert",
		"F1", "F2", "F5", "F10", "F12",
		"cmd+a", "ctrl+c", "shift+Tab", "alt+F4",
	}

	for _, key := range validKeys {
		t.Run(key, func(t *testing.T) {
			t.Helper()
			assert.True(t, isValidKey(key), "expected %s to be valid", key)
		})
	}

	invalidKeys := []string{
		"InvalidKey", "F13", "ctrl", "cmd", "unknown+a",
	}

	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			t.Helper()
			assert.False(t, isValidKey(key), "expected %s to be invalid", key)
		})
	}
}
