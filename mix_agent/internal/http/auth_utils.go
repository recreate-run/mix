package http

import (
	"fmt"
	"strings"

	"mix/internal/constants"
)

const (
	providerAnthropic = "anthropic"
)

// getAuthenticationErrorMessage returns a provider-specific authentication error message
func getAuthenticationErrorMessage() string {
	// Use hardcoded default provider
	preferredProvider := constants.DefaultProvider

	// Get a user-friendly name for the provider
	providerName := getProviderDisplayName(string(preferredProvider))

	// Create provider-specific message with helpful instructions
	return fmt.Sprintf("⚠️ Not authenticated with %s (the default provider)\n\n"+
		"Please authenticate to connect your %s account",
		providerName, providerName)
}

// getProviderDisplayName returns a user-friendly display name for providers
func getProviderDisplayName(provider string) string {
	switch strings.ToLower(provider) {
	case providerAnthropic:
		return "Anthropic (Claude)"
	case "openai":
		return "OpenAI (GPT)"
	case "openrouter":
		return "OpenRouter"
	case "gemini":
		return "Google Gemini"
	case "groq":
		return "Groq"
	case "azure":
		return "Azure OpenAI"
	case "vertexai":
		return "Google Vertex AI"
	case "xai":
		return "xAI (Grok)"
	default:
		// Capitalize first letter of provider name
		if provider == "" {
			return provider
		}
		return strings.ToUpper(provider[:1]) + provider[1:]
	}
}
