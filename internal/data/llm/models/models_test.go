package models

import "testing"

func TestResolveModelUsesMaxTokensFromCatalog(t *testing.T) {
	model := ResolveModel(ProviderOpenAI, "o4-mini")
	if model.MaxTokens != 20000 {
		t.Fatalf("expected catalog MaxTokens 20000, got %d", model.MaxTokens)
	}
}

func TestResolveModelDefaultsMaxTokens(t *testing.T) {
	model := ResolveModel(ProviderOpenAI, "unknown-model")
	if model.MaxTokens != 4096 {
		t.Fatalf("expected fallback MaxTokens 4096, got %d", model.MaxTokens)
	}
}
