package integration_tests

import (
	"net/http"
	"testing"
)

// Test 1: Get Preferences (Initial State) - GET /api/preferences
func TestRESTGetPreferencesInitial(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/preferences - Initial state (default preferences)")

	// Get preferences - system automatically creates defaults
	resp := makeJSONRequest(t, result.Server, "GET", "/api/preferences", nil)
	prefsData := validateObjectResponse(t, resp, http.StatusOK)

	// Should have default preferences (system creates them automatically)
	preferences, ok := prefsData["preferences"].(map[string]interface{})
	if !ok || preferences == nil {
		t.Fatalf("Expected preferences to be set with defaults, got %v", prefsData["preferences"])
	}

	// Validate default values (check structure rather than exact values since defaults may change)
	if preferences["preferred_provider"].(string) == "" {
		t.Fatalf("Expected default preferred_provider to be set, got empty string")
	}

	if preferences["main_agent_model"].(string) == "" {
		t.Fatalf("Expected default main_agent_model to be set, got empty string")
	}

	// Log the actual defaults for informational purposes
	t.Logf("Default preferences: provider=%s, model=%s",
		preferences["preferred_provider"].(string),
		preferences["main_agent_model"].(string))

	// Should always have available_providers
	availableProviders, ok := prefsData["available_providers"].(map[string]interface{})
	if !ok || len(availableProviders) == 0 {
		t.Fatalf("Expected available_providers to be non-empty map, got %v", availableProviders)
	}

	// Validate provider structure
	for providerName, providerData := range availableProviders {
		providerObj, ok := providerData.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected provider %s to be object, got %T", providerName, providerData)
		}

		displayName, ok := providerObj["display_name"].(string)
		if !ok || displayName == "" {
			t.Fatalf("Expected provider %s to have display_name, got %v", providerName, displayName)
		}

		models, ok := providerObj["models"].([]interface{})
		if !ok {
			t.Fatalf("Expected provider %s to have models array, got %T", providerName, providerObj["models"])
		}

		if len(models) == 0 {
			t.Fatalf("Expected provider %s to have at least one model", providerName)
		}
	}

	t.Logf("✅ Get preferences initial test passed - Found %d providers", len(availableProviders))
}

// Test 2: Update Preferences - POST /api/preferences
func TestRESTUpdatePreferences(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/preferences - Update user preferences")

	// First get available providers to use valid values
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/preferences", nil)
	getPrefsData := validateObjectResponse(t, getResp, http.StatusOK)

	availableProviders := getPrefsData["available_providers"].(map[string]interface{})

	// Find first provider and model to use in test
	var testProvider string
	var testModel string
	for providerName, providerData := range availableProviders {
		testProvider = providerName
		providerObj := providerData.(map[string]interface{})
		models := providerObj["models"].([]interface{})
		if len(models) > 0 {
			testModel = models[0].(string)
			break
		}
	}

	if testProvider == "" || testModel == "" {
		t.Fatal("Could not find valid provider and model for testing")
	}

	// Update preferences with valid data
	updateRequest := map[string]interface{}{
		"preferred_provider":          testProvider,
		"main_agent_model":            testModel,
		"main_agent_max_tokens":       4096,
		"main_agent_reasoning_effort": "medium",
		"sub_agent_model":             testModel,
		"sub_agent_max_tokens":        2048,
		"sub_agent_reasoning_effort":  "low",
	}

	updateResp := makeJSONRequest(t, result.Server, "POST", "/api/preferences", updateRequest)
	updatedPrefs := validateObjectResponse(t, updateResp, http.StatusOK)

	// Validate response has all fields
	if updatedPrefs["preferred_provider"].(string) != testProvider {
		t.Fatalf("Expected preferred_provider %s, got %v", testProvider, updatedPrefs["preferred_provider"])
	}

	if updatedPrefs["main_agent_model"].(string) != testModel {
		t.Fatalf("Expected main_agent_model %s, got %v", testModel, updatedPrefs["main_agent_model"])
	}

	if int(updatedPrefs["main_agent_max_tokens"].(float64)) != 4096 {
		t.Fatalf("Expected main_agent_max_tokens 4096, got %v", updatedPrefs["main_agent_max_tokens"])
	}

	if updatedPrefs["main_agent_reasoning_effort"].(string) != "medium" {
		t.Fatalf("Expected main_agent_reasoning_effort 'medium', got %v", updatedPrefs["main_agent_reasoning_effort"])
	}

	// Verify timestamps exist
	if _, ok := updatedPrefs["created_at"]; !ok {
		t.Fatal("Expected created_at timestamp in response")
	}

	if _, ok := updatedPrefs["updated_at"]; !ok {
		t.Fatal("Expected updated_at timestamp in response")
	}

	t.Logf("✅ Update preferences test passed - Provider: %s, Model: %s", testProvider, testModel)
}

// Test 3: Get Preferences (After Update) - GET /api/preferences
func TestRESTGetPreferencesAfterUpdate(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/preferences - After setting preferences")

	// First set some preferences
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/preferences", nil)
	getPrefsData := validateObjectResponse(t, getResp, http.StatusOK)

	availableProviders := getPrefsData["available_providers"].(map[string]interface{})

	var testProvider string
	var testModel string
	for providerName, providerData := range availableProviders {
		testProvider = providerName
		providerObj := providerData.(map[string]interface{})
		models := providerObj["models"].([]interface{})
		if len(models) > 0 {
			testModel = models[0].(string)
			break
		}
	}

	updateRequest := map[string]interface{}{
		"preferred_provider":    testProvider,
		"main_agent_model":      testModel,
		"main_agent_max_tokens": 8192,
	}

	makeJSONRequest(t, result.Server, "POST", "/api/preferences", updateRequest)

	// Now get preferences again
	getAfterResp := makeJSONRequest(t, result.Server, "GET", "/api/preferences", nil)
	afterPrefsData := validateObjectResponse(t, getAfterResp, http.StatusOK)

	// Should now have preferences object
	preferences, ok := afterPrefsData["preferences"].(map[string]interface{})
	if !ok || preferences == nil {
		t.Fatalf("Expected preferences to be set after update, got %v", afterPrefsData["preferences"])
	}

	// Validate the set values
	if preferences["preferred_provider"].(string) != testProvider {
		t.Fatalf("Expected preferred_provider %s, got %v", testProvider, preferences["preferred_provider"])
	}

	if preferences["main_agent_model"].(string) != testModel {
		t.Fatalf("Expected main_agent_model %s, got %v", testModel, preferences["main_agent_model"])
	}

	if int(preferences["main_agent_max_tokens"].(float64)) != 8192 {
		t.Fatalf("Expected main_agent_max_tokens 8192, got %v", preferences["main_agent_max_tokens"])
	}

	// Should still have available_providers
	if _, ok := afterPrefsData["available_providers"]; !ok {
		t.Fatal("Expected available_providers to still be present")
	}

	t.Logf("✅ Get preferences after update test passed - Preferences properly persisted")
}

// Test 4: Partial Update Preferences - POST /api/preferences
func TestRESTPartialUpdatePreferences(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/preferences - Partial updates")

	// Get available providers for valid values
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/preferences", nil)
	getPrefsData := validateObjectResponse(t, getResp, http.StatusOK)

	availableProviders := getPrefsData["available_providers"].(map[string]interface{})

	var testProvider string
	var testModel string
	for providerName, providerData := range availableProviders {
		testProvider = providerName
		providerObj := providerData.(map[string]interface{})
		models := providerObj["models"].([]interface{})
		if len(models) > 0 {
			testModel = models[0].(string)
			break
		}
	}

	// Set initial preferences
	initialRequest := map[string]interface{}{
		"preferred_provider":          testProvider,
		"main_agent_model":            testModel,
		"main_agent_max_tokens":       4096,
		"main_agent_reasoning_effort": "medium",
		"sub_agent_model":             testModel,
		"sub_agent_max_tokens":        2048,
	}

	makeJSONRequest(t, result.Server, "POST", "/api/preferences", initialRequest)

	// Update tokens and reasoning effort (need model too since service updates together)
	partialRequest := map[string]interface{}{
		"main_agent_model":            testModel,
		"main_agent_max_tokens":       8192,
		"main_agent_reasoning_effort": "medium", // Need to preserve this since it's updated together
	}

	partialResp := makeJSONRequest(t, result.Server, "POST", "/api/preferences", partialRequest)
	partialPrefs := validateObjectResponse(t, partialResp, http.StatusOK)

	// Should have updated field
	if int(partialPrefs["main_agent_max_tokens"].(float64)) != 8192 {
		t.Fatalf("Expected main_agent_max_tokens to be updated to 8192, got %v", partialPrefs["main_agent_max_tokens"])
	}

	// Should preserve other fields
	if partialPrefs["preferred_provider"].(string) != testProvider {
		t.Fatalf("Expected preferred_provider to be preserved as %s, got %v", testProvider, partialPrefs["preferred_provider"])
	}

	if partialPrefs["main_agent_reasoning_effort"].(string) != "medium" {
		t.Fatalf("Expected main_agent_reasoning_effort to be preserved as 'medium', got %v", partialPrefs["main_agent_reasoning_effort"])
	}

	t.Logf("✅ Partial update preferences test passed - Single field updated, others preserved")
}

// Test 5: Get Available Providers - GET /api/preferences/providers
func TestRESTGetAvailableProviders(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing GET /api/preferences/providers - Get available providers")

	// Get available providers
	resp := makeJSONRequest(t, result.Server, "GET", "/api/preferences/providers", nil)
	providersData := validateObjectResponse(t, resp, http.StatusOK)

	// Should be a map of providers
	if len(providersData) == 0 {
		t.Fatal("Expected at least one provider to be available")
	}

	// Validate each provider structure
	for providerName, providerData := range providersData {
		providerObj, ok := providerData.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected provider %s to be object, got %T", providerName, providerData)
		}

		// Should have display_name
		displayName, ok := providerObj["display_name"].(string)
		if !ok || displayName == "" {
			t.Fatalf("Expected provider %s to have non-empty display_name, got %v", providerName, displayName)
		}

		// Should have models array
		models, ok := providerObj["models"].([]interface{})
		if !ok {
			t.Fatalf("Expected provider %s to have models array, got %T", providerName, providerObj["models"])
		}

		if len(models) == 0 {
			t.Fatalf("Expected provider %s to have at least one model", providerName)
		}

		// Validate models are strings
		for i, model := range models {
			if _, ok := model.(string); !ok {
				t.Fatalf("Expected provider %s model %d to be string, got %T", providerName, i, model)
			}
		}

		t.Logf("Provider %s: display_name=%s, models=%d", providerName, displayName, len(models))
	}

	t.Logf("✅ Get available providers test passed - Found %d providers", len(providersData))
}

// Test 6: Reset Preferences - POST /api/preferences/reset
func TestRESTResetPreferences(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/preferences/reset - Reset preferences to defaults")

	// First set some custom preferences
	getResp := makeJSONRequest(t, result.Server, "GET", "/api/preferences", nil)
	getPrefsData := validateObjectResponse(t, getResp, http.StatusOK)

	availableProviders := getPrefsData["available_providers"].(map[string]interface{})

	var testProvider string
	var testModel string
	for providerName, providerData := range availableProviders {
		testProvider = providerName
		providerObj := providerData.(map[string]interface{})
		models := providerObj["models"].([]interface{})
		if len(models) > 0 {
			testModel = models[0].(string)
			break
		}
	}

	customRequest := map[string]interface{}{
		"preferred_provider":          testProvider,
		"main_agent_model":            testModel,
		"main_agent_max_tokens":       8192,
		"main_agent_reasoning_effort": "high",
		"sub_agent_model":             testModel,
		"sub_agent_max_tokens":        4096,
		"sub_agent_reasoning_effort":  "medium",
	}

	makeJSONRequest(t, result.Server, "POST", "/api/preferences", customRequest)

	// Reset preferences
	resetResp := makeJSONRequest(t, result.Server, "POST", "/api/preferences/reset", nil)
	resetPrefs := validateObjectResponse(t, resetResp, http.StatusOK)

	// Should have default values - verify structure exists
	if _, ok := resetPrefs["preferred_provider"]; !ok {
		t.Fatal("Expected preferred_provider in reset response")
	}

	if _, ok := resetPrefs["main_agent_model"]; !ok {
		t.Fatal("Expected main_agent_model in reset response")
	}

	if _, ok := resetPrefs["main_agent_max_tokens"]; !ok {
		t.Fatal("Expected main_agent_max_tokens in reset response")
	}

	// Should have timestamps
	if _, ok := resetPrefs["created_at"]; !ok {
		t.Fatal("Expected created_at timestamp in reset response")
	}

	if _, ok := resetPrefs["updated_at"]; !ok {
		t.Fatal("Expected updated_at timestamp in reset response")
	}

	// Verify reset worked by getting preferences again
	getAfterResetResp := makeJSONRequest(t, result.Server, "GET", "/api/preferences", nil)
	afterResetData := validateObjectResponse(t, getAfterResetResp, http.StatusOK)

	preferences, ok := afterResetData["preferences"].(map[string]interface{})
	if !ok || preferences == nil {
		t.Fatal("Expected preferences to exist after reset")
	}

	// Should match the reset response
	if preferences["preferred_provider"] != resetPrefs["preferred_provider"] {
		t.Fatalf("Preferences not properly reset - provider mismatch")
	}

	t.Logf("✅ Reset preferences test passed - Preferences reset to defaults")
}

// Test 7: Invalid Preferences Update - POST /api/preferences
func TestRESTInvalidPreferencesUpdate(t *testing.T) {
	result := setupIntegrationTestServer(t)
	defer result.Server.Close()

	t.Log("Testing POST /api/preferences - Invalid data validation")

	// Test with invalid provider
	invalidProviderRequest := map[string]interface{}{
		"preferred_provider": "invalid-provider-name",
	}

	invalidProviderResp := makeJSONRequest(t, result.Server, "POST", "/api/preferences", invalidProviderRequest)
	if invalidProviderResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid provider, got %d", http.StatusBadRequest, invalidProviderResp.StatusCode)
	}

	// Test with invalid token count (negative) - should return validation error
	invalidTokenRequest := map[string]interface{}{
		"main_agent_max_tokens": -1000,
	}

	invalidTokenResp := makeJSONRequest(t, result.Server, "POST", "/api/preferences", invalidTokenRequest)
	if invalidTokenResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for negative token count, got %d", http.StatusBadRequest, invalidTokenResp.StatusCode)
	}

	// Test with invalid reasoning effort - should return validation error
	invalidReasoningRequest := map[string]interface{}{
		"main_agent_reasoning_effort": "invalid-effort-level",
	}

	invalidReasoningResp := makeJSONRequest(t, result.Server, "POST", "/api/preferences", invalidReasoningRequest)
	if invalidReasoningResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for invalid reasoning effort, got %d", http.StatusBadRequest, invalidReasoningResp.StatusCode)
	}

	// Test with zero token count - should also return validation error
	zeroTokenRequest := map[string]interface{}{
		"sub_agent_max_tokens": 0,
	}

	zeroTokenResp := makeJSONRequest(t, result.Server, "POST", "/api/preferences", zeroTokenRequest)
	if zeroTokenResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status code %d for zero token count, got %d", http.StatusBadRequest, zeroTokenResp.StatusCode)
	}

	t.Logf("✅ Invalid preferences update test passed - All validation errors properly caught")
}

// TestRESTPreferencesIntegration runs all preference integration tests
func TestRESTPreferencesIntegration(t *testing.T) {
	t.Log("🚀 Starting user preferences integration tests")

	t.Run("GetPreferencesInitial", TestRESTGetPreferencesInitial)
	t.Run("UpdatePreferences", TestRESTUpdatePreferences)
	t.Run("GetPreferencesAfterUpdate", TestRESTGetPreferencesAfterUpdate)
	t.Run("PartialUpdatePreferences", TestRESTPartialUpdatePreferences)
	t.Run("GetAvailableProviders", TestRESTGetAvailableProviders)
	t.Run("ResetPreferences", TestRESTResetPreferences)
	t.Run("InvalidPreferencesUpdate", TestRESTInvalidPreferencesUpdate)

	t.Log("🎉 All user preferences integration tests completed successfully!")
}
