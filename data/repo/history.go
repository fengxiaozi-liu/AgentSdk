package repo

import (
	"context"
	"errors"

	datadb "ferryman-agent/data/db"
	"gorm.io/gorm"
)

type HistoryRecord struct {
	ID        string `gorm:"column:id;primaryKey"`
	SessionID string `gorm:"column:session_id;index"`
	Path      string `gorm:"column:path;index"`
	Content   string `gorm:"column:content"`
	Version   string `gorm:"column:version"`
	CreatedAt int64  `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt int64  `gorm:"column:updated_at;autoUpdateTime:milli"`
}

func (HistoryRecord) TableName() string {
	return "history"
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
	Create(context.Context, CreateFileParams) (HistoryRecord, error)
	Get(context.Context, string) (HistoryRecord, error)
	GetLatestByPathAndSession(context.Context, string, string) (HistoryRecord, error)
	ListByPath(context.Context, string) ([]HistoryRecord, error)
	ListBySession(context.Context, string) ([]HistoryRecord, error)
	ListLatestBySession(context.Context, string) ([]HistoryRecord, error)
	Update(context.Context, UpdateFileParams) (HistoryRecord, error)
	Delete(context.Context, string) error
	DeleteBySession(context.Context, string) error
}

type historyRepo struct {
	client *datadb.DbClient
}

func NewHistoryRepo(client *datadb.DbClient) HistoryRepo {
	return &historyRepo{client: client}
}

func (r *historyRepo) Create(ctx context.Context, params CreateFileParams) (HistoryRecord, error) {
	item := HistoryRecord{
		ID:        params.ID,
		SessionID: params.SessionID,
		Path:      params.Path,
		Content:   params.Content,
		Version:   params.Version,
	}
	if err := r.client.DB.WithContext(ctx).Create(&item).Error; err != nil {
		return HistoryRecord{}, err
	}
	return item, nil
}

func (r *historyRepo) Get(ctx context.Context, id string) (HistoryRecord, error) {
	var item HistoryRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HistoryRecord{}, ErrRepoNotFound
		}
		return HistoryRecord{}, err
	}
	return item, nil
}

func (r *historyRepo) GetLatestByPathAndSession(ctx context.Context, path, sessionID string) (HistoryRecord, error) {
	var item HistoryRecord
	if err := r.client.DB.WithContext(ctx).
		Where("path = ? AND session_id = ?", path, sessionID).
		Order("created_at desc, id desc").
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HistoryRecord{}, ErrRepoNotFound
		}
		return HistoryRecord{}, err
	}
	return item, nil
}

func (r *historyRepo) ListByPath(ctx context.Context, path string) ([]HistoryRecord, error) {
	var rows []HistoryRecord
	if err := r.client.DB.WithContext(ctx).Where("path = ?", path).Order("created_at desc, id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *historyRepo) ListBySession(ctx context.Context, sessionID string) ([]HistoryRecord, error) {
	var rows []HistoryRecord
	if err := r.client.DB.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *historyRepo) ListLatestBySession(ctx context.Context, sessionID string) ([]HistoryRecord, error) {
	var rows []HistoryRecord
	if err := r.client.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("path asc, created_at desc, id desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	items := make([]HistoryRecord, 0)
	for _, row := range rows {
		if seen[row.Path] {
			continue
		}
		seen[row.Path] = true
		items = append(items, row)
	}
	return items, nil
}

func (r *historyRepo) Update(ctx context.Context, params UpdateFileParams) (HistoryRecord, error) {
	var item HistoryRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", params.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return HistoryRecord{}, ErrRepoNotFound
		}
		return HistoryRecord{}, err
	}
	item.Content = params.Content
	item.Version = params.Version
	if err := r.client.DB.WithContext(ctx).Save(&item).Error; err != nil {
		return HistoryRecord{}, err
	}
	return item, nil
}

func (r *historyRepo) Delete(ctx context.Context, id string) error {
	result := r.client.DB.WithContext(ctx).Delete(&HistoryRecord{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *historyRepo) DeleteBySession(ctx context.Context, sessionID string) error {
	return r.client.DB.WithContext(ctx).Delete(&HistoryRecord{}, "session_id = ?", sessionID).Error
}
