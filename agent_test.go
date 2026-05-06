package agent

import (
	"path/filepath"
	"testing"

	sdkconfig "ferryman-agent/config"
	"ferryman-agent/llm/models"
)

func TestCreateAgentProviderAllowsConfiguredArbitraryModel(t *testing.T) {
	sdkconfigTestReset()
	workingDir := t.TempDir()

	if _, err := sdkconfig.Use(sdkconfig.Config{
		WorkingDir: workingDir,
		Data:       sdkconfig.Data{Directory: filepath.Join(workingDir, "data")},
		Providers:  map[models.ModelProvider]sdkconfig.Provider{models.ProviderMock: {APIKey: "test-key"}},
		ModelProfiles: map[string]sdkconfig.ModelConfig{
			"coder": {
				Model:     "future-model",
				Provider:  models.ProviderMock,
				MaxTokens: 128,
			},
		},
	}); err != nil {
		t.Fatalf("use config: %v", err)
	}

	provider, err := createAgentProvider("coder")
	if err != nil {
		t.Fatalf("createAgentProvider: %v", err)
	}
	model := provider.Model()
	if model.ID != models.ModelID("future-model") {
		t.Fatalf("unexpected provider model: %+v", model)
	}
}

func sdkconfigTestReset() {
	sdkconfig.Use(sdkconfig.Config{})
}
