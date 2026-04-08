package memory

import (
	"context"
	"testing"

	"ferryman-agent/history"
	"ferryman-agent/message"
	"ferryman-agent/pubsub"
	"ferryman-agent/session"
)

type sessionStub struct {
	get func(context.Context, string) (session.Session, error)
}

func (s sessionStub) Subscribe(context.Context) <-chan pubsub.Event[session.Session] { return nil }
func (s sessionStub) Create(context.Context, string) (session.Session, error) {
	panic("unexpected call")
}
func (s sessionStub) CreateTitleSession(context.Context, string) (session.Session, error) {
	panic("unexpected call")
}
func (s sessionStub) CreateTaskSession(context.Context, string, string, string) (session.Session, error) {
	panic("unexpected call")
}
func (s sessionStub) Get(ctx context.Context, id string) (session.Session, error) {
	return s.get(ctx, id)
}
func (s sessionStub) List(context.Context) ([]session.Session, error) { panic("unexpected call") }
func (s sessionStub) Save(context.Context, session.Session) (session.Session, error) {
	panic("unexpected call")
}
func (s sessionStub) Delete(context.Context, string) error { panic("unexpected call") }

type messageStub struct {
	list func(context.Context, string) ([]message.Message, error)
}

func (m messageStub) Subscribe(context.Context) <-chan pubsub.Event[message.Message] { return nil }
func (m messageStub) Create(context.Context, string, message.CreateMessageParams) (message.Message, error) {
	panic("unexpected call")
}
func (m messageStub) Update(context.Context, message.Message) error { panic("unexpected call") }
func (m messageStub) Get(context.Context, string) (message.Message, error) {
	panic("unexpected call")
}
func (m messageStub) List(ctx context.Context, id string) ([]message.Message, error) {
	return m.list(ctx, id)
}
func (m messageStub) Delete(context.Context, string) error                { panic("unexpected call") }
func (m messageStub) DeleteSessionMessages(context.Context, string) error { panic("unexpected call") }

type historyStub struct {
	list func(context.Context, string) ([]history.File, error)
}

func (h historyStub) Subscribe(context.Context) <-chan pubsub.Event[history.File] { return nil }
func (h historyStub) Create(context.Context, string, string, string) (history.File, error) {
	panic("unexpected call")
}
func (h historyStub) CreateVersion(context.Context, string, string, string) (history.File, error) {
	panic("unexpected call")
}
func (h historyStub) Get(context.Context, string) (history.File, error) { panic("unexpected call") }
func (h historyStub) GetByPathAndSession(context.Context, string, string) (history.File, error) {
	panic("unexpected call")
}
func (h historyStub) ListBySession(context.Context, string) ([]history.File, error) {
	panic("unexpected call")
}
func (h historyStub) ListLatestSessionFiles(ctx context.Context, id string) ([]history.File, error) {
	return h.list(ctx, id)
}
func (h historyStub) Update(context.Context, history.File) (history.File, error) {
	panic("unexpected call")
}
func (h historyStub) Delete(context.Context, string) error             { panic("unexpected call") }
func (h historyStub) DeleteSessionFiles(context.Context, string) error { panic("unexpected call") }

func TestLoadReturnsSnapshot(t *testing.T) {
	t.Parallel()

	mem := NewService(
		sessionStub{get: func(context.Context, string) (session.Session, error) {
			return session.Session{ID: "s1", Title: "chat"}, nil
		}},
		messageStub{list: func(context.Context, string) ([]message.Message, error) {
			return []message.Message{{ID: "m1"}}, nil
		}},
		historyStub{list: func(context.Context, string) ([]history.File, error) {
			return []history.File{{ID: "f1"}}, nil
		}},
	)

	snapshot, err := mem.Load(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if snapshot.Session.ID != "s1" {
		t.Fatalf("expected session id s1, got %q", snapshot.Session.ID)
	}
	if len(snapshot.Messages) != 1 || snapshot.Messages[0].ID != "m1" {
		t.Fatalf("unexpected messages snapshot: %+v", snapshot.Messages)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].ID != "f1" {
		t.Fatalf("unexpected files snapshot: %+v", snapshot.Files)
	}
}
