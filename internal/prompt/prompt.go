package prompt

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrPromptKeyNotFound   = errors.New("prompt key not found")
	ErrPromptConfigInvalid = errors.New("prompt config invalid")
	ErrPromptConfigMissing = errors.New("prompt config path is required")
)

const (
	KeyCoder      = "coder"
	KeyTitle      = "title"
	KeyTask       = "task"
	KeySummarizer = "summarizer"
)

type Service interface {
	GetSystemPrompt(ctx context.Context, key string) (string, error)
	Has(key string) bool
	Keys() []string
	SetPrompt(key, value string)
	SetPrompts(prompts map[string]string)
}

type promptStore struct {
	prompts map[string]string
}

type DefaultPrompt struct {
	*promptStore
}

func NewDefault() Service {
	return &DefaultPrompt{promptStore: newPromptStoreWith(defaultPrompts())}
}

func New() Service {
	return newPromptStore()
}

func (p *promptStore) GetSystemPrompt(ctx context.Context, key string) (string, error) {
	_ = ctx

	if p == nil {
		return "", fmt.Errorf("%w: %s", ErrPromptKeyNotFound, key)
	}
	prompt, ok := p.prompts[key]
	if !ok || strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("%w: %s", ErrPromptKeyNotFound, key)
	}
	return prompt, nil
}

func (p *promptStore) Has(key string) bool {
	if p == nil {
		return false
	}
	_, ok := p.prompts[key]
	return ok
}

func (p *promptStore) Keys() []string {
	if p == nil {
		return nil
	}
	keys := make([]string, 0, len(p.prompts))
	for key := range p.prompts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (p *promptStore) SetPrompt(key, value string) {
	if p == nil {
		return
	}
	if p.prompts == nil {
		p.prompts = make(map[string]string)
	}
	p.prompts[key] = value
}

func (p *promptStore) SetPrompts(prompts map[string]string) {
	for key, value := range prompts {
		p.SetPrompt(key, value)
	}
}

func newPromptStore() *promptStore {
	return &promptStore{prompts: make(map[string]string)}
}

func newPromptStoreWith(prompts map[string]string) *promptStore {
	store := newPromptStore()
	store.SetPrompts(prompts)
	return store
}

func defaultPrompts() map[string]string {
	return map[string]string{
		KeyCoder:      "You are an Agent SDK coding assistant. Help the user understand and modify code safely. Prefer reading relevant files before changing them, keep edits scoped, and verify changes with tests when possible.",
		KeyTitle:      "Generate a concise one-line title for the user's first message. Return only the title text.",
		KeyTask:       "You are a focused task agent. Complete the delegated task using the available tools and return the result clearly.",
		KeySummarizer: "Summarize the conversation clearly and concisely, preserving decisions, changed files, open issues, and next steps.",
	}
}
