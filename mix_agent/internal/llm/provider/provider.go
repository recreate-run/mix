package provider

import (
	"context"
	"fmt"
	"os"

	"mix/internal/config"
	"mix/internal/llm/interfaces"
	"mix/internal/llm/models"
	"mix/internal/logging"
	"mix/internal/message"
)

const maxRetries = 8

type providerClientOptions struct {
	apiKey        string
	model         models.Model
	maxTokens     int64
	systemMessage string

	anthropicOptions []AnthropicOption
	openaiOptions    []OpenAIOption
	geminiOptions    []GeminiOption
	bedrockOptions   []BedrockOption
}

type ProviderClientOption func(*providerClientOptions)

type baseProvider[C interfaces.ProviderClient] struct {
	options providerClientOptions
	client  C
}

// retrieveAPIKey attempts to get the API key from database if not already set
func retrieveAPIKey(options *providerClientOptions, providerName models.ModelProvider) {
	if options.apiKey != "" {
		return
	}

	ctx := context.Background()
	credentialsService := config.GetAPICredentials()
	if credentialsService == nil {
		return
	}

	apiKey, err := credentialsService.GetAPIKey(ctx, providerName)
	if err == nil && apiKey != "" {
		options.apiKey = apiKey
		return
	}

	// Try environment variable fallback for providers that support it
	var envVar string
	switch providerName {
	case models.ProviderGemini:
		envVar = "GEMINI_API_KEY"
	case models.ProviderOpenRouter:
		envVar = "OPENROUTER_API_KEY"
	case models.ProviderGROQ:
		envVar = "GROQ_API_KEY"
	case models.ProviderXAI:
		envVar = "XAI_API_KEY"
	}
	
	if envVar != "" {
		if envAPIKey := os.Getenv(envVar); envAPIKey != "" {
			options.apiKey = envAPIKey
			return
		}
	}

	// Warn for non-OAuth providers that need API keys
	if providerName != models.ProviderAnthropic && providerName != models.ProviderOpenAI {
		logging.Warn("No API key found in database for provider", "provider", providerName)
	}
}

// createProviderClient creates the appropriate provider client based on provider name
func createProviderClient(providerName models.ModelProvider, clientOptions providerClientOptions) (interfaces.Provider, error) {
	switch providerName {
	case models.ProviderAnthropic:
		return &baseProvider[AnthropicClient]{
			options: clientOptions,
			client:  newAnthropicClient(clientOptions),
		}, nil

	case models.ProviderOpenAI:
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil

	case models.ProviderGemini:
		return &baseProvider[GeminiClient]{
			options: clientOptions,
			client:  newGeminiClient(clientOptions),
		}, nil

	case models.ProviderBedrock:
		return &baseProvider[BedrockClient]{
			options: clientOptions,
			client:  newBedrockClient(clientOptions),
		}, nil

	case models.ProviderVertexAI:
		return &baseProvider[VertexAIClient]{
			options: clientOptions,
			client:  newVertexAIClient(clientOptions),
		}, nil

	case models.ProviderAzure:
		client, err := newAzureClient(clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure client: %w", err)
		}
		return &baseProvider[AzureClient]{
			options: clientOptions,
			client:  client,
		}, nil

	case models.ProviderGROQ:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://api.groq.com/openai/v1"),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil

	case models.ProviderOpenRouter:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://openrouter.ai/api/v1"),
			WithOpenAIExtraHeaders(map[string]string{
				"HTTP-Referer": "mix.ai",
				"X-Title":      "Mix",
			}),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil

	case models.ProviderXAI:
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL("https://api.x.ai/v1"),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil

	case models.ProviderLocal:
		localEndpoint := os.Getenv("LOCAL_ENDPOINT")
		if localEndpoint == "" {
			localEndpoint = "http://localhost:8000/v1"
		}
		clientOptions.openaiOptions = append(clientOptions.openaiOptions,
			WithOpenAIBaseURL(localEndpoint),
		)
		return &baseProvider[OpenAIClient]{
			options: clientOptions,
			client:  newOpenAIClient(clientOptions),
		}, nil

	case models.ProviderMock:
		return nil, fmt.Errorf("mock provider not implemented")

	default:
		return nil, fmt.Errorf("provider not supported: %s", providerName)
	}
}

func NewProvider(providerName models.ModelProvider, opts ...ProviderClientOption) (interfaces.Provider, error) {
	clientOptions := providerClientOptions{}

	for _, o := range opts {
		o(&clientOptions)
	}

	retrieveAPIKey(&clientOptions, providerName)

	return createProviderClient(providerName, clientOptions)
}

func (p *baseProvider[C]) cleanMessages(messages []message.Message) (cleaned []message.Message) {
	for _, msg := range messages {
		// The message has no content
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return
}

func (p *baseProvider[C]) SendMessages(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) (*interfaces.ProviderResponse, error) {
	messages = p.cleanMessages(messages)
	return p.client.Send(ctx, messages, tools)
}

func (p *baseProvider[C]) Model() models.Model {
	return p.options.model
}

func (p *baseProvider[C]) StreamResponse(ctx context.Context, messages []message.Message, tools []interfaces.BaseTool) <-chan interfaces.ProviderEvent {
	messages = p.cleanMessages(messages)
	return p.client.Stream(ctx, messages, tools)
}

func WithAPIKey(apiKey string) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.apiKey = apiKey
	}
}

func WithModel(model models.Model) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.model = model
	}
}

func WithMaxTokens(maxTokens int64) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.maxTokens = maxTokens
	}
}

func WithSystemMessage(systemMessage string) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.systemMessage = systemMessage
	}
}

func WithAnthropicOptions(anthropicOptions ...AnthropicOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.anthropicOptions = anthropicOptions
	}
}

func WithOpenAIOptions(openaiOptions ...OpenAIOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.openaiOptions = openaiOptions
	}
}

func WithGeminiOptions(geminiOptions ...GeminiOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.geminiOptions = geminiOptions
	}
}

func WithBedrockOptions(bedrockOptions ...BedrockOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.bedrockOptions = bedrockOptions
	}
}
