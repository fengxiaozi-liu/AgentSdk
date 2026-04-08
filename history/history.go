package history

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	agentdb "ferryman-agent/infra/db"
	"ferryman-agent/pubsub"
	"github.com/google/uuid"
)

const InitialVersion = "initial"

type File struct {
	ID        string
	SessionID string
	Path      string
	Content   string
	Version   string
	CreatedAt int64
	UpdatedAt int64
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
	db *sql.DB
	q  *agentdb.Queries
}

func NewService(q *agentdb.Queries, db *sql.DB) Service {
	return &service{Broker: pubsub.NewBroker[File](), q: q, db: db}
}

func (s *service) Create(ctx context.Context, sessionID, path, content string) (File, error) {
	return s.createWithVersion(ctx, sessionID, path, content, InitialVersion)
}

func (s *service) CreateVersion(ctx context.Context, sessionID, path, content string) (File, error) {
	files, err := s.q.ListFilesByPath(ctx, path)
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
	const maxRetries = 3
	var file File
	var err error

	for attempt := range maxRetries {
		tx, txErr := s.db.Begin()
		if txErr != nil {
			return File{}, fmt.Errorf("failed to begin transaction: %w", txErr)
		}
		qtx := s.q.WithTx(tx)
		dbFile, txErr := qtx.CreateFile(ctx, agentdb.CreateFileParams{
			ID: uuid.New().String(), SessionID: sessionID, Path: path, Content: content, Version: version,
		})
		if txErr != nil {
			_ = tx.Rollback()
			if strings.Contains(txErr.Error(), "UNIQUE constraint failed") && attempt < maxRetries-1 {
				if strings.HasPrefix(version, "v") {
					versionNum, parseErr := strconv.Atoi(version[1:])
					if parseErr == nil {
						version = fmt.Sprintf("v%d", versionNum+1)
						continue
					}
				}
				version = fmt.Sprintf("v%d", time.Now().Unix())
				continue
			}
			return File{}, txErr
		}
		if txErr = tx.Commit(); txErr != nil {
			return File{}, fmt.Errorf("failed to commit transaction: %w", txErr)
		}
		file = s.fromDBItem(dbFile)
		s.Publish(pubsub.CreatedEvent, file)
		return file, nil
	}
	return file, err
}

func (s *service) Get(ctx context.Context, id string) (File, error) {
	dbFile, err := s.q.GetFile(ctx, id)
	if err != nil {
		return File{}, err
	}
	return s.fromDBItem(dbFile), nil
}

func (s *service) GetByPathAndSession(ctx context.Context, path, sessionID string) (File, error) {
	dbFile, err := s.q.GetFileByPathAndSession(ctx, agentdb.GetFileByPathAndSessionParams{Path: path, SessionID: sessionID})
	if err != nil {
		return File{}, err
	}
	return s.fromDBItem(dbFile), nil
}

func (s *service) ListBySession(ctx context.Context, sessionID string) ([]File, error) {
	dbFiles, err := s.q.ListFilesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromDBItem(dbFile)
	}
	return files, nil
}

func (s *service) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]File, error) {
	dbFiles, err := s.q.ListLatestSessionFiles(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromDBItem(dbFile)
	}
	return files, nil
}

func (s *service) Update(ctx context.Context, file File) (File, error) {
	dbFile, err := s.q.UpdateFile(ctx, agentdb.UpdateFileParams{ID: file.ID, Content: file.Content, Version: file.Version})
	if err != nil {
		return File{}, err
	}
	updatedFile := s.fromDBItem(dbFile)
	s.Publish(pubsub.UpdatedEvent, updatedFile)
	return updatedFile, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	file, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.q.DeleteFile(ctx, id); err != nil {
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

func (s *service) fromDBItem(item agentdb.File) File {
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
