package repo

import (
	"context"
	"errors"
	"time"

	datadb "ferryman-agent/internal/data/db"
	"gorm.io/gorm"
)

type SessionRecord struct {
	ID               string  `gorm:"column:id;primaryKey"`
	ParentSessionID  string  `gorm:"column:parent_session_id;index"`
	Title            string  `gorm:"column:title"`
	MessageCount     int64   `gorm:"column:message_count"`
	PromptTokens     int64   `gorm:"column:prompt_tokens"`
	CompletionTokens int64   `gorm:"column:completion_tokens"`
	SummaryMessageID string  `gorm:"column:summary_message_id"`
	Cost             float64 `gorm:"column:cost"`
	CreatedAt        int64   `gorm:"column:created_at;autoCreateTime:nano"`
	UpdatedAt        int64   `gorm:"column:updated_at;autoUpdateTime:nano"`
}

func (SessionRecord) TableName() string {
	return "sessions"
}

func (r *SessionRecord) BeforeCreate(*gorm.DB) error {
	now := time.Now().UnixNano()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = now
	}
	return nil
}

func (r *SessionRecord) BeforeUpdate(*gorm.DB) error {
	r.UpdatedAt = time.Now().UnixNano()
	return nil
}

type SessionRepo interface {
	Create(context.Context, SessionRecord) (SessionRecord, error)
	Get(context.Context, string) (SessionRecord, error)
	ListRoot(context.Context) ([]SessionRecord, error)
	Update(context.Context, SessionRecord) (SessionRecord, error)
	Delete(context.Context, string) error
	IncrementMessageCount(context.Context, string, int64) error
}

type sessionRepo struct {
	client *datadb.DbClient
}

func NewSessionRepo(client *datadb.DbClient) SessionRepo {
	return &sessionRepo{client: client}
}

func (r *sessionRepo) Create(ctx context.Context, item SessionRecord) (SessionRecord, error) {
	if err := r.client.DB.WithContext(ctx).Create(&item).Error; err != nil {
		return SessionRecord{}, err
	}
	return item, nil
}

func (r *sessionRepo) Get(ctx context.Context, id string) (SessionRecord, error) {
	var item SessionRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SessionRecord{}, ErrRepoNotFound
		}
		return SessionRecord{}, err
	}
	return item, nil
}

func (r *sessionRepo) ListRoot(ctx context.Context) ([]SessionRecord, error) {
	var rows []SessionRecord
	if err := r.client.DB.WithContext(ctx).Where("parent_session_id = ?", "").Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *sessionRepo) Update(ctx context.Context, record SessionRecord) (SessionRecord, error) {
	var item SessionRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", record.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SessionRecord{}, ErrRepoNotFound
		}
		return SessionRecord{}, err
	}
	item.Title = record.Title
	item.PromptTokens = record.PromptTokens
	item.CompletionTokens = record.CompletionTokens
	item.SummaryMessageID = record.SummaryMessageID
	item.Cost = record.Cost
	if err := r.client.DB.WithContext(ctx).Save(&item).Error; err != nil {
		return SessionRecord{}, err
	}
	return item, nil
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	result := r.client.DB.WithContext(ctx).Delete(&SessionRecord{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *sessionRepo) IncrementMessageCount(ctx context.Context, id string, delta int64) error {
	session, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	next := session.MessageCount + delta
	if next < 0 {
		next = 0
	}
	return r.client.DB.WithContext(ctx).Model(&SessionRecord{}).Where("id = ?", id).Update("message_count", next).Error
}
