package repo

import (
	"context"
	"errors"

	datadb "ferryman-agent/data/db"
	"gorm.io/gorm"
)

type historyRepo struct {
	db *gorm.DB
}

func NewHistoryRepo(db *gorm.DB) HistoryRepo {
	return &historyRepo{db: db}
}

func (r *historyRepo) Create(ctx context.Context, params CreateFileParams) (File, error) {
	item := datadb.File{
		ID:        params.ID,
		SessionID: params.SessionID,
		Path:      params.Path,
		Content:   params.Content,
		Version:   params.Version,
	}
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		return File{}, err
	}
	return fromDBFile(item), nil
}

func (r *historyRepo) Get(ctx context.Context, id string) (File, error) {
	var item datadb.File
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return File{}, ErrRepoNotFound
		}
		return File{}, err
	}
	return fromDBFile(item), nil
}

func (r *historyRepo) GetLatestByPathAndSession(ctx context.Context, path, sessionID string) (File, error) {
	var item datadb.File
	if err := r.db.WithContext(ctx).
		Where("path = ? AND session_id = ?", path, sessionID).
		Order("created_at desc, id desc").
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return File{}, ErrRepoNotFound
		}
		return File{}, err
	}
	return fromDBFile(item), nil
}

func (r *historyRepo) ListByPath(ctx context.Context, path string) ([]File, error) {
	var rows []datadb.File
	if err := r.db.WithContext(ctx).Where("path = ?", path).Order("created_at desc, id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return fromDBFiles(rows), nil
}

func (r *historyRepo) ListBySession(ctx context.Context, sessionID string) ([]File, error) {
	var rows []datadb.File
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return fromDBFiles(rows), nil
}

func (r *historyRepo) ListLatestBySession(ctx context.Context, sessionID string) ([]File, error) {
	var rows []datadb.File
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("path asc, created_at desc, id desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	items := make([]File, 0)
	for _, row := range rows {
		if seen[row.Path] {
			continue
		}
		seen[row.Path] = true
		items = append(items, fromDBFile(row))
	}
	return items, nil
}

func (r *historyRepo) Update(ctx context.Context, params UpdateFileParams) (File, error) {
	var item datadb.File
	if err := r.db.WithContext(ctx).First(&item, "id = ?", params.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return File{}, ErrRepoNotFound
		}
		return File{}, err
	}
	item.Content = params.Content
	item.Version = params.Version
	if err := r.db.WithContext(ctx).Save(&item).Error; err != nil {
		return File{}, err
	}
	return fromDBFile(item), nil
}

func (r *historyRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&datadb.File{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *historyRepo) DeleteBySession(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Delete(&datadb.File{}, "session_id = ?", sessionID).Error
}

func fromDBFiles(rows []datadb.File) []File {
	items := make([]File, len(rows))
	for i, row := range rows {
		items[i] = fromDBFile(row)
	}
	return items
}

func fromDBFile(item datadb.File) File {
	return File(item)
}
