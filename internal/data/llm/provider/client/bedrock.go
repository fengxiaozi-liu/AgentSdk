package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"ferryman-agent/internal/memory/message"
	toolcore "ferryman-agent/internal/tools"
)

type bedrockOptions struct {
	// Bedrock specific options can be added here
}

type BedrockOption func(*bedrockOptions)

type bedrockClient struct {
	providerOptions Options
	options         bedrockOptions
	childProvider   Client
}

type BedrockClient Client

func NewBedrockClient(opts Options) BedrockClient {
	bedrockOpts := bedrockOptions{}
	// Apply bedrock specific options if they are added in the future

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
		anthropicOpts := opts
		anthropicOpts.AnthropicOptions = append(anthropicOpts.AnthropicOptions,
			WithAnthropicBedrock(true),
			WithAnthropicDisableCache(),
		)
		return &bedrockClient{
			providerOptions: opts,
			options:         bedrockOpts,
			childProvider:   NewAnthropicClient(anthropicOpts),
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

func (b *bedrockClient) Send(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) (*Response, error) {
	if b.childProvider == nil {
		return nil, errors.New("unsupported model for bedrock provider")
	}
	return b.childProvider.Send(ctx, messages, tools)
}

func (b *bedrockClient) Stream(ctx context.Context, messages []message.Message, tools []toolcore.BaseTool) <-chan Event {
	eventChan := make(chan Event)

	if b.childProvider == nil {
		go func() {
			eventChan <- Event{
				Type:  EventError,
				Error: errors.New("unsupported model for bedrock provider"),
			}
			close(eventChan)
		}()
		return eventChan
	}

	return b.childProvider.Stream(ctx, messages, tools)
}
