package client

import (
	"os"

	llmclient "ferryman-agent/internal/data/llm/provider/client"
	openaiclient "ferryman-agent/internal/data/llm/provider/client/openai"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/azure"
	"github.com/openai/openai-go/option"
)

type azureClient struct {
	llmclient.Client
}

func NewClient(opts llmclient.Options, optionFns ...openaiclient.Option) llmclient.Client {

	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")      // ex: https://foo.openai.azure.com
	apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION") // ex: 2025-04-01-preview

	if endpoint == "" || apiVersion == "" {
		return &azureClient{Client: openaiclient.NewClient(opts, optionFns...)}
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

	base := openaiclient.NewClientWithOpenAI(opts, openai.NewClient(reqOpts...), optionFns...)

	return &azureClient{Client: base}
}
