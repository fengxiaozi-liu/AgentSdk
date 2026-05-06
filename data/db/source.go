package db

import "sync"

type Source struct {
	Mu       sync.RWMutex
	NextTime int64
	Sessions map[string]Session
	Messages map[string]Message
	Files    map[string]File
}

type Session struct {
	ID               string
	ParentSessionID  string
	Title            string
	MessageCount     int64
	PromptTokens     int64
	CompletionTokens int64
	SummaryMessageID string
	Cost             float64
	CreatedAt        int64
	UpdatedAt        int64
}

type Message struct {
	ID         string
	SessionID  string
	Role       string
	Parts      string
	Model      string
	FinishedAt int64
	CreatedAt  int64
	UpdatedAt  int64
}

type File struct {
	ID        string
	SessionID string
	Path      string
	Content   string
	Version   string
	CreatedAt int64
	UpdatedAt int64
}

func NewSource() *Source {
	return &Source{
		NextTime: 1,
		Sessions: map[string]Session{},
		Messages: map[string]Message{},
		Files:    map[string]File{},
	}
}
