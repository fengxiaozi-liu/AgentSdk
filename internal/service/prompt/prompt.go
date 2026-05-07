package prompt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrPromptKeyNotFound   = errors.New("prompt key not found")
	ErrPromptConfigInvalid = errors.New("prompt config invalid")
	ErrPromptConfigMissing = errors.New("prompt config path is required")
)

type PromptConfigType string

const (
	PromptConfigPath  PromptConfigType = "path"
	PromptConfigValue PromptConfigType = "value"
)

type PromptConfig struct {
	Type  PromptConfigType `json:"type,omitempty"`
	Key   string           `json:"key,omitempty"`
	Value string           `json:"value,omitempty"`
	Path  string           `json:"path,omitempty"`
}

const (
	KeyCoder      = "coder"
	KeyTitle      = "title"
	KeyTask       = "task"
	KeySummarizer = "summarizer"
)

type Service interface {
	GetSystemPrompt(key string) (string, error)
	Has(key string) bool
	Keys() []string
	SetPrompt(key, value string)
	SetPrompts(prompts map[string]string)
}

type service struct {
	prompts map[string]string
}

func NewService(cfg PromptConfig) (Service, error) {
	promptSvc := New()
	switch cfg.Type {
	case PromptConfigValue:
		if strings.TrimSpace(cfg.Key) != "" {
			promptSvc.SetPrompt(cfg.Key, cfg.Value)
		}
		return promptSvc, nil
	case PromptConfigPath:
		if strings.TrimSpace(cfg.Path) == "" {
			return promptSvc, nil
		}
		prompts, err := loadPrompts(cfg.Path)
		if err != nil {
			return nil, err
		}
		promptSvc.SetPrompts(prompts)
		return promptSvc, nil
	}
	return promptSvc, nil
}

func New() Service {
	return &service{prompts: make(map[string]string)}
}

func (p *service) GetSystemPrompt(key string) (string, error) {
	if p == nil {
		return "", fmt.Errorf("%w: %s", ErrPromptKeyNotFound, key)
	}
	prompt, ok := p.prompts[key]
	if !ok || strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("%w: %s", ErrPromptKeyNotFound, key)
	}
	return prompt, nil
}

func (p *service) Has(key string) bool {
	if p == nil {
		return false
	}
	_, ok := p.prompts[key]
	return ok
}

func (p *service) Keys() []string {
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

func (p *service) SetPrompt(key, value string) {
	if p == nil {
		return
	}
	if p.prompts == nil {
		p.prompts = make(map[string]string)
	}
	p.prompts[key] = value
}

func (p *service) SetPrompts(prompts map[string]string) {
	for key, value := range prompts {
		p.SetPrompt(key, value)
	}
}

func LoadPath(path string) (Service, error) {
	prompts, err := loadPrompts(path)
	if err != nil {
		return nil, err
	}
	return &service{prompts: prompts}, nil
}

func loadPrompts(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrPromptConfigMissing
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return loadPromptDir(path)
	}
	return loadPromptFile(path)
}

func loadPromptFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		key := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return map[string]string{key: string(content)}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported prompt file type %s", ErrPromptConfigInvalid, filepath.Ext(path))
	}
}

func loadPromptDir(path string) (map[string]string, error) {
	prompts := make(map[string]string)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		key := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		content, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		if _, exists := prompts[key]; exists {
			return nil, fmt.Errorf("%w: duplicate prompt key %s", ErrPromptConfigInvalid, key)
		}
		prompts[key] = string(content)
	}
	if len(prompts) == 0 {
		return nil, fmt.Errorf("%w: no markdown prompts found in %s", ErrPromptConfigInvalid, path)
	}
	return prompts, nil
}
