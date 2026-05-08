package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	llmclient "ferryman-agent/internal/data/llm/provider/client"
	anthropicclient "ferryman-agent/internal/data/llm/provider/client/anthropic"
	"ferryman-agent/internal/memory/message"
	toolcore "ferryman-agent/internal/tools"
)

type bedrockClient struct {
	providerOptions llmclient.Options
	options         options
	childProvider   llmclient.Client
}

func NewClient(opts llmclient.Options, optionFns ...Option) llmclient.Client {
	bedrockOpts := options{}
	for _, o := range optionFns {
		o(&bedrockOpts)
	}

	// Get AWS region from environment
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}

	if region == "" {
		region = "us-east-1" // default region
	}
	if len(region) < 2 {
		return &bedrockClient{
			providerOptions: opts,
			options:         bedrockOpts,
			childProvider:   nil, // Will cause an error when used
		}
	}

	// Prefix the model name with region
	regionPrefix := region[:2]
	modelName := opts.Model.APIModel
	opts.Model.APIModel = fmt.Sprintf("%s.%s", regionPrefix, modelName)

	// Determine which provider to use based on the model
	if strings.Contains(string(opts.Model.APIModel), "anthropic") {
		// Create Anthropic client with Bedrock configuration
		return &bedrockClient{
			providerOptions: opts,
			options:         bedrockOpts,
			childProvider:   anthropicclient.NewClient(opts, anthropicclient.WithBedrock(true), anthropicclient.WithDisableCache()),
		}
	}

	// Return client with nil childProvider if model is not supported
	// This will cause an error when used
	return &bedrockClient{
		providerOptions: opts,
		options:         bedrockOpts,
		childProvider:   nil,
	}
}

func (b *bedrockClient) Send(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) (*llmclient.Response, error) {
	if b.childProvider == nil {
		return nil, errors.New("unsupported model for bedrock provider")
	}
	return b.childProvider.Send(ctx, messages, tools)
}

func (b *bedrockClient) Stream(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) <-chan llmclient.Event {
	eventChan := make(chan llmclient.Event)

	if b.childProvider == nil {
		go func() {
			eventChan <- llmclient.Event{
				Type:  llmclient.EventError,
				Error: errors.New("unsupported model for bedrock provider"),
			}
			close(eventChan)
		}()
		return eventChan
	}

	return b.childProvider.Stream(ctx, messages, tools)
}
