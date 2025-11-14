package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/credentials"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/logging"
	"mix/internal/message"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicOptions struct {
	useBedrock             bool
	disableCache           bool
	thinkingBudget         func(userMessage string) int
	explicitThinkingBudget *int
	useOAuth               bool
	oauthCreds             *OAuthCredentials
	useInterleavedThinking bool
}

type AnthropicOption func(*anthropicOptions)

type anthropicClient struct {
	providerOptions   providerClientOptions
	options           anthropicOptions
	client            anthropic.Client
	credentialStorage *CredentialStorage
}

type AnthropicClient interfaces.ProviderClient

func newAnthropicClient(opts providerClientOptions) AnthropicClient {
	anthropicOpts := anthropicOptions{
		useInterleavedThinking: true, // Enable by default
	}
	for _, o := range opts.anthropicOptions {
		o(&anthropicOpts)
	}

	// Get credentials service
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		logging.Warn("Credentials service unavailable")
	}

	// Check for OAuth credentials first (highest priority)
	var oauthCreds *OAuthCredentials // Use old format for internal client compatibility
	if credentialsService != nil {
		ctx := context.Background() // Create context for database operations
		creds, err := credentialsService.GetOAuthCredentials(ctx, "anthropic")
		if err != nil && !errors.Is(err, ErrOAuthCredentialNotFound) {
			logging.Warn("Failed to get OAuth credentials", "error", err)
		} else if err == nil {
			// Convert from database format to client format
			oauthCreds = &OAuthCredentials{
				AccessToken:  creds.AccessToken,
				RefreshToken: creds.RefreshToken,
				ExpiresAt:    creds.ExpiresAt,
				ClientID:     creds.ClientID,
				Provider:     creds.Provider,
			}

			// Check if token needs refresh
			if oauthCreds.IsTokenExpired() && oauthCreds.RefreshToken != "" {
				logging.Info("OAuth token expired, attempting refresh...")
				if refreshedCreds, err := RefreshAccessToken(oauthCreds); err == nil {
					// Store refreshed credentials in database
					newCreds := &credentials.OAuthCredentials{
						AccessToken:  refreshedCreds.AccessToken,
						RefreshToken: refreshedCreds.RefreshToken,
						ExpiresAt:    refreshedCreds.ExpiresAt,
						ClientID:     refreshedCreds.ClientID,
						Provider:     "anthropic",
					}
					if err := credentialsService.StoreOAuthCredentials(ctx, "anthropic", newCreds); err != nil {
						logging.Warn("Failed to store refreshed OAuth credentials", "error", err)
					}
					oauthCreds = refreshedCreds
				} else {
					logging.Warn("Failed to refresh OAuth token: %v", err)
				}
			}
		}
	}

	anthropicClientOptions := []option.RequestOption{}

	// Set up authentication - prioritize OAuth over database API key
	switch {
	case oauthCreds != nil:
		anthropicOpts.useOAuth = true
		anthropicOpts.oauthCreds = oauthCreds
		anthropicClientOptions = append(anthropicClientOptions, option.WithAuthToken(oauthCreds.AccessToken))
	case opts.apiKey != "":
		// Use database API key (passed in opts.apiKey from caller)
		anthropicClientOptions = append(anthropicClientOptions, option.WithAPIKey(opts.apiKey))
		logging.Info("Initialized Anthropic client with database API key authentication")
	default:
		// No authentication available - check database directly as last resort
		if config.GetAPICredentials() != nil {
			ctx := context.Background()
			dbKey, err := config.GetAPICredentials().GetAPIKey(ctx, models.ProviderAnthropic)
			if err == nil && dbKey != "" {
				anthropicClientOptions = append(anthropicClientOptions, option.WithAPIKey(dbKey))
			} else {
				// No valid credentials found - use placeholder for command handling only
				logging.Warn("No authentication method available for Anthropic - neither OAuth nor database API key")
				// Using a placeholder API key to allow the client to initialize
				// This will allow /login and other non-API commands to work
				anthropicClientOptions = append(anthropicClientOptions, option.WithAPIKey("placeholder-for-initialization-only"))
			}
		} else {
			// No credentials service available
			logging.Warn("No authentication method available for Anthropic - credentials service not available")
			// Using a placeholder API key to allow the client to initialize
			// This will allow /login and other non-API commands to work
			anthropicClientOptions = append(anthropicClientOptions, option.WithAPIKey("placeholder-for-initialization-only"))
		}
	}

	if anthropicOpts.useBedrock {
		anthropicClientOptions = append(anthropicClientOptions, bedrock.WithLoadDefaultConfig(context.Background()))
	}

	// Add request timeout to prevent indefinite hangs
	// Set to 15 minutes to allow long-running tool executions (e.g., Bash commands, MCP tools, sub-agents)
	anthropicClientOptions = append(anthropicClientOptions, option.WithRequestTimeout(15*time.Minute))

	anthropicClient := &anthropicClient{
		providerOptions:   opts,
		options:           anthropicOpts,
		credentialStorage: nil, // No longer using file-based storage
	}

	// Add beta headers if needed
	if betaHeader := anthropicClient.buildBetaHeader(); betaHeader != "" {
		anthropicClientOptions = append(anthropicClientOptions, option.WithHeader("anthropic-beta", betaHeader))
	}

	anthropicClient.client = anthropic.NewClient(anthropicClientOptions...)
	return anthropicClient
}

func (a *anthropicClient) convertMessages(messages []message.Message) (anthropicMessages []anthropic.MessageParam) {
	for i, msg := range messages {
		cache := false
		if i > len(messages)-3 {
			cache = true
		}
		switch msg.Role {
		case message.User:
			content := anthropic.NewTextBlock(msg.Content().String())
			if cache && !a.options.disableCache {
				content.OfText.CacheControl = anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				}
			}
			var contentBlocks []anthropic.ContentBlockParamUnion
			contentBlocks = append(contentBlocks, content)
			for _, binaryContent := range msg.BinaryContent() {
				base64Image := binaryContent.String(models.ProviderAnthropic)
				imageBlock := anthropic.NewImageBlockBase64(binaryContent.MIMEType, base64Image)
				contentBlocks = append(contentBlocks, imageBlock)
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(contentBlocks...))

		case message.Assistant:
			blocks := []anthropic.ContentBlockParamUnion{}

			// Add thinking blocks first (must be in sequence)
			for _, thinkingBlock := range msg.ThinkingBlocks() {
				blocks = append(blocks, anthropic.NewThinkingBlock(thinkingBlock.Signature, thinkingBlock.Thinking))
			}

			// Add redacted thinking blocks
			for _, redactedBlock := range msg.RedactedThinkingBlocks() {
				blocks = append(blocks, anthropic.NewRedactedThinkingBlock(redactedBlock.Data))
			}

			// Add text content
			if msg.Content().String() != "" {
				content := anthropic.NewTextBlock(msg.Content().String())
				if cache && !a.options.disableCache {
					content.OfText.CacheControl = anthropic.CacheControlEphemeralParam{
						Type: "ephemeral",
					}
				}
				blocks = append(blocks, content)
			}

			// Add tool calls
			for _, toolCall := range msg.ToolCalls() {
				var inputMap map[string]any
				err := json.Unmarshal([]byte(toolCall.Input), &inputMap)
				if err != nil {
					continue
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(toolCall.ID, inputMap, toolCall.Name))
			}

			if len(blocks) == 0 {
				continue
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(blocks...))

		case message.Tool:
			results := make([]anthropic.ContentBlockParamUnion, len(msg.ToolResults()))
			for i, toolResult := range msg.ToolResults() {
				results[i] = anthropic.NewToolResultBlock(toolResult.ToolCallID, toolResult.Content, toolResult.IsError)
			}
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(results...))
		}
	}
	return
}

func (a *anthropicClient) convertTools(tools []interfaces.BaseTool) []anthropic.ToolUnionParam {
	anthropicTools := make([]anthropic.ToolUnionParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		toolParam := anthropic.ToolParam{
			Name:        info.Name,
			Description: anthropic.String(info.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: info.Parameters,
				// TODO: figure out how we can tell claude the required fields?
			},
		}

		if i == len(tools)-1 && !a.options.disableCache {
			toolParam.CacheControl = anthropic.CacheControlEphemeralParam{
				Type: "ephemeral",
			}
		}

		anthropicTools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	return anthropicTools
}

func (a *anthropicClient) finishReason(reason string) message.FinishReason {
	switch reason {
	case "end_turn":
		return message.FinishReasonEndTurn
	case "max_tokens":
		return message.FinishReasonMaxTokens
	case "tool_use":
		return message.FinishReasonToolUse
	case "stop_sequence":
		return message.FinishReasonEndTurn
	default:
		return message.FinishReasonUnknown
	}
}

func (a *anthropicClient) preparedMessages(messages []anthropic.MessageParam, tools []anthropic.ToolUnionParam) anthropic.MessageNewParams {
	var thinkingParam anthropic.ThinkingConfigParamUnion
	lastMessage := messages[len(messages)-1]
	isUser := lastMessage.Role == anthropic.MessageParamRoleUser
	messageContent := ""
	temperature := anthropic.Float(0)

	// Extract message content for thinking budget calculation
	if isUser {
		for _, m := range lastMessage.Content {
			if m.OfText != nil && m.OfText.Text != "" {
				messageContent = m.OfText.Text
			}
		}
	}

	// Enable thinking based on explicit budget or budget function - but ensure API compatibility
	if a.options.thinkingBudget != nil {
		tokenBudget := 0 // Default to disabled

		// Check for explicit budget override FIRST
		if a.options.explicitThinkingBudget != nil {
			tokenBudget = *a.options.explicitThinkingBudget
			logging.Debug("Using explicit thinking budget", "tokenBudget", tokenBudget)
		} else if messageContent != "" {
			// Fall back to keyword detection
			tokenBudget = a.options.thinkingBudget(messageContent)
			logging.Debug("Using keyword-based thinking budget", "tokenBudget", tokenBudget)
		}

		// Check if conversation history contains thinking blocks
		hasThinkingInHistory := false
		for _, msg := range messages {
			if msg.Role == anthropic.MessageParamRoleAssistant {
				for _, content := range msg.Content {
					if content.OfThinking != nil || content.OfRedactedThinking != nil {
						hasThinkingInHistory = true
						break
					}
				}
				if hasThinkingInHistory {
					break
				}
			}
		}

		switch {
		case tokenBudget > 0:
			thinkingParam = anthropic.ThinkingConfigParamOfEnabled(int64(tokenBudget))
			temperature = anthropic.Float(1)
			logging.Debug("Thinking enabled for Anthropic API", "tokenBudget", tokenBudget)
		case hasThinkingInHistory:
			// Enable with minimal budget for API compatibility
			thinkingParam = anthropic.ThinkingConfigParamOfEnabled(1024)
			temperature = anthropic.Float(1)
			logging.Debug("Thinking enabled for API compatibility", "tokenBudget", 1024)
		default:
			logging.Debug("Thinking disabled - no budget provided and no thinking in history")
		}
	} else {
		logging.Debug("No thinking budget function - thinking disabled")
	}

	// Determine system message based on authentication method
	systemMessage := a.providerOptions.systemMessage
	if a.options.useOAuth {
		// REQUIRED: Use Claude Code system prompt for OAuth
		systemMessage = "You are Claude Code, Anthropic's official CLI for Claude."

		// If the original system message was different, inject it as role context
		// This implements the role injection pattern from the reference manual
		if a.providerOptions.systemMessage != systemMessage && a.providerOptions.systemMessage != "" {
			roleInjectionMsg := fmt.Sprintf("For this conversation, please act as: %s", a.providerOptions.systemMessage)

			// Inject role at the beginning of the conversation if not already present
			if len(messages) == 0 || !strings.Contains(messages[0].Content[0].OfText.Text, "For this conversation, please act as:") {
				roleContent := anthropic.NewTextBlock(roleInjectionMsg)
				roleMessage := anthropic.NewUserMessage(roleContent)

				// Add acknowledgment message
				ackContent := anthropic.NewTextBlock("Understood. I'll act in that role for our conversation.")
				ackMessage := anthropic.NewAssistantMessage(ackContent)

				// Prepend role injection messages
				messages = append([]anthropic.MessageParam{roleMessage, ackMessage}, messages...)
			}
		}
	}

	return anthropic.MessageNewParams{
		Model:       anthropic.Model(a.providerOptions.model.APIModel),
		MaxTokens:   a.providerOptions.maxTokens,
		Temperature: temperature,
		Messages:    messages,
		Tools:       tools,
		Thinking:    thinkingParam,
		System: []anthropic.TextBlockParam{
			{
				Text: systemMessage,
				CacheControl: anthropic.CacheControlEphemeralParam{
					Type: "ephemeral",
				},
			},
		},
	}
}

func (a *anthropicClient) Send(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) (resposne *interfaces.ProviderResponse, err error) {
	// Handle proactive token refresh for OAuth
	if a.options.useOAuth && a.options.oauthCreds != nil {
		if a.options.oauthCreds.IsTokenExpired() && a.options.oauthCreds.RefreshToken != "" {
			if refreshedCreds, err := RefreshAccessToken(a.options.oauthCreds); err == nil {
				// Update stored credentials
				if a.credentialStorage != nil {
					if err := a.credentialStorage.StoreOAuthCredentials(
						"anthropic",
						refreshedCreds.AccessToken,
						refreshedCreds.RefreshToken,
						refreshedCreds.ExpiresAt,
						refreshedCreds.ClientID,
					); err != nil {
						logging.Warn("Failed to store refreshed OAuth credentials", "error", err)
					}
				}
				a.options.oauthCreds = refreshedCreds

				// Update client with new token
				a.recreateClient()
				logging.Info("Refreshed OAuth token proactively")
			}
		}
	}

	// Use SDK for both OAuth and API key authentication
	preparedMessages := a.preparedMessages(a.convertMessages(messages), a.convertTools(tools))

	attempts := 0
	for {
		attempts++
		anthropicResponse, err := a.client.Messages.New(
			ctx,
			preparedMessages,
		)
		// If there is an error we are going to see if we can retry the call
		if err != nil {
			logging.Error("Error in Anthropic API call", "error", err)

			// Check for authentication errors (401)
			if strings.Contains(err.Error(), "401") {
				// Check if using placeholder auth (indicating no real auth provided)
				if !a.options.useOAuth && a.providerOptions.apiKey == "" {
					// Return a proper authentication error that will be handled by the error path
					return nil, errors.New("authentication_error: Authentication required. Please use /login command to authenticate")
				}

				// Try OAuth token refresh if available
				if a.options.useOAuth && a.options.oauthCreds != nil && a.options.oauthCreds.RefreshToken != "" {
					if refreshedCreds, refreshErr := RefreshAccessToken(a.options.oauthCreds); refreshErr == nil {
						// Update stored credentials
						if a.credentialStorage != nil {
							if err := a.credentialStorage.StoreOAuthCredentials(
								"anthropic",
								refreshedCreds.AccessToken,
								refreshedCreds.RefreshToken,
								refreshedCreds.ExpiresAt,
								refreshedCreds.ClientID,
							); err != nil {
								logging.Warn("Failed to store refreshed OAuth credentials", "error", err)
							}
						}
						a.options.oauthCreds = refreshedCreds

						// Update client with new token and retry
						a.recreateClient()
						logging.Info("Refreshed OAuth token and retrying request")
						continue
					}
				}
			}

			retry, after, retryErr := a.shouldRetry(attempts, err)
			if retryErr != nil {
				// For authentication errors, provide a friendly message
				if strings.Contains(retryErr.Error(), "401") {
					return &interfaces.ProviderResponse{
						Content: "⚠️ Authentication failed. Please use /login command to authenticate with Claude using an API key.\n\n" +
							"To login:\n" +
							"1. Visit https://console.anthropic.com/settings/keys\n" +
							"2. Create an API key\n" +
							"3. Use the /login command to authenticate",
						Usage: interfaces.TokenUsage{},
					}, nil
				}
				return nil, retryErr
			}
			if retry {
				logging.Warn(a.getRetryMessage(err, attempts))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			return nil, retryErr
		}

		content := ""
		for i := range anthropicResponse.Content {
			if text, ok := anthropicResponse.Content[i].AsAny().(anthropic.TextBlock); ok {
				content += text.Text
			}
		}

		return &interfaces.ProviderResponse{
			Content:                content,
			ToolCalls:              a.toolCalls(*anthropicResponse),
			Usage:                  a.usage(*anthropicResponse),
			ThinkingBlocks:         a.extractThinkingBlocks(*anthropicResponse),
			RedactedThinkingBlocks: a.extractRedactedThinkingBlocks(*anthropicResponse),
		}, nil
	}
}

func (a *anthropicClient) Stream(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) <-chan interfaces.ProviderEvent {
	eventChan := make(chan interfaces.ProviderEvent)

	a.proactiveRefreshOAuthToken()

	preparedMessages := a.preparedMessages(a.convertMessages(messages), a.convertTools(tools))

	if !a.options.useOAuth && a.providerOptions.apiKey == "" {
		a.sendAuthenticationError(eventChan)
		return eventChan
	}

	go a.streamWithRetry(ctx, preparedMessages, eventChan)
	return eventChan
}

func (a *anthropicClient) proactiveRefreshOAuthToken() {
	if a.options.useOAuth && a.options.oauthCreds != nil {
		if a.options.oauthCreds.IsTokenExpired() && a.options.oauthCreds.RefreshToken != "" {
			if refreshedCreds, err := RefreshAccessToken(a.options.oauthCreds); err == nil {
				a.storeRefreshedCredentials(refreshedCreds)
				a.options.oauthCreds = refreshedCreds
				a.recreateClient()
				logging.Info("Refreshed OAuth token proactively for streaming")
			}
		}
	}
}

func (a *anthropicClient) storeRefreshedCredentials(creds *OAuthCredentials) {
	if a.credentialStorage != nil {
		if err := a.credentialStorage.StoreOAuthCredentials(
			"anthropic",
			creds.AccessToken,
			creds.RefreshToken,
			creds.ExpiresAt,
			creds.ClientID,
		); err != nil {
			logging.Warn("Failed to store refreshed OAuth credentials", "error", err)
		}
	}
}

func (a *anthropicClient) sendAuthenticationError(eventChan chan interfaces.ProviderEvent) {
	go func() {
		authErrMsg := "authentication_error: Authentication required. Please use /login command to authenticate"
		eventChan <- interfaces.ProviderEvent{
			Type:  interfaces.EventError,
			Error: errors.New(authErrMsg),
		}
		close(eventChan)
	}()
}

func (a *anthropicClient) streamWithRetry(ctx context.Context, preparedMessages anthropic.MessageNewParams, eventChan chan interfaces.ProviderEvent) {
	attempts := 0

	for {
		attempts++
		anthropicStream := a.client.Messages.NewStreaming(ctx, preparedMessages)
		accumulatedMessage := anthropic.Message{}

		if a.isContextCancelled(ctx, eventChan) {
			return
		}

		activeToolCalls := make(map[int]*message.ToolCall)
		for anthropicStream.Next() {
			event := anthropicStream.Current()
			if err := accumulatedMessage.Accumulate(event); err != nil {
				logging.Warn("Error accumulating message", "error", err)
				continue
			}

			switch evt := event.AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
				a.handleContentBlockStart(evt, activeToolCalls, eventChan)
			case anthropic.ContentBlockDeltaEvent:
				a.handleContentBlockDelta(evt, activeToolCalls, eventChan)
			case anthropic.ContentBlockStopEvent:
				a.handleContentBlockStop(evt, activeToolCalls, eventChan)
			case anthropic.MessageStopEvent:
				a.handleMessageStop(accumulatedMessage, eventChan)
			}

			if a.isContextCancelled(ctx, eventChan) {
				return
			}
		}

		err := anthropicStream.Err()
		if err == nil || errors.Is(err, io.EOF) {
			close(eventChan)
			return
		}

		if a.tryRefreshOAuthOnError(err) {
			continue
		}

		if !a.handleRetryLogic(ctx, attempts, err, eventChan) {
			return
		}
	}
}

func (a *anthropicClient) isContextCancelled(ctx context.Context, eventChan chan interfaces.ProviderEvent) bool {
	select {
	case <-ctx.Done():
		eventChan <- interfaces.ProviderEvent{Type: interfaces.EventError, Error: ctx.Err()}
		close(eventChan)
		return true
	default:
		return false
	}
}

func (a *anthropicClient) handleContentBlockStart(event anthropic.ContentBlockStartEvent, activeToolCalls map[int]*message.ToolCall, eventChan chan interfaces.ProviderEvent) {
	switch event.ContentBlock.Type {
	case "text":
		eventChan <- interfaces.ProviderEvent{Type: interfaces.EventContentStart}
	case "tool_use":
		toolCall := &message.ToolCall{
			ID:       event.ContentBlock.ID,
			Name:     event.ContentBlock.Name,
			Finished: false,
		}
		activeToolCalls[int(event.Index)] = toolCall
		eventChan <- interfaces.ProviderEvent{
			Type:     interfaces.EventToolUseStart,
			ToolCall: toolCall,
		}
	}
}

func (a *anthropicClient) handleContentBlockDelta(event anthropic.ContentBlockDeltaEvent, activeToolCalls map[int]*message.ToolCall, eventChan chan interfaces.ProviderEvent) {
	switch {
	case event.Delta.Type == "thinking_delta" && event.Delta.Thinking != "":
		eventChan <- interfaces.ProviderEvent{
			Type:     interfaces.EventThinkingDelta,
			Thinking: event.Delta.Thinking,
		}
	case event.Delta.Type == "text_delta" && event.Delta.Text != "":
		eventChan <- interfaces.ProviderEvent{
			Type:    interfaces.EventContentDelta,
			Content: event.Delta.Text,
		}
	case event.Delta.Type == "input_json_delta":
		if toolCall, exists := activeToolCalls[int(event.Index)]; exists {
			deltaInput := event.Delta.JSON.PartialJSON.Raw()

			// PartialJSON.Raw() returns a JSON-encoded string (e.g., "\"hello\"" for the string "hello")
			// We need to unmarshal it to get the actual string content before concatenation
			var unquotedDelta string
			if err := json.Unmarshal([]byte(deltaInput), &unquotedDelta); err != nil {
				// If unmarshal fails, log the details and send the raw delta (fallback to old behavior)
				logging.Error("Failed to unmarshal tool parameter delta, using raw value as fallback",
					"toolCallID", toolCall.ID,
					"deltaRaw", deltaInput,
					"deltaBytes", []byte(deltaInput),
					"error", err.Error())

				// Fallback: send the raw value (old behavior) to avoid breaking the stream
				unquotedDelta = deltaInput
			}

			eventChan <- interfaces.ProviderEvent{
				Type: interfaces.EventToolUseDelta,
				ToolCall: &message.ToolCall{
					ID:       toolCall.ID,
					Finished: false,
					Input:    unquotedDelta, // Send the unquoted string (or raw if unmarshal failed)
				},
			}
		}
	}
}

func (a *anthropicClient) handleContentBlockStop(event anthropic.ContentBlockStopEvent, activeToolCalls map[int]*message.ToolCall, eventChan chan interfaces.ProviderEvent) {
	if toolCall, exists := activeToolCalls[int(event.Index)]; exists {
		eventChan <- interfaces.ProviderEvent{
			Type: interfaces.EventToolUseStop,
			ToolCall: &message.ToolCall{
				ID: toolCall.ID,
			},
		}
		delete(activeToolCalls, int(event.Index))
	} else {
		eventChan <- interfaces.ProviderEvent{Type: interfaces.EventContentStop}
	}
}

func (a *anthropicClient) handleMessageStop(accumulatedMessage anthropic.Message, eventChan chan interfaces.ProviderEvent) {
	content := ""
	for i := range accumulatedMessage.Content {
		if text, ok := accumulatedMessage.Content[i].AsAny().(anthropic.TextBlock); ok {
			content += text.Text
		}
	}

	eventChan <- interfaces.ProviderEvent{
		Type: interfaces.EventComplete,
		Response: &interfaces.ProviderResponse{
			Content:                content,
			ToolCalls:              a.toolCalls(accumulatedMessage),
			Usage:                  a.usage(accumulatedMessage),
			FinishReason:           a.finishReason(string(accumulatedMessage.StopReason)),
			ThinkingBlocks:         a.extractThinkingBlocks(accumulatedMessage),
			RedactedThinkingBlocks: a.extractRedactedThinkingBlocks(accumulatedMessage),
		},
	}
}

func (a *anthropicClient) tryRefreshOAuthOnError(err error) bool {
	if a.options.useOAuth && a.options.oauthCreds != nil && strings.Contains(err.Error(), "401") && a.options.oauthCreds.RefreshToken != "" {
		if refreshedCreds, refreshErr := RefreshAccessToken(a.options.oauthCreds); refreshErr == nil {
			a.storeRefreshedCredentials(refreshedCreds)
			a.options.oauthCreds = refreshedCreds
			a.recreateClient()
			logging.Info("Refreshed OAuth token and retrying streaming request")
			return true
		}
	}
	return false
}

func (a *anthropicClient) handleRetryLogic(ctx context.Context, attempts int, err error, eventChan chan interfaces.ProviderEvent) bool {
	retry, after, retryErr := a.shouldRetry(attempts, err)
	if retryErr != nil {
		eventChan <- interfaces.ProviderEvent{Type: interfaces.EventError, Error: retryErr}
		close(eventChan)
		return false
	}

	if retry {
		logging.Warn(a.getRetryMessage(err, attempts))
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				eventChan <- interfaces.ProviderEvent{Type: interfaces.EventError, Error: ctx.Err()}
			}
			close(eventChan)
			return false
		case <-time.After(time.Duration(after) * time.Millisecond):
			return true
		}
	}

	if ctx.Err() != nil {
		eventChan <- interfaces.ProviderEvent{Type: interfaces.EventError, Error: ctx.Err()}
	}

	close(eventChan)
	return false
}

// errorType represents different types of LLM API errors for better messaging
type errorType string

const (
	errorTypeOverloaded  errorType = "overloaded"
	errorTypeRateLimit   errorType = "rate_limit"
	errorTypeUnavailable errorType = "unavailable"
	errorTypeUnretryable errorType = "unretryable"
)

// detectRetryableError analyzes the error and returns whether it's retryable and its type
func (a *anthropicClient) detectRetryableError(err error) (bool, errorType) {
	var apierr *anthropic.Error

	// Check for HTTP status codes first
	if errors.As(err, &apierr) {
		switch apierr.StatusCode {
		case http.StatusTooManyRequests:
			return true, errorTypeRateLimit
		case 529:
			return true, errorTypeUnavailable
		}
	}

	// Check for Anthropic API error types in response body
	errStr := err.Error()
	if strings.Contains(errStr, `"type":"overloaded_error"`) ||
		strings.Contains(errStr, `"message":"Overloaded"`) {
		return true, errorTypeOverloaded
	}

	if strings.Contains(errStr, `"type":"rate_limit_error"`) ||
		strings.Contains(errStr, "rate_limit_error") {
		return true, errorTypeRateLimit
	}

	return false, errorTypeUnretryable
}

// getRetryMessage returns a descriptive retry message based on the error type
func (a *anthropicClient) getRetryMessage(err error, attempts int) string {
	_, errType := a.detectRetryableError(err)

	var operation string
	switch errType {
	case errorTypeOverloaded:
		operation = "LLM API temporarily overloaded"
	case errorTypeRateLimit:
		operation = "LLM API rate limited"
	case errorTypeUnavailable:
		operation = "LLM API temporarily unavailable"
	default:
		operation = "LLM API error"
	}

	return fmt.Sprintf("Retrying due to %s... attempt %d of %d", operation, attempts, maxRetries)
}

func (a *anthropicClient) shouldRetry(attempts int, err error) (shouldRetry bool, retryAfterMs int64, retryErr error) {
	// Use enhanced error detection
	retryable, errType := a.detectRetryableError(err)
	if !retryable {
		return false, 0, err
	}

	if attempts > maxRetries {
		var errorTypeMsg string
		switch errType {
		case errorTypeOverloaded:
			errorTypeMsg = "LLM API overloaded"
		case errorTypeRateLimit:
			errorTypeMsg = "LLM API rate limited"
		case errorTypeUnavailable:
			errorTypeMsg = "LLM API unavailable"
		default:
			errorTypeMsg = "LLM API error"
		}
		return false, 0, fmt.Errorf("%s - maximum retry attempts reached: %d retries", errorTypeMsg, maxRetries)
	}

	// Calculate retry delay with exponential backoff
	var retryMs int
	var apierr *anthropic.Error

	// Try to get Retry-After header for HTTP status code errors
	if errors.As(err, &apierr) && apierr.Response != nil {
		retryAfterValues := apierr.Response.Header.Values("Retry-After")
		if len(retryAfterValues) > 0 {
			if _, parseErr := fmt.Sscanf(retryAfterValues[0], "%d", &retryMs); parseErr == nil {
				retryMs *= 1000 // Convert to milliseconds
				return true, int64(retryMs), nil
			}
		}
	}

	// Use exponential backoff with jitter
	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs = backoffMs + jitterMs

	return true, int64(retryMs), nil
}

func (a *anthropicClient) toolCalls(msg anthropic.Message) []message.ToolCall {
	var toolCalls []message.ToolCall

	for i := range msg.Content {
		if variant, ok := msg.Content[i].AsAny().(anthropic.ToolUseBlock); ok {
			toolCall := message.ToolCall{
				ID:       variant.ID,
				Name:     variant.Name,
				Input:    string(variant.Input),
				Type:     string(variant.Type),
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (a *anthropicClient) usage(msg anthropic.Message) interfaces.TokenUsage {
	return interfaces.TokenUsage{
		InputTokens:         msg.Usage.InputTokens,
		OutputTokens:        msg.Usage.OutputTokens,
		CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
		CacheReadTokens:     msg.Usage.CacheReadInputTokens,
	}
}

// extractThinkingBlocks extracts thinking blocks from Anthropic response
func (a *anthropicClient) extractThinkingBlocks(msg anthropic.Message) []message.ThinkingBlockContent {
	var thinkingBlocks []message.ThinkingBlockContent

	for i := range msg.Content {
		if variant, ok := msg.Content[i].AsAny().(anthropic.ThinkingBlock); ok {
			thinkingBlocks = append(thinkingBlocks, message.ThinkingBlockContent{
				Thinking:  variant.Thinking,
				Signature: variant.Signature,
			})
		}
	}

	return thinkingBlocks
}

// extractRedactedThinkingBlocks extracts redacted thinking blocks from Anthropic response
func (a *anthropicClient) extractRedactedThinkingBlocks(msg anthropic.Message) []message.RedactedThinkingContent {
	var redactedBlocks []message.RedactedThinkingContent

	for i := range msg.Content {
		if variant, ok := msg.Content[i].AsAny().(anthropic.RedactedThinkingBlock); ok {
			redactedBlocks = append(redactedBlocks, message.RedactedThinkingContent{
				Data: variant.Data,
			})
		}
	}

	return redactedBlocks
}

func WithAnthropicBedrock(useBedrock bool) AnthropicOption {
	return func(options *anthropicOptions) {
		options.useBedrock = useBedrock
	}
}

func WithAnthropicDisableCache() AnthropicOption {
	return func(options *anthropicOptions) {
		options.disableCache = true
	}
}

func DefaultThinkingBudgetFn(s string) int {
	content := strings.ToLower(s)

	// Level 1: 31999 tokens - Check longest phrases first
	if strings.Contains(content, "think harder") ||
		strings.Contains(content, "think intensely") ||
		strings.Contains(content, "think longer") ||
		strings.Contains(content, "think really hard") ||
		strings.Contains(content, "think super hard") ||
		strings.Contains(content, "think very hard") ||
		strings.Contains(content, "ultrathink") {
		return 31999
	}

	// Level 2: 10000 tokens
	if strings.Contains(content, "think about it") ||
		strings.Contains(content, "think a lot") ||
		strings.Contains(content, "think deeply") ||
		strings.Contains(content, "think hard") ||
		strings.Contains(content, "think more") ||
		strings.Contains(content, "megathink") {
		return 10000
	}

	// Level 3: 4000 tokens
	if strings.Contains(content, "think") {
		return 4000
	}

	// No thinking
	return 0
}

func WithAnthropicThinkingBudgetFn(fn func(string) int) AnthropicOption {
	return func(options *anthropicOptions) {
		options.thinkingBudget = fn
	}
}

func WithExplicitThinkingBudget(budget *int) AnthropicOption {
	return func(options *anthropicOptions) {
		options.explicitThinkingBudget = budget
	}
}

func WithAnthropicInterleavedThinking() AnthropicOption {
	return func(options *anthropicOptions) {
		options.useInterleavedThinking = true
	}
}

func (a *anthropicClient) buildBetaHeader() string {
	var features []string
	if a.options.useOAuth {
		features = append(features, "oauth-2025-04-20")
	}
	if a.options.useInterleavedThinking {
		features = append(features, "interleaved-thinking-2025-05-14")
	}
	return strings.Join(features, ",")
}

func (a *anthropicClient) recreateClient() {
	var clientOptions []option.RequestOption

	if a.options.useOAuth && a.options.oauthCreds != nil {
		clientOptions = append(clientOptions, option.WithAuthToken(a.options.oauthCreds.AccessToken))
	} else if a.providerOptions.apiKey != "" {
		clientOptions = append(clientOptions, option.WithAPIKey(a.providerOptions.apiKey))
	}

	if betaHeader := a.buildBetaHeader(); betaHeader != "" {
		clientOptions = append(clientOptions, option.WithHeader("anthropic-beta", betaHeader))
	}

	if a.options.useBedrock {
		clientOptions = append(clientOptions, bedrock.WithLoadDefaultConfig(context.Background()))
	}

	// 15-minute timeout for long-running tool executions
	clientOptions = append(clientOptions, option.WithRequestTimeout(15*time.Minute))
	a.client = anthropic.NewClient(clientOptions...)
}
