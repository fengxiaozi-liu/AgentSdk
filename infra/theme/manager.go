package theme

import "sync"

type Manager struct {
	themes      map[string]Theme
	currentName string
	mu          sync.RWMutex
}

var globalManager = &Manager{
	themes: make(map[string]Theme),
}

func RegisterTheme(name string, theme Theme) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()
	globalManager.themes[name] = theme
	if globalManager.currentName == "" {
		globalManager.currentName = name
	}
}

func CurrentTheme() Theme {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()
	return globalManager.themes[globalManager.currentName]
}

func SetTheme(name string) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()
	if _, ok := globalManager.themes[name]; ok {
		globalManager.currentName = name
	}
}
