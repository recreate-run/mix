package models

import "maps"

type (
	ModelID         string
	ModelProvider   string
	ReasoningEffort string
	AuthMethod      string
)

type Model struct {
	ID                  ModelID       `json:"id"`
	Name                string        `json:"name"`
	Provider            ModelProvider `json:"provider"`
	APIModel            string        `json:"api_model"`
	CostPer1MIn         float64       `json:"cost_per_1m_in"`
	CostPer1MOut        float64       `json:"cost_per_1m_out"`
	CostPer1MInCached   float64       `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached  float64       `json:"cost_per_1m_out_cached"`
	ContextWindow       int64         `json:"context_window"`
	DefaultMaxTokens    int64         `json:"default_max_tokens"`
	CanReason           bool          `json:"can_reason"`
	SupportsAttachments bool          `json:"supports_attachments"`
}

// Model IDs
const ( // GEMINI
	// Bedrock
	BedrockClaude37Sonnet ModelID = "bedrock.claude-3.7-sonnet"
)

const (
	ProviderBedrock ModelProvider = "bedrock"
	// ForTests
	ProviderMock ModelProvider = "__mock"
)

// ReasoningEffort levels
const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
	ReasoningEffortNone   ReasoningEffort = ""
)

// AuthMethod types
const (
	AuthMethodOAuth  AuthMethod = "oauth"
	AuthMethodAPIKey AuthMethod = "api_key"
	AuthMethodNone   AuthMethod = "none"
)

// Providers in order of popularity
var ProviderPopularity = map[ModelProvider]int{
	ProviderAnthropic:    2,
	ProviderAzureFoundry: 2, // Same priority as Anthropic
	ProviderOpenAI:       3,
	ProviderGemini:       4,
	ProviderGROQ:         5,
	ProviderOpenRouter:   6,
	ProviderBedrock:      7,
	ProviderAzure:        8,
	ProviderVertexAI:     9,
}

// ProviderInfo represents information about a provider
type ProviderInfo struct {
	DisplayName string    `json:"display_name"`
	Models      []ModelID `json:"models"`
}

// getProviderDisplayName returns a user-friendly display name for a provider
func getProviderDisplayName(provider ModelProvider) string {
	switch provider {
	case ProviderOpenAI:
		return "OpenAI"
	case ProviderOpenRouter:
		return "OpenRouter"
	case ProviderAnthropic:
		return "Anthropic"
	case ProviderAzureFoundry:
		return "Azure Foundry"
	case ProviderGemini:
		return "Google Gemini"
	default:
		return string(provider)
	}
}

// GetSupportedProviders returns a slice of supported providers in a specific order
func GetSupportedProviders() []ModelProvider {
	// Return providers in a specific, consistent order
	return []ModelProvider{
		ProviderOpenAI,
		ProviderOpenRouter,
		ProviderAnthropic,     // Claude
		ProviderAzureFoundry,
		// ProviderGemini,
	}
}

// GetModelsForProvider returns models for a specific provider in a consistent order
func GetModelsForProvider(provider ModelProvider) []ModelID {
	switch provider {
	case ProviderOpenAI:
		return []ModelID{
			// GPT4oMini,
			// GPT4o,
			GPT41,
			// GPT41Mini,
			// GPT41Nano,
			// O1,
			// O1Mini,
			// O1Pro,
			O3,
			// O3Mini,
			O4Mini,
		}
	case ProviderOpenRouter:
		return []ModelID{
			OpenRouterDeepSeekV31,
			OpenRouterZAIGLM45Air,
			OpenRouterZAIGLM46,
		}
	case ProviderAnthropic:
		return []ModelID{
			Claude45Sonnet,
			Claude4Sonnet,
			Claude37Sonnet,
			Claude4Opus,
		}
	case ProviderAzureFoundry:
		return []ModelID{
			Claude45Sonnet,
			Claude4Sonnet,
			Claude37Sonnet,
			Claude4Opus,
		}
	case ProviderGemini:
		return []ModelID{
			Gemini25,
			Gemini15Pro,
			Gemini20Flash,
			Gemini25Flash,
		}
	default:
		return []ModelID{}
	}
}

// GetProviders returns a map of all providers with their information
func GetProviders() map[ModelProvider]ProviderInfo {
	providers := make(map[ModelProvider]ProviderInfo)

	// Get providers in a fixed order
	orderedProviders := GetSupportedProviders()

	// Process providers in that specific order
	for _, provider := range orderedProviders {
		// Get models for this provider in a specific order
		models := GetModelsForProvider(provider)

		// Create provider info
		providers[provider] = ProviderInfo{
			DisplayName: getProviderDisplayName(provider),
			Models:      models,
		}
	}

	return providers
}

var SupportedModels = map[ModelID]Model{
	//
	// // GEMINI
	// GEMINI25: {
	// 	ID:                 GEMINI25,
	// 	Name:               "Gemini 2.5 Pro",
	// 	Provider:           ProviderGemini,
	// 	APIModel:           "gemini-2.5-pro-exp-03-25",
	// 	CostPer1MIn:        0,
	// 	CostPer1MInCached:  0,
	// 	CostPer1MOutCached: 0,
	// 	CostPer1MOut:       0,
	// },
	//
	// GRMINI20Flash: {
	// 	ID:                 GRMINI20Flash,
	// 	Name:               "Gemini 2.0 Flash",
	// 	Provider:           ProviderGemini,
	// 	APIModel:           "gemini-2.0-flash",
	// 	CostPer1MIn:        0.1,
	// 	CostPer1MInCached:  0,
	// 	CostPer1MOutCached: 0.025,
	// 	CostPer1MOut:       0.4,
	// },
	//
	// // Bedrock
	//BedrockClaude37Sonnet: {
	//	ID:                 BedrockClaude37Sonnet,
	//	Name:               "Bedrock: Claude 3.7 Sonnet",
	//	Provider:           ProviderBedrock,
	//	APIModel:           "anthropic.claude-3-7-sonnet-20250219-v1:0",
	//	CostPer1MIn:        3.0,
	//	CostPer1MInCached:  3.75,
	//	CostPer1MOutCached: 0.30,
	//	CostPer1MOut:       15.0,
	// },
}

func init() {
	maps.Copy(SupportedModels, AnthropicModels)
	maps.Copy(SupportedModels, OpenAIModels)
	maps.Copy(SupportedModels, OpenRouterModels)
	maps.Copy(SupportedModels, AzureFoundryModels)
	// Additional models can be added here when needed:
	// GeminiModels, GroqModels, AzureModels, XAIModels, VertexAIGeminiModels
}

// GetModelByIDAndProvider retrieves a model variant by ID and preferred provider.
// If the model doesn't exist for the preferred provider, it falls back to the default provider.
//
// This handles the case where the same ModelID (e.g., Claude4Sonnet) has multiple provider variants
// (e.g., anthropic and azure-foundry), and we want to select based on user preference.
func GetModelByIDAndProvider(modelID ModelID, preferredProvider ModelProvider) (Model, bool) {
	// Try to find the model with the preferred provider
	var modelVariants []Model

	// Collect all variants of this model from different provider maps
	if m, ok := AnthropicModels[modelID]; ok && (preferredProvider == "" || m.Provider == preferredProvider) {
		if preferredProvider == m.Provider {
			return m, true
		}
		modelVariants = append(modelVariants, m)
	}

	if m, ok := AzureFoundryModels[modelID]; ok && (preferredProvider == "" || m.Provider == preferredProvider) {
		if preferredProvider == m.Provider {
			return m, true
		}
		modelVariants = append(modelVariants, m)
	}

	if m, ok := OpenAIModels[modelID]; ok && (preferredProvider == "" || m.Provider == preferredProvider) {
		if preferredProvider == m.Provider {
			return m, true
		}
		modelVariants = append(modelVariants, m)
	}

	if m, ok := OpenRouterModels[modelID]; ok && (preferredProvider == "" || m.Provider == preferredProvider) {
		if preferredProvider == m.Provider {
			return m, true
		}
		modelVariants = append(modelVariants, m)
	}

	// If no match with preferred provider, return first available variant
	if len(modelVariants) > 0 {
		return modelVariants[0], true
	}

	// Fallback to SupportedModels (last resort, will use whatever was last copied in init)
	if m, ok := SupportedModels[modelID]; ok {
		return m, true
	}

	return Model{}, false
}
