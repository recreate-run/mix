package models

const (
	ProviderGemini ModelProvider = "gemini"

	// Models
	Gemini25Flash ModelID = "gemini-2.5-flash"
	Gemini25      ModelID = "gemini-2.5"
	Gemini15Pro   ModelID = "gemini-1.5-pro"
	Gemini20Flash ModelID = "gemini-2.0-flash"
)

var GeminiModels = map[ModelID]Model{
	Gemini25: {
		ID:                  Gemini25,
		Name:                "Gemini 2.5 Pro",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.5-pro",
		CostPer1MIn:         1.25,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        10,
		ContextWindow:       1000000,
		DefaultMaxTokens:    50000,
		SupportsAttachments: true,
	},
	Gemini15Pro: {
		ID:                  Gemini15Pro,
		Name:                "Gemini 1.5 Pro",
		Provider:            ProviderGemini,
		APIModel:            "gemini-1.5-pro",
		CostPer1MIn:         1.25,
		CostPer1MInCached:   0.31,
		CostPer1MOutCached:  0,
		CostPer1MOut:        5.0,
		ContextWindow:       2000000,
		DefaultMaxTokens:    50000,
		SupportsAttachments: true,
	},
	Gemini20Flash: {
		ID:                  Gemini20Flash,
		Name:                "Gemini 2.0 Flash",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.0-flash",
		CostPer1MIn:         0.15,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.60,
		ContextWindow:       1000000,
		DefaultMaxTokens:    50000,
		SupportsAttachments: true,
	},
	Gemini25Flash: {
		ID:                  Gemini25Flash,
		Name:                "Gemini 2.5 Flash",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.5-flash",
		CostPer1MIn:         0.15,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.60,
		ContextWindow:       1000000,
		DefaultMaxTokens:    50000,
		SupportsAttachments: true,
	},
}
