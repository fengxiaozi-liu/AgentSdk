package history

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/pubsub"

	"github.com/google/uuid"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewService)

const InitialVersion = "initial"

type File struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Content   string `json:"content"`
	Version   string `json:"version"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type Service interface {
	pubsub.Subscriber[File]
	Create(ctx context.Context, sessionID, path, content string) (File, error)
	CreateVersion(ctx context.Context, sessionID, path, content string) (File, error)
	Get(ctx context.Context, id string) (File, error)
	GetByPathAndSession(ctx context.Context, path, sessionID string) (File, error)
	ListBySession(ctx context.Context, sessionID string) ([]File, error)
	ListLatestSessionFiles(ctx context.Context, sessionID string) ([]File, error)
	Update(ctx context.Context, file File) (File, error)
	Delete(ctx context.Context, id string) error
	DeleteSessionFiles(ctx context.Context, sessionID string) error
}

type service struct {
	*pubsub.Broker[File]
	repo repo.HistoryRepo
}

func NewService(historyRepo repo.HistoryRepo) Service {
	return &service{Broker: pubsub.NewBroker[File](), repo: historyRepo}
}

func (s *service) Create(ctx context.Context, sessionID, path, content string) (File, error) {
	return s.createWithVersion(ctx, sessionID, path, content, InitialVersion)
}

func (s *service) CreateVersion(ctx context.Context, sessionID, path, content string) (File, error) {
	files, err := s.repo.ListByPath(ctx, path)
	if err != nil {
		return File{}, err
	}
	if len(files) == 0 {
		return s.Create(ctx, sessionID, path, content)
	}

	latestFile := files[0]
	latestVersion := latestFile.Version
	var nextVersion string
	if latestVersion == InitialVersion {
		nextVersion = "v1"
	} else if strings.HasPrefix(latestVersion, "v") {
		versionNum, err := strconv.Atoi(latestVersion[1:])
		if err != nil {
			nextVersion = fmt.Sprintf("v%d", latestFile.CreatedAt)
		} else {
			nextVersion = fmt.Sprintf("v%d", versionNum+1)
		}
	} else {
		nextVersion = fmt.Sprintf("v%d", latestFile.CreatedAt)
	}
	return s.createWithVersion(ctx, sessionID, path, content, nextVersion)
}

func (s *service) createWithVersion(ctx context.Context, sessionID, path, content, version string) (File, error) {
	dbFile, err := s.repo.Create(ctx, repo.HistoryRecord{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Path:      path,
		Content:   content,
		Version:   version,
	})
	if err != nil {
		return File{}, err
	}
	file := s.fromRepoItem(dbFile)
	s.Publish(pubsub.CreatedEvent, file)
	return file, nil
}

func (s *service) Get(ctx context.Context, id string) (File, error) {
	dbFile, err := s.repo.Get(ctx, id)
	if err != nil {
		return File{}, err
	}
	return s.fromRepoItem(dbFile), nil
}

func (s *service) GetByPathAndSession(ctx context.Context, path, sessionID string) (File, error) {
	dbFile, err := s.repo.GetLatestByPathAndSession(ctx, path, sessionID)
	if err != nil {
		return File{}, err
	}
	return s.fromRepoItem(dbFile), nil
}

func (s *service) ListBySession(ctx context.Context, sessionID string) ([]File, error) {
	dbFiles, err := s.repo.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromRepoItem(dbFile)
	}
	return files, nil
}

func (s *service) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]File, error) {
	dbFiles, err := s.repo.ListLatestBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromRepoItem(dbFile)
	}
	return files, nil
}

func (s *service) Update(ctx context.Context, file File) (File, error) {
	dbFile, err := s.repo.Update(ctx, repo.HistoryRecord{ID: file.ID, Content: file.Content, Version: file.Version})
	if err != nil {
		return File{}, err
	}
	updatedFile := s.fromRepoItem(dbFile)
	s.Publish(pubsub.UpdatedEvent, updatedFile)
	return updatedFile, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	file, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, file)
	return nil
}

func (s *service) DeleteSessionFiles(ctx context.Context, sessionID string) error {
	files, err := s.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := s.Delete(ctx, file.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) fromRepoItem(item repo.HistoryRecord) File {
	return File{
		ID:        item.ID,
		SessionID: item.SessionID,
		Path:      item.Path,
		Content:   item.Content,
		Version:   item.Version,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
