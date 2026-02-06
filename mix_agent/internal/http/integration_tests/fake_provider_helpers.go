package integration_tests

import (
	"testing"

	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/llm/provider"
	"mix/internal/message"
)

// setupIntegrationTestServerWithFakeProvider sets up an integration test server
// with a fake LLM provider for fast, deterministic testing without API calls
func setupIntegrationTestServerWithFakeProvider(t *testing.T, config *provider.FakeResponseConfig) *TestServerResult {
	t.Helper()

	// Get mock model from SupportedModels
	mockModel, exists := models.SupportedModels[models.MockClaudeSonnet]
	if !exists {
		t.Fatal("Mock model not found in SupportedModels")
	}

	// Set up the test provider factory to return FakeProvider
	provider.SetTestProviderFactory(func(providerName models.ModelProvider) (interfaces.Provider, error) {
		return provider.NewFakeProvider(mockModel, config), nil
	})

	// Clean up the factory after test
	t.Cleanup(func() {
		provider.ClearTestProviderFactory()
	})

	// Call the standard setup function
	return setupIntegrationTestServer(t)
}

// Helper functions for building common fake response configs

// simpleFakeResponse creates a simple text response
func simpleFakeResponse(content string) *provider.FakeResponseConfig {
	return provider.NewFakeTextResponse(content)
}

// fakeBashToolResponse creates a complete Bash tool sequence:
// 1. Agent wants to use Bash tool
// 2. After tool executes, agent provides final response
func fakeBashToolResponse(command, finalResponse string) *provider.FakeResponseConfig {
	return provider.NewFakeSequence(
		// First: Agent wants to execute bash command
		provider.FakeResponse{
			ToolCalls: []message.ToolCall{{
				Name:  "Bash",
				Input: `{"command":"` + command + `"}`,
				Type:  "tool_use",
			}},
			FinishReason: message.FinishReasonToolUse,
			Usage: interfaces.TokenUsage{
				InputTokens:  10,
				OutputTokens: 15,
			},
		},
		// Second: After tool result, agent provides final response
		provider.FakeResponse{
			Content:      finalResponse,
			FinishReason: message.FinishReasonEndTurn,
			Usage: interfaces.TokenUsage{
				InputTokens:  20,
				OutputTokens: 25,
			},
		},
	)
}
