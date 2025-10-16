package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test NewExitPlanModeTool constructor
func TestNewExitPlanModeTool(t *testing.T) {
	tool := NewExitPlanModeTool()
	assert.NotNil(t, tool)
	assert.IsType(t, &ExitPlanModeTool{}, tool)
}

// Test ExitPlanModeTool Info method
func TestExitPlanModeTool_Info(t *testing.T) {
	tool := NewExitPlanModeTool()
	info := tool.Info()

	// Verify basic structure
	assert.Equal(t, "ExitPlanMode", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.Description, "plan mode")
	assert.Contains(t, info.Description, "finished presenting your plan")

	// Verify parameters structure
	assert.NotNil(t, info.Parameters)
	assert.Contains(t, info.Parameters, "plan")

	planParam, ok := info.Parameters["plan"].(map[string]any)
	require.True(t, ok, "plan parameter should be a map")
	assert.Equal(t, "string", planParam["type"])
	assert.NotEmpty(t, planParam["description"])
	assert.Contains(t, planParam["description"], "markdown")
	assert.Contains(t, planParam["description"], "concise")

	// Verify required fields
	assert.Equal(t, []string{"plan"}, info.Required)
}

// Test ExitPlanModeTool Run method with valid input
func TestExitPlanModeTool_Run_Success(t *testing.T) {
	tool := NewExitPlanModeTool()
	ctx := context.Background()

	tests := []struct {
		name         string
		plan         string
		expectedSubstr []string
	}{
		{
			name: "simple plan",
			plan: "Step 1: Do something\nStep 2: Do something else",
			expectedSubstr: []string{
				"# Plan Ready for Approval",
				"Step 1: Do something",
				"Step 2: Do something else",
				"✅ Ready to proceed when you confirm",
			},
		},
		{
			name: "plan with markdown",
			plan: "## Implementation Plan\n\n- **Phase 1**: Setup\n- **Phase 2**: Execute",
			expectedSubstr: []string{
				"# Plan Ready for Approval",
				"## Implementation Plan",
				"**Phase 1**: Setup",
				"**Phase 2**: Execute",
				"✅ Ready to proceed when you confirm",
			},
		},
		{
			name: "single line plan",
			plan: "Just do the thing",
			expectedSubstr: []string{
				"# Plan Ready for Approval",
				"Just do the thing",
				"✅ Ready to proceed when you confirm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ToolCall{
				ID:    "test-id",
				Name:  "ExitPlanMode",
				Input: `{"plan":"` + strings.ReplaceAll(tt.plan, "\n", "\\n") + `"}`,
			}

			response, err := tool.Run(ctx, params)

			assert.NoError(t, err)
			assert.False(t, response.IsError)
			assert.Equal(t, "text", string(response.Type))

			for _, substr := range tt.expectedSubstr {
				assert.Contains(t, response.Content, substr)
			}

			// Verify structure - should have title, plan content, separator, and ready message
			assert.Contains(t, response.Content, "# Plan Ready for Approval")
			assert.Contains(t, response.Content, "---")
			assert.Contains(t, response.Content, "✅ Ready to proceed when you confirm")
		})
	}
}

// Test ExitPlanModeTool Run method with empty plan
func TestExitPlanModeTool_Run_EmptyPlan(t *testing.T) {
	tool := NewExitPlanModeTool()
	ctx := context.Background()

	// Test truly empty plan
	params := ToolCall{
		ID:    "test-id",
		Name:  "ExitPlanMode",
		Input: `{"plan":""}`,
	}

	response, err := tool.Run(ctx, params)

	assert.NoError(t, err) // Error is nil but response indicates error
	assert.True(t, response.IsError)
	assert.Equal(t, "text", string(response.Type))
	assert.Equal(t, "Plan is required", response.Content)
}

// Test ExitPlanModeTool Run method with whitespace-only plan (should be accepted)
func TestExitPlanModeTool_Run_WhitespaceOnlyPlan(t *testing.T) {
	tool := NewExitPlanModeTool()
	ctx := context.Background()

	// Whitespace-only plan should be accepted since implementation only checks for empty string
	params := ToolCall{
		ID:    "test-id",
		Name:  "ExitPlanMode",
		Input: `{"plan":"   "}`,
	}

	response, err := tool.Run(ctx, params)

	assert.NoError(t, err)
	assert.False(t, response.IsError) // Should succeed with whitespace plan
	assert.Equal(t, "text", string(response.Type))
	assert.Contains(t, response.Content, "# Plan Ready for Approval")
	assert.Contains(t, response.Content, "   ") // Whitespace should be preserved
	assert.Contains(t, response.Content, "✅ Ready to proceed when you confirm")
}

// Test ExitPlanModeTool Run method with invalid JSON
func TestExitPlanModeTool_Run_InvalidJSON(t *testing.T) {
	tool := NewExitPlanModeTool()
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "malformed JSON",
			input: `{"plan":}`,
		},
		{
			name:  "invalid JSON structure",
			input: `{plan:"test"}`,
		},
		{
			name:  "empty input",
			input: "",
		},
		{
			name:  "non-JSON input",
			input: "not json at all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ToolCall{
				ID:    "test-id",
				Name:  "ExitPlanMode",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, params)

			assert.Error(t, err) // JSON parsing should fail
			assert.True(t, response.IsError)
			assert.Equal(t, "text", string(response.Type))
			assert.Equal(t, "Failed to parse parameters", response.Content)
		})
	}
}

// Test ExitPlanModeTool Run method with missing plan field
func TestExitPlanModeTool_Run_MissingPlanField(t *testing.T) {
	tool := NewExitPlanModeTool()
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing plan field",
			input: `{"other_field":"value"}`,
		},
		{
			name:  "empty JSON object",
			input: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ToolCall{
				ID:    "test-id",
				Name:  "ExitPlanMode",
				Input: tt.input,
			}

			response, err := tool.Run(ctx, params)

			assert.NoError(t, err) // JSON parsing succeeds but plan is empty
			assert.True(t, response.IsError)
			assert.Equal(t, "text", string(response.Type))
			assert.Equal(t, "Plan is required", response.Content)
		})
	}
}

// Test ExitPlanModeParams struct
func TestExitPlanModeParams(t *testing.T) {
	// Test struct can be created and fields accessed
	params := ExitPlanModeParams{
		Plan: "Test plan content",
	}

	assert.Equal(t, "Test plan content", params.Plan)
}

// Test response format consistency
func TestExitPlanModeTool_ResponseFormat(t *testing.T) {
	tool := NewExitPlanModeTool()
	ctx := context.Background()

	planContent := "Test plan with specific content"
	params := ToolCall{
		ID:    "test-id",
		Name:  "ExitPlanMode",
		Input: `{"plan":"` + planContent + `"}`,
	}

	response, err := tool.Run(ctx, params)

	assert.NoError(t, err)
	assert.False(t, response.IsError)

	// Verify the exact format structure
	lines := strings.Split(response.Content, "\n")
	assert.True(t, len(lines) >= 5) // Title, empty, content, empty, separator, empty, ready message

	// Check specific positions
	assert.Equal(t, "# Plan Ready for Approval", lines[0])
	assert.Equal(t, "", lines[1]) // Empty line after title
	assert.Equal(t, planContent, lines[2]) // Plan content
	assert.Equal(t, "", lines[3]) // Empty line before separator
	assert.Equal(t, "---", lines[4]) // Separator
	assert.Equal(t, "", lines[5]) // Empty line after separator
	assert.Equal(t, "✅ Ready to proceed when you confirm.", lines[6]) // Ready message
}

// Test tool implements BaseTool interface
func TestExitPlanModeTool_ImplementsBaseTool(t *testing.T) {
	tool := NewExitPlanModeTool()

	// Should implement BaseTool interface
	var _ BaseTool = tool

	// Should have required methods
	info := tool.Info()
	assert.NotEmpty(t, info.Name)

	// Run method should be callable
	ctx := context.Background()
	params := ToolCall{
		ID:    "test-id",
		Name:  "ExitPlanMode",
		Input: `{"plan":"test"}`,
	}

	response, err := tool.Run(ctx, params)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Content)
}