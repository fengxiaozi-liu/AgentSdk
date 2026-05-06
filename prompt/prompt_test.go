package prompt

import (
	"errors"
	"sync"
	"testing"
)

func TestResolveSystemPromptByKeyUsesDefaultConfig(t *testing.T) {
	oncePrompts = sync.Once{}
	prompts = nil
	promptsErr = nil

	text, err := ResolveSystemPromptByKey("coder")
	if err != nil {
		t.Fatalf("ResolveSystemPromptByKey: %v", err)
	}
	if text == "" {
		t.Fatal("expected coder prompt text")
	}

	_, err = ResolveSystemPromptByKey("missing")
	if !errors.Is(err, ErrPromptKeyNotFound) {
		t.Fatalf("expected ErrPromptKeyNotFound, got %v", err)
	}
}

func TestParsePromptConfigSupportsYAML(t *testing.T) {
	prompts, err := parsePromptConfig("prompts.yaml", []byte("prompts:\n  coder: hello\n"))
	if err != nil {
		t.Fatalf("parsePromptConfig: %v", err)
	}
	if prompts["coder"] != "hello" {
		t.Fatalf("unexpected parsed prompt: %+v", prompts)
	}
}
