package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceGetSystemPromptUsesMarkdownFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coder.md")
	t.Setenv("NOOP", "noop")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}

	promptSvc, err := NewService(PromptConfig{
		Type: PromptConfigPath,
		Path: path,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	text, err := promptSvc.GetSystemPrompt("coder")
	if err != nil {
		t.Fatalf("GetSystemPrompt: %v", err)
	}
	if text != "hello" {
		t.Fatalf("expected prompt %q, got %q", "hello", text)
	}

	_, err = promptSvc.GetSystemPrompt("missing")
	if !errors.Is(err, ErrPromptKeyNotFound) {
		t.Fatalf("expected ErrPromptKeyNotFound, got %v", err)
	}
}

func TestLoadPathSupportsMarkdownDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reviewer.md"), []byte("review carefully"), 0o644); err != nil {
		t.Fatalf("write markdown prompt: %v", err)
	}

	promptSvc, err := LoadPath(dir)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	text, err := promptSvc.GetSystemPrompt("reviewer")
	if err != nil {
		t.Fatalf("GetSystemPrompt: %v", err)
	}
	if text != "review carefully" {
		t.Fatalf("expected markdown prompt, got %q", text)
	}
}

func TestNewServiceSupportsConfigValue(t *testing.T) {
	promptSvc, err := NewService(PromptConfig{
		Type:  PromptConfigValue,
		Key:   "reviewer",
		Value: "review carefully",
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	text, err := promptSvc.GetSystemPrompt("reviewer")
	if err != nil {
		t.Fatalf("GetSystemPrompt: %v", err)
	}
	if text != "review carefully" {
		t.Fatalf("expected prompt value, got %q", text)
	}
}

func TestServiceSetPromptAndKeys(t *testing.T) {
	promptSvc := New()
	promptSvc.SetPrompt("coder", "code carefully")
	promptSvc.SetPrompts(map[string]string{"task": "run task"})

	if !promptSvc.Has("coder") {
		t.Fatal("expected coder prompt")
	}
	keys := promptSvc.Keys()
	if len(keys) != 2 || keys[0] != "coder" || keys[1] != "task" {
		t.Fatalf("unexpected keys: %+v", keys)
	}
}
