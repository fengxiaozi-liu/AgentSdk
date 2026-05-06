package repo

import (
	"context"
	"sort"

	datadb "ferryman-agent/data/db"
)

type Repositories struct {
	Sessions SessionRepo
	Messages MessageRepo
	History  HistoryRepo
}

func NewRepositories(source *datadb.Source) Repositories {
	if source == nil {
		source = datadb.NewSource()
	}
	return Repositories{
		Sessions: &sessionRepo{source: source},
		Messages: &messageRepo{source: source},
		History:  &historyRepo{source: source},
	}
}

func nextTime(source *datadb.Source) int64 {
	source.NextTime++
	return source.NextTime
}

type sessionRepo struct {
	source *datadb.Source
}

func (r *sessionRepo) Create(ctx context.Context, params CreateSessionParams) (Session, error) {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	ts := nextTime(r.source)
	item := datadb.Session{
		ID:               params.ID,
		ParentSessionID:  params.ParentSessionID,
		Title:            params.Title,
		MessageCount:     params.MessageCount,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		Cost:             params.Cost,
		CreatedAt:        ts,
		UpdatedAt:        ts,
	}
	r.source.Sessions[item.ID] = item
	return fromDBSession(item), nil
}

func (r *sessionRepo) Get(ctx context.Context, id string) (Session, error) {
	r.source.Mu.RLock()
	defer r.source.Mu.RUnlock()
	item, ok := r.source.Sessions[id]
	if !ok {
		return Session{}, ErrRepoNotFound
	}
	return fromDBSession(item), nil
}

func (r *sessionRepo) ListRoot(ctx context.Context) ([]Session, error) {
	r.source.Mu.RLock()
	defer r.source.Mu.RUnlock()
	items := []Session{}
	for _, item := range r.source.Sessions {
		if item.ParentSessionID == "" {
			items = append(items, fromDBSession(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func (r *sessionRepo) Update(ctx context.Context, params UpdateSessionParams) (Session, error) {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	item, ok := r.source.Sessions[params.ID]
	if !ok {
		return Session{}, ErrRepoNotFound
	}
	item.Title = params.Title
	item.PromptTokens = params.PromptTokens
	item.CompletionTokens = params.CompletionTokens
	item.SummaryMessageID = params.SummaryMessageID
	item.Cost = params.Cost
	item.UpdatedAt = nextTime(r.source)
	r.source.Sessions[item.ID] = item
	return fromDBSession(item), nil
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	if _, ok := r.source.Sessions[id]; !ok {
		return ErrRepoNotFound
	}
	delete(r.source.Sessions, id)
	return nil
}

func (r *sessionRepo) IncrementMessageCount(ctx context.Context, id string, delta int64) error {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	item, ok := r.source.Sessions[id]
	if !ok {
		return ErrRepoNotFound
	}
	item.MessageCount += delta
	if item.MessageCount < 0 {
		item.MessageCount = 0
	}
	item.UpdatedAt = nextTime(r.source)
	r.source.Sessions[id] = item
	return nil
}

type messageRepo struct {
	source *datadb.Source
}

func (r *messageRepo) Create(ctx context.Context, params CreateMessageParams) (Message, error) {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	ts := nextTime(r.source)
	item := datadb.Message{
		ID:        params.ID,
		SessionID: params.SessionID,
		Role:      params.Role,
		Parts:     params.Parts,
		Model:     params.Model,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	r.source.Messages[item.ID] = item
	if session, ok := r.source.Sessions[item.SessionID]; ok {
		session.MessageCount++
		session.UpdatedAt = ts
		r.source.Sessions[item.SessionID] = session
	}
	return fromDBMessage(item), nil
}

func (r *messageRepo) Update(ctx context.Context, params UpdateMessageParams) error {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	item, ok := r.source.Messages[params.ID]
	if !ok {
		return ErrRepoNotFound
	}
	item.Parts = params.Parts
	item.FinishedAt = params.FinishedAt
	item.UpdatedAt = nextTime(r.source)
	r.source.Messages[item.ID] = item
	return nil
}

func (r *messageRepo) Get(ctx context.Context, id string) (Message, error) {
	r.source.Mu.RLock()
	defer r.source.Mu.RUnlock()
	item, ok := r.source.Messages[id]
	if !ok {
		return Message{}, ErrRepoNotFound
	}
	return fromDBMessage(item), nil
}

func (r *messageRepo) ListBySession(ctx context.Context, sessionID string) ([]Message, error) {
	r.source.Mu.RLock()
	defer r.source.Mu.RUnlock()
	items := []Message{}
	for _, item := range r.source.Messages {
		if item.SessionID == sessionID {
			items = append(items, fromDBMessage(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items, nil
}

func (r *messageRepo) Delete(ctx context.Context, id string) error {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	item, ok := r.source.Messages[id]
	if !ok {
		return ErrRepoNotFound
	}
	delete(r.source.Messages, id)
	if session, ok := r.source.Sessions[item.SessionID]; ok {
		session.MessageCount--
		if session.MessageCount < 0 {
			session.MessageCount = 0
		}
		session.UpdatedAt = nextTime(r.source)
		r.source.Sessions[item.SessionID] = session
	}
	return nil
}

func (r *messageRepo) DeleteBySession(ctx context.Context, sessionID string) error {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	deleted := int64(0)
	for id, item := range r.source.Messages {
		if item.SessionID == sessionID {
			delete(r.source.Messages, id)
			deleted++
		}
	}
	if session, ok := r.source.Sessions[sessionID]; ok {
		session.MessageCount -= deleted
		if session.MessageCount < 0 {
			session.MessageCount = 0
		}
		session.UpdatedAt = nextTime(r.source)
		r.source.Sessions[sessionID] = session
	}
	return nil
}

type historyRepo struct {
	source *datadb.Source
}

func (r *historyRepo) Create(ctx context.Context, params CreateFileParams) (File, error) {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	ts := nextTime(r.source)
	item := datadb.File{
		ID:        params.ID,
		SessionID: params.SessionID,
		Path:      params.Path,
		Content:   params.Content,
		Version:   params.Version,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	r.source.Files[item.ID] = item
	return fromDBFile(item), nil
}

func (r *historyRepo) Get(ctx context.Context, id string) (File, error) {
	r.source.Mu.RLock()
	defer r.source.Mu.RUnlock()
	item, ok := r.source.Files[id]
	if !ok {
		return File{}, ErrRepoNotFound
	}
	return fromDBFile(item), nil
}

func (r *historyRepo) GetLatestByPathAndSession(ctx context.Context, path, sessionID string) (File, error) {
	items, err := r.ListByPath(ctx, path)
	if err != nil {
		return File{}, err
	}
	for _, item := range items {
		if item.SessionID == sessionID {
			return item, nil
		}
	}
	return File{}, ErrRepoNotFound
}

func (r *historyRepo) ListByPath(ctx context.Context, path string) ([]File, error) {
	r.source.Mu.RLock()
	defer r.source.Mu.RUnlock()
	items := []File{}
	for _, item := range r.source.Files {
		if item.Path == path {
			items = append(items, fromDBFile(item))
		}
	}
	sortFilesDesc(items)
	return items, nil
}

func (r *historyRepo) ListBySession(ctx context.Context, sessionID string) ([]File, error) {
	r.source.Mu.RLock()
	defer r.source.Mu.RUnlock()
	items := []File{}
	for _, item := range r.source.Files {
		if item.SessionID == sessionID {
			items = append(items, fromDBFile(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items, nil
}

func (r *historyRepo) ListLatestBySession(ctx context.Context, sessionID string) ([]File, error) {
	all, err := r.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	latest := map[string]File{}
	for _, item := range all {
		current, ok := latest[item.Path]
		if !ok || item.CreatedAt >= current.CreatedAt {
			latest[item.Path] = item
		}
	}
	items := make([]File, 0, len(latest))
	for _, item := range latest {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})
	return items, nil
}

func (r *historyRepo) Update(ctx context.Context, params UpdateFileParams) (File, error) {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	item, ok := r.source.Files[params.ID]
	if !ok {
		return File{}, ErrRepoNotFound
	}
	item.Content = params.Content
	item.Version = params.Version
	item.UpdatedAt = nextTime(r.source)
	r.source.Files[item.ID] = item
	return fromDBFile(item), nil
}

func (r *historyRepo) Delete(ctx context.Context, id string) error {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	if _, ok := r.source.Files[id]; !ok {
		return ErrRepoNotFound
	}
	delete(r.source.Files, id)
	return nil
}

func (r *historyRepo) DeleteBySession(ctx context.Context, sessionID string) error {
	r.source.Mu.Lock()
	defer r.source.Mu.Unlock()
	for id, item := range r.source.Files {
		if item.SessionID == sessionID {
			delete(r.source.Files, id)
		}
	}
	return nil
}

func sortFilesDesc(items []File) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
}

func fromDBSession(item datadb.Session) Session {
	return Session(item)
}

func fromDBMessage(item datadb.Message) Message {
	return Message(item)
}

func fromDBFile(item datadb.File) File {
	return File(item)
}
