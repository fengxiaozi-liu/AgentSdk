package client

import (
	"context"
	"os"

	"ferryman-agent/internal/data/logging"
	"google.golang.org/genai"
)

type VertexAIClient Client

func NewVertexAIClient(opts Options) VertexAIClient {
	geminiOpts := geminiOptions{}
	for _, o := range opts.GeminiOptions {
		o(&geminiOpts)
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:  os.Getenv("VERTEXAI_PROJECT"),
		Location: os.Getenv("VERTEXAI_LOCATION"),
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		logging.Error("Failed to create VertexAI client", "error", err)
		return nil
	}

	return &geminiClient{
		providerOptions: opts,
		options:         geminiOpts,
		client:          client,
	}
}
