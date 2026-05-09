package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DirectoryPrompt struct {
	*promptStore
}

func NewDirectoryPrompt(path string) (Service, error) {
	prompts, err := loadMarkdownPromptDir(path)
	if err != nil {
		return nil, err
	}
	return &DirectoryPrompt{promptStore: newPromptStoreWith(prompts)}, nil
}

func LoadPath(path string) (Service, error) {
	return NewDirectoryPrompt(path)
}

func loadMarkdownPromptDir(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrPromptConfigMissing
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: prompt path must be a directory", ErrPromptConfigInvalid)
	}

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
