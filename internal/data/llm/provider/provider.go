package provider

import (
	"context"
	client "ferryman-agent/internal/data/llm/client"
	anthropicclient "ferryman-agent/internal/data/llm/client/anthropic"
	azureclient "ferryman-agent/internal/data/llm/client/azure"
	bedrockclient "ferryman-agent/internal/data/llm/client/bedrock"
	copilotclient "ferryman-agent/internal/data/llm/client/copilot"
	geminiclient "ferryman-agent/internal/data/llm/client/gemini"
	mockclient "ferryman-agent/internal/data/llm/client/mock"
	openaiclient "ferryman-agent/internal/data/llm/client/openai"
	vertexaiclient "ferryman-agent/internal/data/llm/client/vertexai"
	"ferryman-agent/internal/data/llm/models"
	"fmt"
	"os"

	"ferryman-agent/internal/memory/message"
	toolcore "ferryman-agent/internal/tools"
)

type ProviderConfig struct {
	Provider    models.ModelProvider `json:"provider"`
	APIKey      string               `json:"apiKey"`
	BaseURL     string               `json:"baseURL"`
	ModelConfig models.ModelConfig   `json:"modelConfig"`
	Disabled    bool                 `json:"disabled"`
}

type Provider interface {
	SendMessages(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) (*client.Response, error)
	StreamResponse(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) <-chan client.Event
	Model() models.Model
}

type baseProvider struct {
	options client.Options
	client  client.Client
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

func CreateProvider(providerCfg ProviderConfig, systemPrompt string, extraOpts ...ProviderClientOption) (Provider, error) {
	providerName := providerCfg.Provider
	if providerName == "" {
		return nil, fmt.Errorf("provider is required for model %s", providerCfg.ModelConfig.ModelId)
	}
	if providerCfg.Disabled {
		return nil, fmt.Errorf("provider %s is not enabled", providerName)
	}
	model := models.ResolveModel(providerName, providerCfg.ModelConfig.ModelId)
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
				openaiclient.WithReasoningEffort(providerCfg.ModelConfig.ReasoningEffort),
			),
		)
	} else if providerName == models.ProviderAnthropic && model.CanReason {
		opts = append(
			opts,
			WithAnthropicOptions(
				anthropicclient.WithShouldThinkFn(anthropicclient.DefaultShouldThinkFn),
			),
		)
	}
	if providerCfg.BaseURL != "" {
		switch providerName {
		case models.ProviderOpenAI, models.ProviderGROQ, models.ProviderOpenRouter, models.ProviderXAI, models.ProviderLocal:
			opts = append(opts, WithOpenAIOptions(openaiclient.WithBaseURL(providerCfg.BaseURL)))
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
	return providerCfg.Provider != "" || providerCfg.ModelConfig.ModelId != ""
}

func NewProvider(providerName models.ModelProvider, opts ...ProviderClientOption) (Provider, error) {
	clientOptions := providerClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}
	switch providerName {
	case models.ProviderCopilot:
		return &baseProvider{
			options: clientOptions.Options,
			client:  copilotclient.NewClient(clientOptions.Options, clientOptions.CopilotOptions...),
		}, nil
	case models.ProviderAnthropic:
		return &baseProvider{
			options: clientOptions.Options,
			client:  anthropicclient.NewClient(clientOptions.Options, clientOptions.AnthropicOptions...),
		}, nil
	case models.ProviderOpenAI:
		return &baseProvider{
			options: clientOptions.Options,
			client:  openaiclient.NewClient(clientOptions.Options, clientOptions.OpenAIOptions...),
		}, nil
	case models.ProviderGemini:
		return &baseProvider{
			options: clientOptions.Options,
			client:  geminiclient.NewClient(clientOptions.Options, clientOptions.GeminiOptions...),
		}, nil
	case models.ProviderBedrock:
		return &baseProvider{
			options: clientOptions.Options,
			client:  bedrockclient.NewClient(clientOptions.Options, clientOptions.BedrockOptions...),
		}, nil
	case models.ProviderGROQ:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			openaiclient.WithDefaultBaseURL("https://api.groq.com/openai/v1"),
		)
		return &baseProvider{
			options: clientOptions.Options,
			client:  openaiclient.NewClient(clientOptions.Options, clientOptions.OpenAIOptions...),
		}, nil
	case models.ProviderAzure:
		return &baseProvider{
			options: clientOptions.Options,
			client:  azureclient.NewClient(clientOptions.Options, clientOptions.OpenAIOptions...),
		}, nil
	case models.ProviderVertexAI:
		return &baseProvider{
			options: clientOptions.Options,
			client:  vertexaiclient.NewClient(clientOptions.Options, clientOptions.GeminiOptions...),
		}, nil
	case models.ProviderOpenRouter:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			openaiclient.WithDefaultBaseURL("https://openrouter.ai/api/v1"),
			openaiclient.WithExtraHeaders(map[string]string{
				"HTTP-Referer": "ferryer.ai",
				"X-Title":      "Ferryer",
			}),
		)
		return &baseProvider{
			options: clientOptions.Options,
			client:  openaiclient.NewClient(clientOptions.Options, clientOptions.OpenAIOptions...),
		}, nil
	case models.ProviderXAI:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			openaiclient.WithDefaultBaseURL("https://api.x.ai/v1"),
		)
		return &baseProvider{
			options: clientOptions.Options,
			client:  openaiclient.NewClient(clientOptions.Options, clientOptions.OpenAIOptions...),
		}, nil
	case models.ProviderLocal:
		clientOptions.OpenAIOptions = append(clientOptions.OpenAIOptions,
			openaiclient.WithDefaultBaseURL(os.Getenv("LOCAL_ENDPOINT")),
		)
		return &baseProvider{
			options: clientOptions.Options,
			client:  openaiclient.NewClient(clientOptions.Options, clientOptions.OpenAIOptions...),
		}, nil
	case models.ProviderMock:
		return &baseProvider{
			options: clientOptions.Options,
			client:  mockclient.NewClient(clientOptions.Options),
		}, nil
	}
	return nil, fmt.Errorf("provider not supported: %s", providerName)
}
