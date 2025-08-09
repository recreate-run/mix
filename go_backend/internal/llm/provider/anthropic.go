package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"mix/internal/config"
	"mix/internal/llm/models"
	toolsPkg "mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicOptions struct {
	useBedrock             bool
	disableCache          bool
	thinkingBudget        func(userMessage string) int
	useOAuth              bool
	oauthCreds            *OAuthCredentials
	useInterleavedThinking bool
}

type AnthropicOption func(*anthropicOptions)

type anthropicClient struct {
	providerOptions   providerClientOptions
	options           anthropicOptions
	client            anthropic.Client
	credentialStorage *CredentialStorage
}

type AnthropicClient ProviderClient

func newAnthropicClient(opts providerClientOptions) AnthropicClient {
	anthropicOpts := anthropicOptions{
		useInterleavedThinking: true, // Enable by default
	}
	for _, o := range opts.anthropicOptions {
		o(&anthropicOpts)
	}

	// Initialize credential storage
	credStorage, err := NewCredentialStorage()
	if err != nil {
		logging.Warn("Failed to initialize OAuth credential storage: %v", err)
	}

	// Check for OAuth credentials first
	var oauthCreds *OAuthCredentials
	if credStorage != nil {
		if creds, err := credStorage.GetOAuthCredentials("anthropic"); err == nil && creds != nil {
			// Check if token needs refresh
			if creds.IsTokenExpired() && creds.RefreshToken != "" {
				logging.Info("OAuth token expired, attempting refresh...")
				if refreshedCreds, err := RefreshAccessToken(creds); err == nil {
					// Store refreshed credentials
					credStorage.StoreOAuthCredentials(
						"anthropic",
						refreshedCreds.AccessToken,
						refreshedCreds.RefreshToken,
						refreshedCreds.ExpiresAt,
						refreshedCreds.ClientID,
					)
					oauthCreds = refreshedCreds
					logging.Info("OAuth token refreshed successfully")
				} else {
					logging.Warn("Failed to refresh OAuth token: %v", err)
				}
			} else if !creds.IsTokenExpired() {
				oauthCreds = creds
				logging.Info("Using valid OAuth credentials")
			}
		}
	}

	anthropicClientOptions := []option.RequestOption{}

	// Set up authentication
	if oauthCreds != nil {
		anthropicOpts.useOAuth = true
		anthropicOpts.oauthCreds = oauthCreds
		anthropicClientOptions = append(anthropicClientOptions, option.WithAuthToken(oauthCreds.AccessToken))
		logging.Info("Initialized Anthropic client with OAuth authentication via SDK")
	} else if opts.apiKey != "" {
		anthropicClientOptions = append(anthropicClientOptions, option.WithAPIKey(opts.apiKey))
		logging.Info("Initialized Anthropic client with API key authentication")
	} else {
		logging.Warn("No authentication method available - neither OAuth nor API key")
	}

	if anthropicOpts.useBedrock {
		anthropicClientOptions = append(anthropicClientOptions, bedrock.WithLoadDefaultConfig(context.Background()))
	}

	// Add request timeout to prevent indefinite hangs
	anthropicClientOptions = append(anthropicClientOptions, option.WithRequestTimeout(60*time.Second))

	anthropicClient := &anthropicClient{
		providerOptions:   opts,
		options:           anthropicOpts,
		credentialStorage: credStorage,
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
			if msg.Content().String() != "" {
				content := anthropic.NewTextBlock(msg.Content().String())
				if cache && !a.options.disableCache {
					content.OfText.CacheControl = anthropic.CacheControlEphemeralParam{
						Type: "ephemeral",
					}
				}
				blocks = append(blocks, content)
			}

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

func (a *anthropicClient) convertTools(tools []toolsPkg.BaseTool) []anthropic.ToolUnionParam {
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
	if isUser {
		for _, m := range lastMessage.Content {
			if m.OfText != nil && m.OfText.Text != "" {
				messageContent = m.OfText.Text
			}
		}
		if messageContent != "" && a.options.thinkingBudget != nil {
			if tokenBudget := a.options.thinkingBudget(messageContent); tokenBudget > 0 {
				thinkingParam = anthropic.ThinkingConfigParamOfEnabled(int64(tokenBudget))
				temperature = anthropic.Float(1)
			}
		}
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

func (a *anthropicClient) send(ctx context.Context, messages []message.Message, tools []toolsPkg.BaseTool) (resposne *ProviderResponse, err error) {
	// Handle proactive token refresh for OAuth
	if a.options.useOAuth && a.options.oauthCreds != nil {
		if a.options.oauthCreds.IsTokenExpired() && a.options.oauthCreds.RefreshToken != "" {
			if refreshedCreds, err := RefreshAccessToken(a.options.oauthCreds); err == nil {
				// Update stored credentials
				if a.credentialStorage != nil {
					a.credentialStorage.StoreOAuthCredentials(
						"anthropic",
						refreshedCreds.AccessToken,
						refreshedCreds.RefreshToken,
						refreshedCreds.ExpiresAt,
						refreshedCreds.ClientID,
					)
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
	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(preparedMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

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

			// Check for 401 and try OAuth token refresh
			if a.options.useOAuth && a.options.oauthCreds != nil && strings.Contains(err.Error(), "401") && a.options.oauthCreds.RefreshToken != "" {
				if refreshedCreds, refreshErr := RefreshAccessToken(a.options.oauthCreds); refreshErr == nil {
					// Update stored credentials
					if a.credentialStorage != nil {
						a.credentialStorage.StoreOAuthCredentials(
							"anthropic",
							refreshedCreds.AccessToken,
							refreshedCreds.RefreshToken,
							refreshedCreds.ExpiresAt,
							refreshedCreds.ClientID,
						)
					}
					a.options.oauthCreds = refreshedCreds

					// Update client with new token and retry
					a.recreateClient()
					logging.Info("Refreshed OAuth token and retrying request")
					continue
				}
			}

			retry, after, retryErr := a.shouldRetry(attempts, err)
			if retryErr != nil {
				return nil, retryErr
			}
			if retry {
				logging.Warn(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries))
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
		for _, block := range anthropicResponse.Content {
			if text, ok := block.AsAny().(anthropic.TextBlock); ok {
				content += text.Text
			}
		}

		return &ProviderResponse{
			Content:   content,
			ToolCalls: a.toolCalls(*anthropicResponse),
			Usage:     a.usage(*anthropicResponse),
		}, nil
	}
}

func (a *anthropicClient) stream(ctx context.Context, messages []message.Message, tools []toolsPkg.BaseTool) <-chan ProviderEvent {
	eventChan := make(chan ProviderEvent)

	// Handle proactive token refresh for OAuth
	if a.options.useOAuth && a.options.oauthCreds != nil {
		if a.options.oauthCreds.IsTokenExpired() && a.options.oauthCreds.RefreshToken != "" {
			if refreshedCreds, err := RefreshAccessToken(a.options.oauthCreds); err == nil {
				// Update stored credentials
				if a.credentialStorage != nil {
					a.credentialStorage.StoreOAuthCredentials(
						"anthropic",
						refreshedCreds.AccessToken,
						refreshedCreds.RefreshToken,
						refreshedCreds.ExpiresAt,
						refreshedCreds.ClientID,
					)
				}
				a.options.oauthCreds = refreshedCreds

				// Update client with new token
				a.recreateClient()
				logging.Info("Refreshed OAuth token proactively for streaming")
			}
		}
	}

	// Use SDK for both OAuth and API key authentication
	preparedMessages := a.preparedMessages(a.convertMessages(messages), a.convertTools(tools))
	cfg := config.Get()

	if cfg.Debug {
		jsonData, _ := json.Marshal(preparedMessages)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}
	attempts := 0
	go func() {
		for {
			attempts++
			anthropicStream := a.client.Messages.NewStreaming(
				ctx,
				preparedMessages,
			)
			accumulatedMessage := anthropic.Message{}

			// Check if context is already cancelled before starting the streaming loop
			select {
			case <-ctx.Done():
				eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
				close(eventChan)
				return
			default:
			}

			activeToolCalls := make(map[int]*message.ToolCall)
			for anthropicStream.Next() {
				event := anthropicStream.Current()
				err := accumulatedMessage.Accumulate(event)
				if err != nil {
					logging.Warn("Error accumulating message", "error", err)
					continue
				}

				switch event := event.AsAny().(type) {
				case anthropic.ContentBlockStartEvent:
					if event.ContentBlock.Type == "text" {
						eventChan <- ProviderEvent{Type: EventContentStart}
					} else if event.ContentBlock.Type == "tool_use" {
						toolCall := &message.ToolCall{
							ID:       event.ContentBlock.ID,
							Name:     event.ContentBlock.Name,
							Finished: false,
						}
						activeToolCalls[int(event.Index)] = toolCall
						eventChan <- ProviderEvent{
							Type:     EventToolUseStart,
							ToolCall: toolCall,
						}
					}

				case anthropic.ContentBlockDeltaEvent:
					if event.Delta.Type == "thinking_delta" && event.Delta.Thinking != "" {
						eventChan <- ProviderEvent{
							Type:     EventThinkingDelta,
							Thinking: event.Delta.Thinking,
						}
					} else if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
						eventChan <- ProviderEvent{
							Type:    EventContentDelta,
							Content: event.Delta.Text,
						}
					} else if event.Delta.Type == "input_json_delta" {
						if toolCall, exists := activeToolCalls[int(event.Index)]; exists {
							eventChan <- ProviderEvent{
								Type: EventToolUseDelta,
								ToolCall: &message.ToolCall{
									ID:       toolCall.ID,
									Finished: false,
									Input:    event.Delta.JSON.PartialJSON.Raw(),
								},
							}
						}
					}
				case anthropic.ContentBlockStopEvent:
					if toolCall, exists := activeToolCalls[int(event.Index)]; exists {
						eventChan <- ProviderEvent{
							Type: EventToolUseStop,
							ToolCall: &message.ToolCall{
								ID: toolCall.ID,
							},
						}
						delete(activeToolCalls, int(event.Index))
					} else {
						eventChan <- ProviderEvent{Type: EventContentStop}
					}

				case anthropic.MessageStopEvent:
					content := ""
					for _, block := range accumulatedMessage.Content {
						if text, ok := block.AsAny().(anthropic.TextBlock); ok {
							content += text.Text
						}
					}

					eventChan <- ProviderEvent{
						Type: EventComplete,
						Response: &ProviderResponse{
							Content:      content,
							ToolCalls:    a.toolCalls(accumulatedMessage),
							Usage:        a.usage(accumulatedMessage),
							FinishReason: a.finishReason(string(accumulatedMessage.StopReason)),
						},
					}
				}

				// Check if context is cancelled after processing each event
				select {
				case <-ctx.Done():
					eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
					close(eventChan)
					return
				default:
				}
			}

			err := anthropicStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				close(eventChan)
				return
			}

			// Check for 401 and try OAuth token refresh
			if a.options.useOAuth && a.options.oauthCreds != nil && strings.Contains(err.Error(), "401") && a.options.oauthCreds.RefreshToken != "" {
				if refreshedCreds, refreshErr := RefreshAccessToken(a.options.oauthCreds); refreshErr == nil {
					// Update stored credentials
					if a.credentialStorage != nil {
						a.credentialStorage.StoreOAuthCredentials(
							"anthropic",
							refreshedCreds.AccessToken,
							refreshedCreds.RefreshToken,
							refreshedCreds.ExpiresAt,
							refreshedCreds.ClientID,
						)
					}
					a.options.oauthCreds = refreshedCreds

					// Update client with new token and retry
					a.recreateClient()
					logging.Info("Refreshed OAuth token and retrying streaming request")
					continue
				}
			}

			// If there is an error we are going to see if we can retry the call
			retry, after, retryErr := a.shouldRetry(attempts, err)
			if retryErr != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
				close(eventChan)
				return
			}
			if retry {
				logging.Warn(fmt.Sprintf("Retrying due to rate limit... attempt %d of %d", attempts, maxRetries))
				select {
				case <-ctx.Done():
					// context cancelled
					if ctx.Err() != nil {
						eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
					}
					close(eventChan)
					return
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			if ctx.Err() != nil {
				eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
			}

			close(eventChan)
			return
		}
	}()
	return eventChan
}

func (a *anthropicClient) shouldRetry(attempts int, err error) (bool, int64, error) {
	var apierr *anthropic.Error
	if !errors.As(err, &apierr) {
		return false, 0, err
	}

	if apierr.StatusCode != 429 && apierr.StatusCode != 529 {
		return false, 0, err
	}

	if attempts > maxRetries {
		return false, 0, fmt.Errorf("maximum retry attempts reached for rate limit: %d retries", maxRetries)
	}

	retryMs := 0
	retryAfterValues := apierr.Response.Header.Values("Retry-After")

	backoffMs := 2000 * (1 << (attempts - 1))
	jitterMs := int(float64(backoffMs) * 0.2)
	retryMs = backoffMs + jitterMs
	if len(retryAfterValues) > 0 {
		if _, err := fmt.Sscanf(retryAfterValues[0], "%d", &retryMs); err == nil {
			retryMs = retryMs * 1000
		}
	}
	return true, int64(retryMs), nil
}

func (a *anthropicClient) toolCalls(msg anthropic.Message) []message.ToolCall {
	var toolCalls []message.ToolCall

	for _, block := range msg.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.ToolUseBlock:
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

func (a *anthropicClient) usage(msg anthropic.Message) TokenUsage {
	return TokenUsage{
		InputTokens:         msg.Usage.InputTokens,
		OutputTokens:        msg.Usage.OutputTokens,
		CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
		CacheReadTokens:     msg.Usage.CacheReadInputTokens,
	}
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

	clientOptions = append(clientOptions, option.WithRequestTimeout(60*time.Second))
	a.client = anthropic.NewClient(clientOptions...)
}
