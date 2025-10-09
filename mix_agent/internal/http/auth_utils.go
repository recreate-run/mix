package http

import (
	"context"
	"fmt"
	"strings"

	"mix/internal/config"
)

// getAuthenticationErrorMessage returns a provider-specific authentication error message
func getAuthenticationErrorMessage(ctx context.Context) string {
	// Get user preferences to determine their preferred provider
	userPrefs := config.GetUserPreferences()
	if userPrefs == nil {
		return "⚠️ Authentication required. Please go to settings and authenticate"
	}

	preferredProvider, err := userPrefs.GetPreferredProvider(ctx)
	if err != nil || preferredProvider == "" {
		return "⚠️ Authentication required. Please go to settings and authenticate"
	}

	// Get a user-friendly name for the provider
	providerName := getProviderDisplayName(string(preferredProvider))

	// Create provider-specific message with helpful instructions
	return fmt.Sprintf("⚠️ Not authenticated with %s (your preferred provider)\n\n"+
		"Choose one option:\n"+
		"•Authentication to connect your %s account\n"+
		"•change your preferred provider",
		providerName, providerName)
}

// getProviderDisplayName returns a user-friendly display name for providers
func getProviderDisplayName(provider string) string {
	switch strings.ToLower(provider) {
	case "anthropic":
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
		return strings.Title(provider)
	}
}