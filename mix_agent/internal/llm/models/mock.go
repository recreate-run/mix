package models

const (
	// MockClaudeSonnet is a test-only model that mimics Claude Sonnet behavior
	MockClaudeSonnet ModelID = "mock.claude-4-sonnet"
)

// MockModels contains test-only models for integration testing
var MockModels = map[ModelID]Model{
	MockClaudeSonnet: {
		ID:                  MockClaudeSonnet,
		Name:                "Mock Claude Sonnet (Test)",
		Provider:            ProviderMock,
		APIModel:            "mock-claude-4-sonnet",
		CostPer1MIn:         0.0, // No cost for test models
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    4096,
		CanReason:           true,
		SupportsAttachments: true,
	},
}
