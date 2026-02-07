package models

const (
	ProviderGemini ModelProvider = "gemini"

	// Models
	Gemini3Flash ModelID = "gemini-3-flash-preview"
	Gemini3Pro   ModelID = "gemini-3-pro-preview"
)

var GeminiModels = map[ModelID]Model{
	Gemini3Flash: {
		ID:                  Gemini3Flash,
		Name:                "Gemini 3 Flash Preview",
		Provider:            ProviderGemini,
		APIModel:            "gemini-3-flash-preview",
		CostPer1MIn:         0.50,
		CostPer1MInCached:   0.05,
		CostPer1MOutCached:  0,
		CostPer1MOut:        3.0,
		ContextWindow:       1000000,
		DefaultMaxTokens:    50000,
		SupportsAttachments: true,
	},
	Gemini3Pro: {
		ID:                  Gemini3Pro,
		Name:                "Gemini 3 Pro Preview",
		Provider:            ProviderGemini,
		APIModel:            "gemini-3-pro-preview",
		CostPer1MIn:         2.0,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        12.0,
		ContextWindow:       1000000,
		DefaultMaxTokens:    50000,
		SupportsAttachments: true,
	},
}
