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
	"mix/internal/llm/tools"
	"mix/internal/logging"
	"mix/internal/message"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

type openaiOptions struct {
	baseURL         string
	disableCache    bool
	reasoningEffort string
	extraHeaders    map[string]string
	useOAuth        bool
	oauthCreds      *OpenAICredentials
}

type OpenAIOption func(*openaiOptions)

type openaiClient struct {
	providerOptions   providerClientOptions
	options           openaiOptions
	client            openai.Client
	credentialStorage *CredentialStorage
}

type OpenAIClient ProviderClient

func newOpenAIClient(opts providerClientOptions) OpenAIClient {
	openaiOpts := openaiOptions{
		reasoningEffort: "medium",
	}
	for _, o := range opts.openaiOptions {
		o(&openaiOpts)
	}

	// Initialize credential storage
	credStorage, err := NewCredentialStorage()
	if err != nil {
		logging.Warn("Failed to initialize OAuth credential storage: %v", err)
	}

	// Check for OAuth credentials first
	var oauthCreds *OpenAICredentials
	if credStorage != nil {
		if creds, err := credStorage.GetOpenAICredentials("openai"); err == nil && creds != nil {
			// Check if token needs refresh
			if creds.IsTokenExpired() && creds.RefreshToken != "" {
				logging.Info("OpenAI OAuth token expired, attempting refresh...")
				if refreshedCreds, err := RefreshOpenAIAccessToken(creds); err == nil {
					// Store refreshed credentials
					credStorage.StoreOpenAICredentials("openai", refreshedCreds)
					oauthCreds = refreshedCreds
					logging.Info("OpenAI OAuth token refreshed successfully")
				} else {
					logging.Warn("Failed to refresh OpenAI OAuth token: %v", err)
				}
			} else if !creds.IsTokenExpired() {
				oauthCreds = creds
				logging.Info("Using valid OpenAI OAuth credentials")
			}
		}
	}

	openaiClientOptions := []option.RequestOption{}

	// Set up authentication - prioritize OAuth over API key
	if oauthCreds != nil && oauthCreds.APIKey != "" {
		openaiOpts.useOAuth = true
		openaiOpts.oauthCreds = oauthCreds
		openaiClientOptions = append(openaiClientOptions, option.WithAPIKey(oauthCreds.APIKey))
		logging.Info("Initialized OpenAI client with OAuth authentication")
	} else if opts.apiKey != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithAPIKey(opts.apiKey))
		logging.Info("Initialized OpenAI client with API key authentication")
	} else {
		logging.Warn("No authentication method available - neither OAuth nor API key")
	}

	if openaiOpts.baseURL != "" {
		openaiClientOptions = append(openaiClientOptions, option.WithBaseURL(openaiOpts.baseURL))
	}

	if openaiOpts.extraHeaders != nil {
		for key, value := range openaiOpts.extraHeaders {
			openaiClientOptions = append(openaiClientOptions, option.WithHeader(key, value))
		}
	}

	// Add request timeout to prevent indefinite hangs
	openaiClientOptions = append(openaiClientOptions, option.WithRequestTimeout(90*time.Second))

	client := openai.NewClient(openaiClientOptions...)
	return &openaiClient{
		providerOptions:   opts,
		options:           openaiOpts,
		client:            client,
		credentialStorage: credStorage,
	}
}

func (o *openaiClient) convertMessages(messages []message.Message) (openaiMessages []openai.ChatCompletionMessageParamUnion) {
	// Add system message first
	openaiMessages = append(openaiMessages, openai.SystemMessage(o.providerOptions.systemMessage))

	for _, msg := range messages {
		switch msg.Role {
		case message.User:
			var content []openai.ChatCompletionContentPartUnionParam
			textBlock := openai.ChatCompletionContentPartTextParam{Text: msg.Content().String()}
			content = append(content, openai.ChatCompletionContentPartUnionParam{OfText: &textBlock})
			for _, binaryContent := range msg.BinaryContent() {
				imageURL := openai.ChatCompletionContentPartImageImageURLParam{URL: binaryContent.String(models.ProviderOpenAI)}
				imageBlock := openai.ChatCompletionContentPartImageParam{ImageURL: imageURL}

				content = append(content, openai.ChatCompletionContentPartUnionParam{OfImageURL: &imageBlock})
			}

			openaiMessages = append(openaiMessages, openai.UserMessage(content))

		case message.Assistant:
			assistantMsg := openai.ChatCompletionAssistantMessageParam{
				Role: "assistant",
			}

			if msg.Content().String() != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(msg.Content().String()),
				}
			}

			if len(msg.ToolCalls()) > 0 {
				assistantMsg.ToolCalls = make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls()))
				for i, call := range msg.ToolCalls() {
					assistantMsg.ToolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID:   call.ID,
						Type: "function",
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      call.Name,
							Arguments: call.Input,
						},
					}
				}
			}

			openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &assistantMsg,
			})

		case message.Tool:
			for _, result := range msg.ToolResults() {
				openaiMessages = append(openaiMessages,
					openai.ToolMessage(result.Content, result.ToolCallID),
				)
			}
		}
	}

	return
}

func (o *openaiClient) convertTools(tools []tools.BaseTool) []openai.ChatCompletionToolParam {
	openaiTools := make([]openai.ChatCompletionToolParam, len(tools))

	for i, tool := range tools {
		info := tool.Info()
		openaiTools[i] = openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        info.Name,
				Description: openai.String(info.Description),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": info.Parameters,
					"required":   info.Required,
				},
			},
		}
	}

	return openaiTools
}

func (o *openaiClient) finishReason(reason string) message.FinishReason {
	switch reason {
	case "stop":
		return message.FinishReasonEndTurn
	case "length":
		return message.FinishReasonMaxTokens
	case "tool_calls":
		return message.FinishReasonToolUse
	default:
		return message.FinishReasonUnknown
	}
}

func (o *openaiClient) preparedParams(messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(o.providerOptions.model.APIModel),
		Messages: messages,
		Tools:    tools,
	}

	if o.providerOptions.model.CanReason == true {
		params.MaxCompletionTokens = openai.Int(o.providerOptions.maxTokens)
		switch o.options.reasoningEffort {
		case "low":
			params.ReasoningEffort = shared.ReasoningEffortLow
		case "medium":
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case "high":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		}
	} else {
		params.MaxTokens = openai.Int(o.providerOptions.maxTokens)
	}

	return params
}

func (o *openaiClient) send(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (response *ProviderResponse, err error) {
	// Handle proactive token refresh for OAuth
	if o.options.useOAuth && o.options.oauthCreds != nil {
		if o.options.oauthCreds.IsTokenExpired() && o.options.oauthCreds.RefreshToken != "" {
			if refreshedCreds, err := RefreshOpenAIAccessToken(o.options.oauthCreds); err == nil {
				// Update stored credentials
				if o.credentialStorage != nil {
					o.credentialStorage.StoreOpenAICredentials("openai", refreshedCreds)
				}
				o.options.oauthCreds = refreshedCreds

				// Update client with new token
				o.recreateClient()
				logging.Info("Refreshed OpenAI OAuth token proactively")
			}
		}
	}

	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))
	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}
	attempts := 0
	for {
		attempts++
		openaiResponse, err := o.client.Chat.Completions.New(
			ctx,
			params,
		)
		// If there is an error we are going to see if we can retry the call
		if err != nil {
			// Check for 401 and try OAuth token refresh
			if o.options.useOAuth && o.options.oauthCreds != nil && strings.Contains(err.Error(), "401") && o.options.oauthCreds.RefreshToken != "" {
				if refreshedCreds, refreshErr := RefreshOpenAIAccessToken(o.options.oauthCreds); refreshErr == nil {
					// Update stored credentials
					if o.credentialStorage != nil {
						o.credentialStorage.StoreOpenAICredentials("openai", refreshedCreds)
					}
					o.options.oauthCreds = refreshedCreds

					// Update client with new token and retry
					o.recreateClient()
					logging.Info("Refreshed OpenAI OAuth token and retrying request")
					continue
				}
			}

			retry, after, retryErr := o.shouldRetry(attempts, err)
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
		if openaiResponse.Choices[0].Message.Content != "" {
			content = openaiResponse.Choices[0].Message.Content
		}

		toolCalls := o.toolCalls(*openaiResponse)
		finishReason := o.finishReason(string(openaiResponse.Choices[0].FinishReason))

		if len(toolCalls) > 0 {
			finishReason = message.FinishReasonToolUse
		}

		return &ProviderResponse{
			Content:      content,
			ToolCalls:    toolCalls,
			Usage:        o.usage(*openaiResponse),
			FinishReason: finishReason,
		}, nil
	}
}

func (o *openaiClient) stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan ProviderEvent {
	eventChan := make(chan ProviderEvent)

	// Handle proactive token refresh for OAuth
	if o.options.useOAuth && o.options.oauthCreds != nil {
		if o.options.oauthCreds.IsTokenExpired() && o.options.oauthCreds.RefreshToken != "" {
			if refreshedCreds, err := RefreshOpenAIAccessToken(o.options.oauthCreds); err == nil {
				// Update stored credentials
				if o.credentialStorage != nil {
					o.credentialStorage.StoreOpenAICredentials("openai", refreshedCreds)
				}
				o.options.oauthCreds = refreshedCreds

				// Update client with new token
				o.recreateClient()
				logging.Info("Refreshed OpenAI OAuth token proactively for streaming")
			}
		}
	}

	params := o.preparedParams(o.convertMessages(messages), o.convertTools(tools))
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	cfg := config.Get()
	if cfg.Debug {
		jsonData, _ := json.Marshal(params)
		logging.Debug("Prepared messages", "messages", string(jsonData))
	}

	attempts := 0

	go func() {
		for {
			attempts++
			openaiStream := o.client.Chat.Completions.NewStreaming(
				ctx,
				params,
			)

			acc := openai.ChatCompletionAccumulator{}
			currentContent := ""
			toolCalls := make([]message.ToolCall, 0)

			for openaiStream.Next() {
				chunk := openaiStream.Current()
				acc.AddChunk(chunk)

				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						eventChan <- ProviderEvent{
							Type:    EventContentDelta,
							Content: choice.Delta.Content,
						}
						currentContent += choice.Delta.Content
					}
				}
			}

			err := openaiStream.Err()
			if err == nil || errors.Is(err, io.EOF) {
				// Stream completed successfully
				finishReason := o.finishReason(string(acc.ChatCompletion.Choices[0].FinishReason))
				if len(acc.ChatCompletion.Choices[0].Message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, o.toolCalls(acc.ChatCompletion)...)
				}
				if len(toolCalls) > 0 {
					finishReason = message.FinishReasonToolUse
				}

				eventChan <- ProviderEvent{
					Type: EventComplete,
					Response: &ProviderResponse{
						Content:      currentContent,
						ToolCalls:    toolCalls,
						Usage:        o.usage(acc.ChatCompletion),
						FinishReason: finishReason,
					},
				}
				close(eventChan)
				return
			}

			// Check for 401 and try OAuth token refresh
			if o.options.useOAuth && o.options.oauthCreds != nil && strings.Contains(err.Error(), "401") && o.options.oauthCreds.RefreshToken != "" {
				if refreshedCreds, refreshErr := RefreshOpenAIAccessToken(o.options.oauthCreds); refreshErr == nil {
					// Update stored credentials
					if o.credentialStorage != nil {
						o.credentialStorage.StoreOpenAICredentials("openai", refreshedCreds)
					}
					o.options.oauthCreds = refreshedCreds

					// Update client with new token and retry
					o.recreateClient()
					logging.Info("Refreshed OpenAI OAuth token and retrying streaming request")
					continue
				}
			}

			// If there is an error we are going to see if we can retry the call
			retry, after, retryErr := o.shouldRetry(attempts, err)
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
					if ctx.Err() == nil {
						eventChan <- ProviderEvent{Type: EventError, Error: ctx.Err()}
					}
					close(eventChan)
					return
				case <-time.After(time.Duration(after) * time.Millisecond):
					continue
				}
			}
			eventChan <- ProviderEvent{Type: EventError, Error: retryErr}
			close(eventChan)
			return
		}
	}()

	return eventChan
}

func (o *openaiClient) shouldRetry(attempts int, err error) (bool, int64, error) {
	var apierr *openai.Error
	if !errors.As(err, &apierr) {
		return false, 0, err
	}

	if apierr.StatusCode != 429 && apierr.StatusCode != 500 {
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

func (o *openaiClient) toolCalls(completion openai.ChatCompletion) []message.ToolCall {
	var toolCalls []message.ToolCall

	if len(completion.Choices) > 0 && len(completion.Choices[0].Message.ToolCalls) > 0 {
		for _, call := range completion.Choices[0].Message.ToolCalls {
			toolCall := message.ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Input:    call.Function.Arguments,
				Type:     "function",
				Finished: true,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return toolCalls
}

func (o *openaiClient) usage(completion openai.ChatCompletion) TokenUsage {
	cachedTokens := completion.Usage.PromptTokensDetails.CachedTokens
	inputTokens := completion.Usage.PromptTokens - cachedTokens

	return TokenUsage{
		InputTokens:         inputTokens,
		OutputTokens:        completion.Usage.CompletionTokens,
		CacheCreationTokens: 0, // OpenAI doesn't provide this directly
		CacheReadTokens:     cachedTokens,
	}
}

func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(options *openaiOptions) {
		options.baseURL = baseURL
	}
}

func WithOpenAIExtraHeaders(headers map[string]string) OpenAIOption {
	return func(options *openaiOptions) {
		options.extraHeaders = headers
	}
}

func WithOpenAIDisableCache() OpenAIOption {
	return func(options *openaiOptions) {
		options.disableCache = true
	}
}

func WithReasoningEffort(effort string) OpenAIOption {
	return func(options *openaiOptions) {
		defaultReasoningEffort := "medium"
		switch effort {
		case "low", "medium", "high":
			defaultReasoningEffort = effort
		default:
			logging.Warn("Invalid reasoning effort, using default: medium")
		}
		options.reasoningEffort = defaultReasoningEffort
	}
}

func (o *openaiClient) recreateClient() {
	var clientOptions []option.RequestOption

	if o.options.useOAuth && o.options.oauthCreds != nil && o.options.oauthCreds.APIKey != "" {
		clientOptions = append(clientOptions, option.WithAPIKey(o.options.oauthCreds.APIKey))
	} else if o.providerOptions.apiKey != "" {
		clientOptions = append(clientOptions, option.WithAPIKey(o.providerOptions.apiKey))
	}

	if o.options.baseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(o.options.baseURL))
	}

	if o.options.extraHeaders != nil {
		for key, value := range o.options.extraHeaders {
			clientOptions = append(clientOptions, option.WithHeader(key, value))
		}
	}

	clientOptions = append(clientOptions, option.WithRequestTimeout(90*time.Second))
	o.client = openai.NewClient(clientOptions...)
}

// WithOpenAIOAuth configures the OpenAI client to use OAuth authentication
func WithOpenAIOAuth(credentials *OpenAICredentials) OpenAIOption {
	return func(options *openaiOptions) {
		options.useOAuth = true
		options.oauthCreds = credentials
	}
}
