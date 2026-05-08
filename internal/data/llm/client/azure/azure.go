package client

import (
	llmclient "ferryman-agent/internal/data/llm/client"
	client3 "ferryman-agent/internal/data/llm/client/openai"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

type azureClient struct {
	llmclient.Client
}

func NewClient(opts llmclient.Options, optionFns ...client3.Option) llmclient.Client {

	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")      // ex: https://foo.openai.azure.com
	apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION") // ex: 2025-04-01-preview

	if endpoint == "" || apiVersion == "" {
		return &azureClient{Client: client3.NewClient(opts, optionFns...)}
	}

	reqOpts := []option.RequestOption{
		azure.WithEndpoint(endpoint, apiVersion),
	}

	if opts.APIKey != "" || os.Getenv("AZURE_OPENAI_API_KEY") != "" {
		key := opts.APIKey
		if key == "" {
			key = os.Getenv("AZURE_OPENAI_API_KEY")
		}
		reqOpts = append(reqOpts, azure.WithAPIKey(key))
	} else if cred, err := azidentity.NewDefaultAzureCredential(nil); err == nil {
		reqOpts = append(reqOpts, azure.WithTokenCredential(cred))
	}

	base := client3.NewClientWithOpenAI(opts, openai.NewClient(reqOpts...), optionFns...)

	return &azureClient{Client: base}
}
