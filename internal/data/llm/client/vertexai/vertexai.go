package client

import (
	"context"
	llmclient "ferryman-agent/internal/data/llm/client"
	client3 "ferryman-agent/internal/data/llm/client/gemini"
	"os"

	"ferryman-agent/internal/data/logging"

	"google.golang.org/genai"
)

func NewClient(_ string, optionFns ...client3.Option) llmclient.Client {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:  os.Getenv("VERTEXAI_PROJECT"),
		Location: os.Getenv("VERTEXAI_LOCATION"),
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		logging.Error("Failed to create VertexAI client", "error", err)
		return nil
	}

	return client3.NewClientWithGenAI(client, optionFns...)
}
