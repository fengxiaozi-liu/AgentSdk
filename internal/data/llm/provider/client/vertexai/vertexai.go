package client

import (
	"context"
	"os"

	llmclient "ferryman-agent/internal/data/llm/provider/client"
	geminiclient "ferryman-agent/internal/data/llm/provider/client/gemini"
	"ferryman-agent/internal/data/logging"
	"google.golang.org/genai"
)

func NewClient(opts llmclient.Options, optionFns ...geminiclient.Option) llmclient.Client {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:  os.Getenv("VERTEXAI_PROJECT"),
		Location: os.Getenv("VERTEXAI_LOCATION"),
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		logging.Error("Failed to create VertexAI client", "error", err)
		return nil
	}

	return geminiclient.NewClientWithGenAI(opts, client, optionFns...)
}
