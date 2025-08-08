package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"mix/internal/llm/provider"
	"mix/internal/logging"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication credentials",
	Long: `Manage authentication credentials for various AI providers.
	
Currently supports:
  - anthropic-claude-pro-max: Claude Code OAuth authentication
  - openai: OpenAI ChatGPT OAuth authentication`,
}

var authAddCmd = &cobra.Command{
	Use:   "add [provider]",
	Short: "Add authentication for a provider",
	Long: `Add authentication credentials for a specific AI provider.

Supported providers:
  - anthropic-claude-pro-max: Authenticate with Claude using OAuth
  - openai: Authenticate with OpenAI using OAuth

Examples:
  mix auth add anthropic-claude-pro-max
  mix auth add openai`,
	Args: cobra.ExactArgs(1),
	RunE: handleAuthAdd,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display the current authentication status for all configured providers.`,
	RunE:  handleAuthStatus,
}

func handleAuthAdd(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	switch providerName {
	case "anthropic-claude-pro-max", "anthropic":
		return handleAnthropicOAuth()
	case "openai":
		return handleOpenAIOAuth()
	default:
		return fmt.Errorf("unsupported provider: %s\n\nSupported providers:\n  - anthropic-claude-pro-max\n  - openai", providerName)
	}
}

func handleAuthStatus(cmd *cobra.Command, args []string) error {
	storage, err := provider.NewCredentialStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize credential storage: %w", err)
	}

	fmt.Println("Authentication Status:")
	fmt.Println("=====================")

	// Check Anthropic OAuth
	creds, err := storage.GetOAuthCredentials("anthropic")
	if err != nil {
		fmt.Printf("❌ Anthropic Claude Pro Max: Error checking credentials (%v)\n", err)
	} else if creds != nil {
		if creds.IsTokenExpired() {
			fmt.Printf("⚠️  Anthropic Claude Pro Max: Token expired, refresh needed\n")
		} else {
			fmt.Printf("✅ Anthropic Claude Pro Max: Authenticated (expires in ~%.0f minutes)\n",
				float64(creds.ExpiresAt-time.Now().Unix())/60)
		}
	} else {
		fmt.Printf("❌ Anthropic Claude Pro Max: Not authenticated\n")
	}

	// Check OpenAI OAuth
	openaiCreds, err := storage.GetOpenAICredentials("openai")
	if err != nil {
		fmt.Printf("❌ OpenAI: Error checking credentials (%v)\n", err)
	} else if openaiCreds != nil {
		if openaiCreds.IsTokenExpired() {
			fmt.Printf("⚠️  OpenAI: Token expired, refresh needed\n")
		} else {
			fmt.Printf("✅ OpenAI: Authenticated (expires in ~%.0f minutes)\n",
				float64(openaiCreds.ExpiresAt-time.Now().Unix())/60)
		}
	} else {
		fmt.Printf("❌ OpenAI: Not authenticated\n")
	}

	fmt.Println("\nTo authenticate:")
	fmt.Println("  mix auth add anthropic-claude-pro-max  # Claude Code OAuth")
	fmt.Println("  mix auth add openai                   # OpenAI OAuth")

	return nil
}

func handleAnthropicOAuth() error {
	fmt.Println("🔐 Authenticating with Claude Code OAuth...")
	fmt.Println()

	// Initialize credential storage
	storage, err := provider.NewCredentialStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize credential storage: %w", err)
	}

	// Check if already authenticated
	existingCreds, err := storage.GetOAuthCredentials("anthropic")
	if err != nil {
		logging.Warn("Error checking existing credentials: %v", err)
	} else if existingCreds != nil && !existingCreds.IsTokenExpired() {
		fmt.Printf("✅ Already authenticated with Claude Code OAuth!\n")
		fmt.Printf("   Token expires in ~%.0f minutes\n",
			float64(existingCreds.ExpiresAt-time.Now().Unix())/60)
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Do you want to re-authenticate? (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Authentication cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Create OAuth flow
	oauthFlow, err := provider.NewOAuthFlow("")
	if err != nil {
		return fmt.Errorf("failed to create OAuth flow: %w", err)
	}

	// Display auth URL and try to open browser
	authURL := oauthFlow.GetAuthorizationURL()
	fmt.Printf("🌐 Opening browser for authentication...\n")
	fmt.Printf("   URL: %s\n", authURL)
	fmt.Println()

	// Important: User must be logged into Claude
	fmt.Printf("⚠️  IMPORTANT: You must be logged into claude.ai in your browser first!\n")
	fmt.Printf("   If you're not logged in, please:\n")
	fmt.Printf("   1. Go to https://claude.ai and log in\n")
	fmt.Printf("   2. Then proceed with the OAuth authorization\n")
	fmt.Println()

	// Try to open browser
	if err := oauthFlow.OpenBrowser(); err != nil {
		fmt.Printf("⚠️  Failed to open browser automatically: %v\n", err)
		fmt.Printf("   Please manually open the URL above in your browser.\n")
	}

	// Instructions for user
	fmt.Println("📋 After authorization:")
	fmt.Println("   1. Complete authentication in your browser")
	fmt.Println("   2. You'll be redirected to a callback URL")
	fmt.Println("   3. Copy the authorization code AND state from the URL")
	fmt.Println("   4. Example URL: https://console.anthropic.com/oauth/code/callback?code=ABC123...&state=XYZ456...")
	fmt.Println("   5. Format the input as: code#state")
	fmt.Println("   6. Example input: ABC123defgh456ijklm#XYZ456defgh789ijklm")
	fmt.Println()

	// Get authorization code from user
	reader := bufio.NewReader(os.Stdin)
	var authCode string
	for {
		fmt.Print("Enter authorization code (format: code#state): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		authCode = strings.TrimSpace(input)
		if authCode != "" {
			break
		}
		fmt.Println("❌ Please enter a valid authorization code.")
	}

	// Exchange code for tokens
	fmt.Println("🔄 Exchanging authorization code for tokens...")
	credentials, err := oauthFlow.ExchangeCodeForTokens(authCode)
	if err != nil {
		fmt.Printf("❌ Token exchange failed: %v\n", err)
		fmt.Println()
		fmt.Println("💡 Troubleshooting:")
		fmt.Println("   - Make sure you copied the entire authorization code")
		fmt.Println("   - Check that the code hasn't expired (they expire quickly)")
		fmt.Println("   - Try the authentication process again")
		fmt.Println()
		fmt.Println("   If the problem persists, you can use an API key instead:")
		fmt.Println("   - Set ANTHROPIC_API_KEY environment variable")
		return err
	}

	// Store credentials
	err = storage.StoreOAuthCredentials(
		"anthropic",
		credentials.AccessToken,
		credentials.RefreshToken,
		credentials.ExpiresAt,
		credentials.ClientID,
	)
	if err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Success message
	fmt.Println()
	fmt.Println("🎉 Authentication successful!")
	fmt.Printf("   ✅ OAuth tokens stored securely\n")
	fmt.Printf("   ⏰ Token expires in ~%.0f minutes\n",
		float64(credentials.ExpiresAt-time.Now().Unix())/60)
	fmt.Printf("   🔄 Automatic refresh enabled\n")
	fmt.Println()
	fmt.Println("You can now use Claude Code OAuth authentication in your requests!")
	fmt.Println("The system will automatically use OAuth when available and fall back to API keys if needed.")

	return nil
}

func handleOpenAIOAuth() error {
	fmt.Println("🔐 Authenticating with OpenAI OAuth...")
	fmt.Println()

	// Initialize credential storage
	storage, err := provider.NewCredentialStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize credential storage: %w", err)
	}

	// Check if already authenticated
	existingCreds, err := storage.GetOpenAICredentials("openai")
	if err != nil {
		logging.Warn("Error checking existing OpenAI credentials: %v", err)
	} else if existingCreds != nil && !existingCreds.IsTokenExpired() {
		fmt.Printf("✅ Already authenticated with OpenAI OAuth!\n")
		fmt.Printf("   Token expires in ~%.0f minutes\n",
			float64(existingCreds.ExpiresAt-time.Now().Unix())/60)
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Do you want to re-authenticate? (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Authentication cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Create OAuth flow
	oauthFlow, err := provider.NewOpenAIOAuthFlow()
	if err != nil {
		return fmt.Errorf("failed to create OAuth flow: %w", err)
	}

	// Important: Prerequisites for OAuth authentication
	fmt.Printf("⚠️  PREREQUISITES - You must complete these steps first:\n")
	fmt.Printf("   1. Log into https://chat.openai.com (ChatGPT Plus/Pro required)\n")
	fmt.Printf("   2. Create an organization at https://platform.openai.com\n")
	fmt.Printf("   3. Create a project within that organization\n")
	fmt.Printf("   \n")
	fmt.Printf("   Without these, authentication will fail with 'missing organization or project ID' error.\n")
	fmt.Println()

	// Start OAuth flow
	fmt.Printf("🌐 Starting OAuth flow...\n")
	fmt.Printf("   A browser window will open for authentication\n")
	fmt.Printf("   The system will automatically handle the callback\n")
	fmt.Println()

	// Start the authentication flow
	credentials, err := oauthFlow.StartAuthFlow()
	if err != nil {
		fmt.Printf("❌ Authentication failed: %v\n", err)
		fmt.Println()
		fmt.Println("💡 Troubleshooting:")
		fmt.Println("   - Make sure you're logged into chat.openai.com")
		fmt.Println("   - Check that you have a ChatGPT Plus or Pro subscription")
		fmt.Println("   - Try refreshing your browser and logging in again")
		fmt.Println()
		fmt.Println("   If the problem persists, you can use an API key instead:")
		fmt.Println("   - Set OPENAI_API_KEY environment variable")
		return err
	}

	// Store credentials
	err = storage.StoreOpenAICredentials("openai", credentials)
	if err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Success message
	fmt.Println()
	fmt.Println("🎉 Authentication successful!")
	fmt.Printf("   ✅ OAuth tokens stored securely\n")
	fmt.Printf("   🔑 OpenAI API key generated and stored\n")
	fmt.Printf("   ⏰ Token expires in ~%.0f minutes\n",
		float64(credentials.ExpiresAt-time.Now().Unix())/60)
	fmt.Printf("   🔄 Automatic refresh enabled\n")
	fmt.Println()
	fmt.Println("You can now use OpenAI OAuth authentication in your requests!")
	fmt.Println("The system will automatically use OAuth when available and fall back to API keys if needed.")

	return nil
}

func init() {
	// Add auth subcommands
	authCmd.AddCommand(authAddCmd)
	authCmd.AddCommand(authStatusCmd)
}
