package repo

import (
	"context"
	"errors"

	datadb "ferryman-agent/data/db"
	"gorm.io/gorm"
)

type MessageRecord struct {
	ID         string `gorm:"column:id;primaryKey"`
	SessionID  string `gorm:"column:session_id;index"`
	Role       string `gorm:"column:role"`
	Parts      string `gorm:"column:parts"`
	Model      string `gorm:"column:model"`
	FinishedAt int64  `gorm:"column:finished_at"`
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt  int64  `gorm:"column:updated_at;autoUpdateTime:milli"`
}

func (MessageRecord) TableName() string {
	return "messages"
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
	Create(context.Context, CreateMessageParams) (MessageRecord, error)
	Update(context.Context, UpdateMessageParams) error
	Get(context.Context, string) (MessageRecord, error)
	ListBySession(context.Context, string) ([]MessageRecord, error)
	Delete(context.Context, string) error
	DeleteBySession(context.Context, string) error
}

type messageRepo struct {
	client *datadb.DbClient
}

func NewMessageRepo(client *datadb.DbClient) MessageRepo {
	return &messageRepo{client: client}
}

func (r *messageRepo) Create(ctx context.Context, params CreateMessageParams) (MessageRecord, error) {
	item := MessageRecord{
		ID:        params.ID,
		SessionID: params.SessionID,
		Role:      params.Role,
		Parts:     params.Parts,
		Model:     params.Model,
	}
	err := r.client.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		return tx.Model(&SessionRecord{}).Where("id = ?", item.SessionID).
			Update("message_count", gorm.Expr("message_count + ?", 1)).Error
	})
	if err != nil {
		return MessageRecord{}, err
	}
	return item, nil
}

func (r *messageRepo) Update(ctx context.Context, params UpdateMessageParams) error {
	var item MessageRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", params.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRepoNotFound
		}
		return err
	}
	item.Parts = params.Parts
	item.FinishedAt = params.FinishedAt
	return r.client.DB.WithContext(ctx).Save(&item).Error
}

func (r *messageRepo) Get(ctx context.Context, id string) (MessageRecord, error) {
	var item MessageRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MessageRecord{}, ErrRepoNotFound
		}
		return MessageRecord{}, err
	}
	return item, nil
}

func (r *messageRepo) ListBySession(ctx context.Context, sessionID string) ([]MessageRecord, error) {
	var rows []MessageRecord
	if err := r.client.DB.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *messageRepo) Delete(ctx context.Context, id string) error {
	var item MessageRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRepoNotFound
		}
		return err
	}
	return r.client.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&MessageRecord{}, "id = ?", id).Error; err != nil {
			return err
		}
		return decrementMessageCount(tx, item.SessionID)
	})
}

func (r *messageRepo) DeleteBySession(ctx context.Context, sessionID string) error {
	return r.client.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var deleted int64
		if err := tx.Model(&MessageRecord{}).Where("session_id = ?", sessionID).Count(&deleted).Error; err != nil {
			return err
		}
		if err := tx.Delete(&MessageRecord{}, "session_id = ?", sessionID).Error; err != nil {
			return err
		}
		if deleted == 0 {
			return nil
		}
		return tx.Model(&SessionRecord{}).Where("id = ?", sessionID).
			Update("message_count", gorm.Expr("CASE WHEN message_count - ? < 0 THEN 0 ELSE message_count - ? END", deleted, deleted)).Error
	})
}

func decrementMessageCount(tx *gorm.DB, sessionID string) error {
	return tx.Model(&SessionRecord{}).Where("id = ?", sessionID).
		Update("message_count", gorm.Expr("CASE WHEN message_count - 1 < 0 THEN 0 ELSE message_count - 1 END")).Error
}
