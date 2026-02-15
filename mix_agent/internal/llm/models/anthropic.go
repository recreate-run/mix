package models

const (
	ProviderAnthropic ModelProvider = "anthropic"

	// Models
	Claude35Sonnet ModelID = "claude-3.5-sonnet"
	Claude3Haiku   ModelID = "claude-3-haiku"
	Claude45Sonnet ModelID = "claude-sonnet-4-5"
	ClaudeOpus46   ModelID = "claude-opus-4-6"
	ClaudeHaiku45  ModelID = "claude-haiku-4-5"
)

// https://docs.anthropic.com/en/docs/about-claude/models/all-models
var AnthropicModels = map[ModelID]Model{
	Claude45Sonnet: {
		ID:                  Claude45Sonnet,
		Name:                "Claude 4.5 Sonnet",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-sonnet-4-5-20250929",
		CostPer1MIn:         3.0,
		CostPer1MInCached:   3.75,
		CostPer1MOutCached:  0.30,
		CostPer1MOut:        15.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    50000,
		CanReason:           true,
		SupportsAttachments: true,
	},
	ClaudeOpus46: {
		ID:                  ClaudeOpus46,
		Name:                "Claude Opus 4.6",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-opus-4-6",
		CostPer1MIn:         15.0,
		CostPer1MInCached:   18.75,
		CostPer1MOutCached:  1.50,
		CostPer1MOut:        75.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
	ClaudeHaiku45: {
		ID:                  ClaudeHaiku45,
		Name:                "Claude Haiku 4.5",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-haiku-4-5",
		CostPer1MIn:         0.80,
		CostPer1MInCached:   1.0,
		CostPer1MOutCached:  0.08,
		CostPer1MOut:        4.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
}
