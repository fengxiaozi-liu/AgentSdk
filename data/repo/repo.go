package repo

import (
	"context"
	"errors"
)

var ErrRepoNotFound = errors.New("repo item not found")

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

type CreateSessionParams struct {
	ID               string
	ParentSessionID  string
	Title            string
	MessageCount     int64
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
}

type UpdateSessionParams struct {
	ID               string
	Title            string
	PromptTokens     int64
	CompletionTokens int64
	SummaryMessageID string
	Cost             float64
}

type SessionRepo interface {
	Create(context.Context, CreateSessionParams) (Session, error)
	Get(context.Context, string) (Session, error)
	ListRoot(context.Context) ([]Session, error)
	Update(context.Context, UpdateSessionParams) (Session, error)
	Delete(context.Context, string) error
	IncrementMessageCount(context.Context, string, int64) error
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

type CreateMessageParams struct {
	ID        string
	SessionID string
	Role      string
	Parts     string
	Model     string
}

type UpdateMessageParams struct {
	ID         string
	Parts      string
	FinishedAt int64
}

type MessageRepo interface {
	Create(context.Context, CreateMessageParams) (Message, error)
	Update(context.Context, UpdateMessageParams) error
	Get(context.Context, string) (Message, error)
	ListBySession(context.Context, string) ([]Message, error)
	Delete(context.Context, string) error
	DeleteBySession(context.Context, string) error
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

type CreateFileParams struct {
	ID        string
	SessionID string
	Path      string
	Content   string
	Version   string
}

type UpdateFileParams struct {
	ID      string
	Content string
	Version string
}

type HistoryRepo interface {
	Create(context.Context, CreateFileParams) (File, error)
	Get(context.Context, string) (File, error)
	GetLatestByPathAndSession(context.Context, string, string) (File, error)
	ListByPath(context.Context, string) ([]File, error)
	ListBySession(context.Context, string) ([]File, error)
	ListLatestBySession(context.Context, string) ([]File, error)
	Update(context.Context, UpdateFileParams) (File, error)
	Delete(context.Context, string) error
	DeleteBySession(context.Context, string) error
}
