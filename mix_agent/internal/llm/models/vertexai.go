package models

const (
	ProviderVertexAI ModelProvider = "vertexai"

	// Models
	VertexAIGemini25Flash ModelID = "vertexai.gemini-2.5-flash"
	VertexAIGemini25      ModelID = "vertexai.gemini-2.5"
)

var VertexAIGeminiModels = map[ModelID]Model{
	VertexAIGemini25Flash: {
		ID:                  VertexAIGemini25Flash,
		Name:                "VertexAI: Gemini 3 Flash Preview",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-3-flash-preview",
		CostPer1MIn:         GeminiModels[Gemini3Flash].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini3Flash].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini3Flash].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini3Flash].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini3Flash].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini3Flash].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	VertexAIGemini25: {
		ID:                  VertexAIGemini25,
		Name:                "VertexAI: Gemini 3 Pro Preview",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-3-pro-preview",
		CostPer1MIn:         GeminiModels[Gemini3Pro].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini3Pro].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini3Pro].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini3Pro].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini3Pro].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini3Pro].DefaultMaxTokens,
		SupportsAttachments: true,
	},
}
