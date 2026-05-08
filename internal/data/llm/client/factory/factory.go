package factory

import (
	"fmt"
	"os"

	llmclient "ferryman-agent/internal/data/llm/client"
	anthropicclient "ferryman-agent/internal/data/llm/client/anthropic"
	azureclient "ferryman-agent/internal/data/llm/client/azure"
	bedrockclient "ferryman-agent/internal/data/llm/client/bedrock"
	copilotclient "ferryman-agent/internal/data/llm/client/copilot"
	geminiclient "ferryman-agent/internal/data/llm/client/gemini"
	mockclient "ferryman-agent/internal/data/llm/client/mock"
	openaiclient "ferryman-agent/internal/data/llm/client/openai"
	vertexaiclient "ferryman-agent/internal/data/llm/client/vertexai"
	"ferryman-agent/internal/data/llm/models"
)

type Option func(*options)

type options struct {
	APIKey  string
	BaseURL string

	OpenAIOptions    []openaiclient.Option
	AnthropicOptions []anthropicclient.Option
	GeminiOptions    []geminiclient.Option
	BedrockOptions   []bedrockclient.Option
	CopilotOptions   []copilotclient.Option
}

func WithAPIKey(apiKey string) Option {
	return func(options *options) {
		options.APIKey = apiKey
	}
}

func WithBaseURL(baseURL string) Option {
	return func(options *options) {
		options.BaseURL = baseURL
	}
}

func WithOpenAIOptions(openaiOptions ...openaiclient.Option) Option {
	return func(options *options) {
		options.OpenAIOptions = append(options.OpenAIOptions, openaiOptions...)
	}
}

func WithAnthropicOptions(anthropicOptions ...anthropicclient.Option) Option {
	return func(options *options) {
		options.AnthropicOptions = append(options.AnthropicOptions, anthropicOptions...)
	}
}

func WithGeminiOptions(geminiOptions ...geminiclient.Option) Option {
	return func(options *options) {
		options.GeminiOptions = append(options.GeminiOptions, geminiOptions...)
	}
}

func WithBedrockOptions(bedrockOptions ...bedrockclient.Option) Option {
	return func(options *options) {
		options.BedrockOptions = append(options.BedrockOptions, bedrockOptions...)
	}
}

func WithCopilotOptions(copilotOptions ...copilotclient.Option) Option {
	return func(options *options) {
		options.CopilotOptions = append(options.CopilotOptions, copilotOptions...)
	}
}

func NewClient(provider models.ModelProvider, opts ...Option) (llmclient.Client, error) {
	clientOptions := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&clientOptions)
		}
	}

	switch provider {
	case models.ProviderCopilot:
		return copilotclient.NewClient(clientOptions.APIKey, clientOptions.CopilotOptions...), nil
	case models.ProviderAnthropic:
		clientOptions.AnthropicOptions = append(clientOptions.AnthropicOptions, anthropicclient.WithShouldThinkFn(anthropicclient.DefaultShouldThinkFn))
		return anthropicclient.NewClient(clientOptions.APIKey, clientOptions.AnthropicOptions...), nil
	case models.ProviderOpenAI:
		if clientOptions.BaseURL != "" {
			clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions, openaiclient.WithBaseURL(clientOptions.BaseURL))
		}
		return openaiclient.NewClient(clientOptions.APIKey, clientOptions.OpenAIOptions...), nil
	case models.ProviderGemini:
		return geminiclient.NewClient(clientOptions.APIKey, clientOptions.GeminiOptions...), nil
	case models.ProviderBedrock:
		return bedrockclient.NewClient(clientOptions.APIKey, clientOptions.BedrockOptions...), nil
	case models.ProviderGROQ:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions, openaiclient.WithDefaultBaseURL("https://api.groq.com/openai/v1"))
		return openaiclient.NewClient(clientOptions.APIKey, clientOptions.OpenAIOptions...), nil
	case models.ProviderAzure:
		return azureclient.NewClient(clientOptions.APIKey, clientOptions.OpenAIOptions...), nil
	case models.ProviderVertexAI:
		return vertexaiclient.NewClient(clientOptions.APIKey, clientOptions.GeminiOptions...), nil
	case models.ProviderOpenRouter:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			openaiclient.WithDefaultBaseURL("https://openrouter.ai/api/v1"),
			openaiclient.WithExtraHeaders(map[string]string{
				"HTTP-Referer": "ferryer.ai",
				"X-Title":      "Ferryer",
			}),
		)
		return openaiclient.NewClient(clientOptions.APIKey, clientOptions.OpenAIOptions...), nil
	case models.ProviderXAI:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions, openaiclient.WithDefaultBaseURL("https://api.x.ai/v1"))
		return openaiclient.NewClient(clientOptions.APIKey, clientOptions.OpenAIOptions...), nil
	case models.ProviderLocal:
		baseURL := clientOptions.BaseURL
		if baseURL == "" {
			baseURL = os.Getenv("LOCAL_ENDPOINT")
		}
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions, openaiclient.WithDefaultBaseURL(baseURL))
		return openaiclient.NewClient(clientOptions.APIKey, clientOptions.OpenAIOptions...), nil
	case models.ProviderMock:
		return mockclient.NewClient(), nil
	default:
		return nil, fmt.Errorf("provider not supported: %s", provider)
	}
}
