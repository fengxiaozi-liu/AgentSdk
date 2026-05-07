package provider

import (
	"context"
	"fmt"
	"os"

	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/data/llm/provider/client"
	"ferryman-agent/internal/memory/message"
	toolcore "ferryman-agent/internal/tools"
)

type Provider interface {
	SendMessages(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) (*client.Response, error)
	StreamResponse(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) <-chan client.Event
	Model() models.Model
}

type ProviderConfig struct {
	Provider    models.ModelProvider `json:"provider"`
	APIKey      string               `json:"apiKey"`
	BaseURL     string               `json:"baseURL"`
	ModelConfig ModelConfig          `json:"modelConfig"`
	Disabled    bool                 `json:"disabled"`
}

type ModelConfig struct {
	Model           models.ModelID `json:"model"`
	MaxTokens       int64          `json:"maxTokens,omitempty"`
	ReasoningEffort string         `json:"reasoningEffort,omitempty"`
}

type baseProvider struct {
	options client.Options
	client  client.Client
}

func CreateProvider(providerCfg ProviderConfig, systemPrompt string, extraOpts ...ProviderClientOption) (Provider, error) {
	providerName := providerCfg.Provider
	if providerName == "" {
		return nil, fmt.Errorf("provider is required for model %s", providerCfg.ModelConfig.Model)
	}
	if providerCfg.Disabled {
		return nil, fmt.Errorf("provider %s is not enabled", providerName)
	}
	model := models.ResolveModel(providerName, providerCfg.ModelConfig.Model)
	maxTokens := model.DefaultMaxTokens
	if providerCfg.ModelConfig.MaxTokens > 0 {
		maxTokens = providerCfg.ModelConfig.MaxTokens
	}
	opts := []ProviderClientOption{
		WithAPIKey(providerCfg.APIKey),
		WithModel(model),
		WithSystemMessage(systemPrompt),
		WithMaxTokens(maxTokens),
	}
	opts = append(opts, extraOpts...)
	if (providerName == models.ProviderOpenAI || providerName == models.ProviderLocal) && model.CanReason {
		opts = append(
			opts,
			WithOpenAIOptions(
				client.WithReasoningEffort(providerCfg.ModelConfig.ReasoningEffort),
			),
		)
	} else if providerName == models.ProviderAnthropic && model.CanReason {
		opts = append(
			opts,
			WithAnthropicOptions(
				client.WithAnthropicShouldThinkFn(client.DefaultShouldThinkFn),
			),
		)
	}
	if providerCfg.BaseURL != "" {
		switch providerName {
		case models.ProviderOpenAI, models.ProviderGROQ, models.ProviderOpenRouter, models.ProviderXAI, models.ProviderLocal:
			opts = append(opts, WithOpenAIOptions(client.WithOpenAIBaseURL(providerCfg.BaseURL)))
		}
	}
	createdProvider, err := NewProvider(
		providerName,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create provider: %v", err)
	}

	return createdProvider, nil
}

func Configured(providerCfg ProviderConfig) bool {
	return providerCfg.Provider != "" || providerCfg.ModelConfig.Model != ""
}

func NewProvider(providerName models.ModelProvider, opts ...ProviderClientOption) (Provider, error) {
	clientOptions := client.Options{}
	for _, o := range opts {
		o(&clientOptions)
	}
	switch providerName {
	case models.ProviderCopilot:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewCopilotClient(clientOptions),
		}, nil
	case models.ProviderAnthropic:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewAnthropicClient(clientOptions),
		}, nil
	case models.ProviderOpenAI:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewOpenAIClient(clientOptions),
		}, nil
	case models.ProviderGemini:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewGeminiClient(clientOptions),
		}, nil
	case models.ProviderBedrock:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewBedrockClient(clientOptions),
		}, nil
	case models.ProviderGROQ:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			client.WithOpenAIDefaultBaseURL("https://api.groq.com/openai/v1"),
		)
		return &baseProvider{
			options: clientOptions,
			client:  client.NewOpenAIClient(clientOptions),
		}, nil
	case models.ProviderAzure:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewAzureClient(clientOptions),
		}, nil
	case models.ProviderVertexAI:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewVertexAIClient(clientOptions),
		}, nil
	case models.ProviderOpenRouter:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			client.WithOpenAIDefaultBaseURL("https://openrouter.ai/api/v1"),
			client.WithOpenAIExtraHeaders(map[string]string{
				"HTTP-Referer": "ferryer.ai",
				"X-Title":      "Ferryer",
			}),
		)
		return &baseProvider{
			options: clientOptions,
			client:  client.NewOpenAIClient(clientOptions),
		}, nil
	case models.ProviderXAI:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			client.WithOpenAIDefaultBaseURL("https://api.x.ai/v1"),
		)
		return &baseProvider{
			options: clientOptions,
			client:  client.NewOpenAIClient(clientOptions),
		}, nil
	case models.ProviderLocal:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			client.WithOpenAIDefaultBaseURL(os.Getenv("LOCAL_ENDPOINT")),
		)
		return &baseProvider{
			options: clientOptions,
			client:  client.NewOpenAIClient(clientOptions),
		}, nil
	case models.ProviderMock:
		return &baseProvider{
			options: clientOptions,
			client:  client.NewMockClient(clientOptions),
		}, nil
	}
	return nil, fmt.Errorf("provider not supported: %s", providerName)
}

func (p *baseProvider) cleanMessages(messages []message.Message) (cleaned []message.Message) {
	for _, msg := range messages {
		// The message has no content
		if len(msg.Parts) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return
}

func (p *baseProvider) SendMessages(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) (*client.Response, error) {
	messages = p.cleanMessages(messages)
	return p.client.Send(ctx, messages, tools)
}

func (p *baseProvider) Model() models.Model {
	return p.options.Model
}

func (p *baseProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) <-chan client.Event {
	messages = p.cleanMessages(messages)
	return p.client.Stream(ctx, messages, tools)
}
