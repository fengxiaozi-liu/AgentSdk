package provider

import (
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/data/llm/provider/client"
	anthropicclient "ferryman-agent/internal/data/llm/provider/client/anthropic"
	bedrockclient "ferryman-agent/internal/data/llm/provider/client/bedrock"
	copilotclient "ferryman-agent/internal/data/llm/provider/client/copilot"
	geminiclient "ferryman-agent/internal/data/llm/provider/client/gemini"
	openaiclient "ferryman-agent/internal/data/llm/provider/client/openai"
)

type providerClientOptions struct {
	client.Options

	AnthropicOptions []anthropicclient.Option
	OpenAIOptions    []openaiclient.Option
	GeminiOptions    []geminiclient.Option
	BedrockOptions   []bedrockclient.Option
	CopilotOptions   []copilotclient.Option
}

type ProviderClientOption func(*providerClientOptions)

func WithAPIKey(apiKey string) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.APIKey = apiKey
	}
}

func WithModel(model models.Model) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.Model = model
	}
}

func WithMaxTokens(maxTokens int64) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.MaxTokens = maxTokens
	}
}

func WithSystemMessage(systemMessage string) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.SystemMessage = systemMessage
	}
}

func WithDebug(debug bool) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.Debug = debug
	}
}

func WithAnthropicOptions(anthropicOptions ...anthropicclient.Option) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.AnthropicOptions = append(options.AnthropicOptions, anthropicOptions...)
	}
}

func WithOpenAIOptions(openaiOptions ...openaiclient.Option) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.OpenAIOptions = append(options.OpenAIOptions, openaiOptions...)
	}
}

func WithGeminiOptions(geminiOptions ...geminiclient.Option) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.GeminiOptions = append(options.GeminiOptions, geminiOptions...)
	}
}

func WithBedrockOptions(bedrockOptions ...bedrockclient.Option) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.BedrockOptions = append(options.BedrockOptions, bedrockOptions...)
	}
}

func WithCopilotOptions(copilotOptions ...copilotclient.Option) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.CopilotOptions = append(options.CopilotOptions, copilotOptions...)
	}
}
