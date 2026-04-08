package memory

import (
	"context"

	"github.com/opencode-ai/opencode/agent/history"
	"github.com/opencode-ai/opencode/agent/message"
	"github.com/opencode-ai/opencode/agent/session"
)

type Snapshot struct {
	Session  session.Session
	Messages []message.Message
	Files    []history.File
}

type Service interface {
	Sessions() session.Service
	Messages() message.Service
	History() history.Service
	Load(ctx context.Context, sessionID string) (Snapshot, error)
}

type service struct {
	sessions session.Service
	messages message.Service
	history  history.Service
}

func NewService(
	sessions session.Service,
	messages message.Service,
	history history.Service,
) Service {
	return &service{
		sessions: sessions,
		messages: messages,
		history:  history,
	}
}

func (s *service) Sessions() session.Service {
	return s.sessions
}

func (s *service) Messages() message.Service {
	return s.messages
}

func (s *service) History() history.Service {
	return s.history
}

func (s *service) Load(ctx context.Context, sessionID string) (Snapshot, error) {
	currentSession, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return Snapshot{}, err
	}
	messages, err := s.messages.List(ctx, sessionID)
	if err != nil {
		return Snapshot{}, err
	}
	files, err := s.history.ListLatestSessionFiles(ctx, sessionID)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Session:  currentSession,
		Messages: messages,
		Files:    files,
	}, nil
}
