package provider

import (
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/data/llm/provider/client"
)

type ProviderClientOption func(*client.Options)

func WithAPIKey(apiKey string) ProviderClientOption {
	return func(options *client.Options) {
		options.APIKey = apiKey
	}
}

func WithModel(model models.Model) ProviderClientOption {
	return func(options *client.Options) {
		options.Model = model
	}
}

func WithMaxTokens(maxTokens int64) ProviderClientOption {
	return func(options *client.Options) {
		options.MaxTokens = maxTokens
	}
}

func WithSystemMessage(systemMessage string) ProviderClientOption {
	return func(options *client.Options) {
		options.SystemMessage = systemMessage
	}
}

func WithDebug(debug bool) ProviderClientOption {
	return func(options *client.Options) {
		options.Debug = debug
	}
}

func WithAnthropicOptions(anthropicOptions ...client.AnthropicOption) ProviderClientOption {
	return func(options *client.Options) {
		options.AnthropicOptions = append(options.AnthropicOptions, anthropicOptions...)
	}
}

func WithOpenAIOptions(openaiOptions ...client.OpenAIOption) ProviderClientOption {
	return func(options *client.Options) {
		options.OpenAIOptions = append(options.OpenAIOptions, openaiOptions...)
	}
}

func WithGeminiOptions(geminiOptions ...client.GeminiOption) ProviderClientOption {
	return func(options *client.Options) {
		options.GeminiOptions = append(options.GeminiOptions, geminiOptions...)
	}
}

func WithBedrockOptions(bedrockOptions ...client.BedrockOption) ProviderClientOption {
	return func(options *client.Options) {
		options.BedrockOptions = append(options.BedrockOptions, bedrockOptions...)
	}
}

func WithCopilotOptions(copilotOptions ...client.CopilotOption) ProviderClientOption {
	return func(options *client.Options) {
		options.CopilotOptions = append(options.CopilotOptions, copilotOptions...)
	}
}
