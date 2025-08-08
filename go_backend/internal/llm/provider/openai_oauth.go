package provider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"mix/internal/logging"
)

// OpenAI OAuth constants from openai_auth.md specification
const (
	openaiClientID       = "app_EMoamEEZ73f0CkXaXp7hrann"
	openaiRequiredPort   = 1455
	openaiIssuer         = "https://auth.openai.com"
	openaiRedirectURI    = "http://localhost:1455/auth/callback"
	openaiRequiredScopes = "openid profile email offline_access"
)

// OpenAIPKCECodes holds PKCE challenge and verifier
type OpenAIPKCECodes struct {
	CodeVerifier  string
	CodeChallenge string
}

// OpenAIOAuthFlow handles the OpenAI OAuth authentication flow
type OpenAIOAuthFlow struct {
	ClientID     string
	PKCE         *OpenAIPKCECodes
	State        string
	RedirectURI  string
	server       *http.Server
	resultChan   chan OpenAIAuthResult
	shutdownChan chan bool
}

// OpenAIAuthResult holds the result of OAuth authentication
type OpenAIAuthResult struct {
	Success     bool
	Error       error
	Credentials *OpenAICredentials
}

// OpenAIIDTokenInfo holds parsed ID token information
type OpenAIIDTokenInfo struct {
	Email           string `json:"email,omitempty"`
	ChatGPTPlanType string `json:"chatgpt_plan_type,omitempty"`
}

// NewOpenAIOAuthFlow creates a new OpenAI OAuth flow with PKCE
func NewOpenAIOAuthFlow() (*OpenAIOAuthFlow, error) {
	// Generate PKCE codes
	pkce, err := generateOpenAIPKCE()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Generate secure state parameter
	state, err := generateOpenAISecureRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OAuth state: %w", err)
	}

	flow := &OpenAIOAuthFlow{
		ClientID:     openaiClientID,
		PKCE:         pkce,
		State:        state,
		RedirectURI:  openaiRedirectURI,
		resultChan:   make(chan OpenAIAuthResult, 1),
		shutdownChan: make(chan bool, 1),
	}

	// Set up HTTP server for callback handling
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", flow.handleCallback)
	mux.HandleFunc("/success", flow.handleSuccess)

	flow.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", openaiRequiredPort),
		Handler: mux,
	}

	return flow, nil
}

// generateOpenAIPKCE creates PKCE codes for OAuth security - EXACTLY matching Python implementation
func generateOpenAIPKCE() (*OpenAIPKCECodes, error) {
	// Generate 64-byte random data, then convert to 128-character hex string (like Python secrets.token_hex(64))
	randomBytes := make([]byte, 64)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	// Convert to hex string (128 characters) - exactly matching Python secrets.token_hex(64)
	codeVerifier := hex.EncodeToString(randomBytes)

	// Generate SHA256 hash of verifier for code challenge - exactly matching Python
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &OpenAIPKCECodes{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
	}, nil
}

// generateOpenAISecureRandomString creates a cryptographically secure random string
func generateOpenAISecureRandomString(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate secure random string: %w", err)
	}

	result := make([]byte, length)
	for i, b := range randomBytes {
		result[i] = chars[int(b)%len(chars)]
	}

	return string(result), nil
}

// GetAuthorizationURL generates the OAuth authorization URL
func (flow *OpenAIOAuthFlow) GetAuthorizationURL() string {
	params := url.Values{
		"response_type":              {"code"},
		"client_id":                  {flow.ClientID},
		"redirect_uri":               {flow.RedirectURI},
		"scope":                      {openaiRequiredScopes},
		"code_challenge":             {flow.PKCE.CodeChallenge},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"state":                      {flow.State},
	}

	return fmt.Sprintf("%s/oauth/authorize?%s", openaiIssuer, params.Encode())
}

// StartAuthFlow starts the OAuth authentication flow
func (flow *OpenAIOAuthFlow) StartAuthFlow() (*OpenAICredentials, error) {
	// Start HTTP server in background
	go func() {
		if err := flow.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Error("OAuth server error", "error", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Open browser to authorization URL
	authURL := flow.GetAuthorizationURL()
	
	if err := openBrowser(authURL); err != nil {
		logging.Warn("Failed to open browser automatically", "error", err)
		fmt.Printf("Please manually open this URL in your browser:\n%s\n", authURL)
	}

	// Wait for authentication result with timeout
	var authResult OpenAIAuthResult
	select {
	case authResult = <-flow.resultChan:
		if !authResult.Success {
			// Shutdown immediately on error
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			flow.server.Shutdown(ctx)
			return nil, authResult.Error
		}
		// Success case - don't shutdown yet, wait for success page to be served
		
	case <-time.After(10 * time.Minute): // 10 minute timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		flow.server.Shutdown(ctx)
		return nil, fmt.Errorf("authentication timeout after 10 minutes")
	}
	
	// Wait for shutdown signal from success page handler
	select {
	case <-flow.shutdownChan:
		// Success page served, now safe to shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		flow.server.Shutdown(ctx)
		return authResult.Credentials, nil
		
	case <-time.After(30 * time.Second): // Additional timeout for success page
		// Force shutdown if success page takes too long
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		flow.server.Shutdown(ctx)
		return authResult.Credentials, nil
	}
}

// handleCallback handles the OAuth callback from OpenAI
func (flow *OpenAIOAuthFlow) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Validate state parameter for CSRF protection
	if query.Get("state") != flow.State {
		http.Error(w, "State parameter mismatch", http.StatusBadRequest)
		flow.resultChan <- OpenAIAuthResult{Success: false, Error: fmt.Errorf("state mismatch - potential CSRF attack")}
		return
	}

	// Get authorization code
	code := query.Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		flow.resultChan <- OpenAIAuthResult{Success: false, Error: fmt.Errorf("missing authorization code")}
		return
	}

	// Exchange code for tokens and API key
	credentials, successURL, err := flow.exchangeCodeForCredentials(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		flow.resultChan <- OpenAIAuthResult{Success: false, Error: err}
		return
	}

	// Redirect to success page
	http.Redirect(w, r, successURL, http.StatusFound)
	flow.resultChan <- OpenAIAuthResult{Success: true, Credentials: credentials}
}

// handleSuccess serves the authentication success page
func (flow *OpenAIOAuthFlow) handleSuccess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(openaiLoginSuccessHTML))
	
	// Ensure response is fully sent before signaling shutdown
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	
	// Signal that success page has been served - safe to shutdown server
	go func() {
		time.Sleep(100 * time.Millisecond) // Give response time to complete
		select {
		case flow.shutdownChan <- true:
			// Successfully signaled shutdown
		default:
			// Channel already has a value, no need to block
		}
	}()
}

// exchangeCodeForCredentials performs the complete OAuth flow: code → tokens → API key
func (flow *OpenAIOAuthFlow) exchangeCodeForCredentials(code string) (*OpenAICredentials, string, error) {
	// Step 1: Exchange authorization code for OAuth tokens
	tokenData, err := flow.exchangeAuthCode(code)
	if err != nil {
		return nil, "", fmt.Errorf("auth code exchange failed: %w", err)
	}

	// Step 2: Parse ID token to extract claims
	tokenClaims, err := parseOpenAIJWTClaims(tokenData.IDToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse ID token: %w", err)
	}

	// Parse access token claims
	accessClaims, err := parseOpenAIJWTClaims(tokenData.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse access token: %w", err)
	}

	// PROCEED WITH API KEY GENERATION - reference implementation works with organizations array
	
	// Step 3: Fallback - try token exchange with current structure
	apiKey, successURL, err := flow.obtainAPIKey(tokenClaims, accessClaims, tokenData)
	if err != nil {
		return nil, "", fmt.Errorf("API key exchange failed: %w", err)
	}

	credentials := &OpenAICredentials{
		IDToken:      tokenData.IDToken,
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
		APIKey:       apiKey,
		AccountID:    tokenData.AccountID,
		ExpiresAt:    time.Now().Unix() + 3600, // 1 hour default
		ClientID:     flow.ClientID,
		Provider:     "openai",
		LastRefresh:  time.Now().UTC().Format(time.RFC3339),
	}

	return credentials, successURL, nil
}

// exchangeAuthCode exchanges authorization code for OAuth tokens
func (flow *OpenAIOAuthFlow) exchangeAuthCode(code string) (*OpenAICredentials, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {flow.RedirectURI},
		"client_id":     {flow.ClientID},
		"code_verifier": {flow.PKCE.CodeVerifier},
	}

	resp, err := http.PostForm(fmt.Sprintf("%s/oauth/token", openaiIssuer), data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OAuth token exchange failed (status %d). Please ensure you're logged into https://chat.openai.com and try again: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Extract account ID from ID token claims
	claims, err := parseOpenAIJWTClaims(tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	var accountID string
	if authClaims, ok := claims["https://api.openai.com/auth"].(map[string]interface{}); ok {
		if id, ok := authClaims["chatgpt_account_id"].(string); ok {
			accountID = id
		}
	}

	return &OpenAICredentials{
		IDToken:      tokenResp.IDToken,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		AccountID:    accountID,
	}, nil
}


// obtainAPIKey exchanges OAuth tokens for OpenAI API key
func (flow *OpenAIOAuthFlow) obtainAPIKey(tokenClaims, accessClaims map[string]interface{}, tokenData *OpenAICredentials) (string, string, error) {
	authClaims, ok := tokenClaims["https://api.openai.com/auth"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("missing auth claims in ID token - you may need to create an organization and project at https://platform.openai.com first")
	}

	// CRITICAL FIX: Extract organization from either direct field OR organizations array (reference implementation works with array)
	orgID, hasOrgID := authClaims["organization_id"].(string)
	projectID, _ := authClaims["project_id"].(string)

	// If no direct organization_id, extract from organizations array like reference implementation
	if !hasOrgID || orgID == "" {
		if orgs, ok := authClaims["organizations"].([]interface{}); ok && len(orgs) > 0 {
			// Find the default organization first, otherwise use the first one
			for _, org := range orgs {
				if orgMap, ok := org.(map[string]interface{}); ok {
					if isDefault, _ := orgMap["is_default"].(bool); isDefault {
						if id, ok := orgMap["id"].(string); ok {
							orgID = id
							break
						}
					}
				}
			}
			// If no default found, use the first organization
			if orgID == "" && len(orgs) > 0 {
				if orgMap, ok := orgs[0].(map[string]interface{}); ok {
					if id, ok := orgMap["id"].(string); ok {
						orgID = id
					}
				}
			}
		}
	}

	// Require organization but project_id is optional
	if orgID == "" {
		return "", "", fmt.Errorf("no organization found in token - please ensure you have an organization at https://platform.openai.com")
	}

	// Generate API key name with random ID
	randomID, err := generateOpenAISecureRandomString(6)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate API key name: %w", err)
	}
	today := time.Now().UTC().Format("2006-01-02")
	keyName := fmt.Sprintf("Codex CLI [auto-generated] (%s) [%s]", today, randomID)

	
	exchangeData := url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"client_id":          {flow.ClientID},
		"requested_token":    {"openai-api-key"},
		"subject_token":      {tokenData.IDToken}, // Back to ID token
		"subject_token_type": {"urn:ietf:params:oauth:token-type:id_token"}, // ID token type
		"name":               {keyName},
	}

	// Use same approach as working Python implementation - NO EXTRA HEADERS
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth/token", openaiIssuer), strings.NewReader(exchangeData.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("API key exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		
		// Fallback to access token like the reference implementation does
		successURL := fmt.Sprintf("http://localhost:%d/success", openaiRequiredPort)
		return tokenData.AccessToken, successURL, nil
	}

	var exchangeResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&exchangeResp); err != nil {
		return "", "", fmt.Errorf("failed to decode API key response: %w", err)
	}

	// Simple success redirect
	successURL := fmt.Sprintf("http://localhost:%d/success", openaiRequiredPort)

	return exchangeResp.AccessToken, successURL, nil
}

// parseOpenAIJWTClaims parses JWT claims from a token
func parseOpenAIJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format - expected 3 parts, got %d", len(parts))
	}

	// JWT uses base64url encoding without padding (RFC 7515)
	payload := parts[1]
	if payload == "" {
		return nil, fmt.Errorf("empty JWT payload")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload (length %d): %w", len(payload), err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims from decoded payload (length %d): %w", len(decoded), err)
	}

	return claims, nil
}

// parseOpenAIIDTokenInfo extracts ID token information
func parseOpenAIIDTokenInfo(idToken string) (*OpenAIIDTokenInfo, error) {
	claims, err := parseOpenAIJWTClaims(idToken)
	if err != nil {
		return nil, err
	}

	info := &OpenAIIDTokenInfo{}

	if email, ok := claims["email"].(string); ok {
		info.Email = email
	}

	if authClaims, ok := claims["https://api.openai.com/auth"].(map[string]interface{}); ok {
		if planType, ok := authClaims["chatgpt_plan_type"].(string); ok {
			info.ChatGPTPlanType = planType
		}
	}

	return info, nil
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default: // "linux", "freebsd", "openbsd", "netbsd"
		err = exec.Command("xdg-open", url).Start()
	}
	return err
}

// RefreshOpenAIAccessToken refreshes an expired OpenAI access token
func RefreshOpenAIAccessToken(credentials *OpenAICredentials) (*OpenAICredentials, error) {
	if credentials.RefreshToken == "" {
		return nil, errors.New("no refresh token available")
	}

	data := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": credentials.RefreshToken,
		"client_id":     credentials.ClientID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh request data: %w", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/oauth/token", openaiIssuer),
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	expiresAt := time.Now().Unix() + tokenResponse.ExpiresIn

	// Keep existing refresh token if new one not provided
	refreshToken := tokenResponse.RefreshToken
	if refreshToken == "" {
		refreshToken = credentials.RefreshToken
	}

	return &OpenAICredentials{
		IDToken:      tokenResponse.IDToken,
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: refreshToken,
		APIKey:       credentials.APIKey, // Keep existing API key
		AccountID:    credentials.AccountID,
		ExpiresAt:    expiresAt,
		ClientID:     credentials.ClientID,
		Provider:     credentials.Provider,
		LastRefresh:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// Login success HTML page
const openaiLoginSuccessHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Sign into Codex CLI</title>
    <style>
      .container {
        margin: auto;
        height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: white;
        font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      }
      .inner-container {
        width: 400px;
        text-align: center;
      }
      .logo {
        width: 4rem;
        height: 4rem;
        margin: 0 auto 2rem;
        border-radius: 16px;
        border: .5px solid rgba(0, 0, 0, 0.1);
        box-shadow: rgba(0, 0, 0, 0.1) 0px 4px 16px 0px;
        background-color: rgb(255, 255, 255);
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .title {
        font-size: 32px;
        font-weight: 400;
        line-height: 40px;
        color: #0D0D0D;
      }
      .description {
        margin-top: 1rem;
        color: #5D5D5D;
        font-size: 14px;
      }
    </style>
  </head>
  <body>
    <div class="container">
      <div class="inner-container">
        <div class="logo">
          <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="none" viewBox="0 0 32 32">
            <path stroke="#000" stroke-linecap="round" stroke-width="2.484" 
                  d="M22.356 19.797H17.17M9.662 12.29l1.979 3.576a.511.511 0 0 1-.005.504l-1.974 3.409M30.758 16c0 8.15-6.607 14.758-14.758 14.758-8.15 0-14.758-6.607-14.758-14.758C1.242 7.85 7.85 1.242 16 1.242c8.15 0 14.758 6.608 14.758 14.758Z"></path>
          </svg>
        </div>
        <div class="title">Signed in to Codex CLI</div>
        <div class="description">OpenAI authentication successful. You may now close this page.</div>
      </div>
    </div>
  </body>
</html>`
