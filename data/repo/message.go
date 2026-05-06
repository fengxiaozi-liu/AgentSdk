package repo

import (
	"context"
	"errors"
	"time"

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
	CreatedAt  int64  `gorm:"column:created_at;autoCreateTime:nano"`
	UpdatedAt  int64  `gorm:"column:updated_at;autoUpdateTime:nano"`
}

func (MessageRecord) TableName() string {
	return "messages"
}

func (r *MessageRecord) BeforeCreate(*gorm.DB) error {
	now := time.Now().UnixNano()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = now
	}
	return nil
}

func (r *MessageRecord) BeforeUpdate(*gorm.DB) error {
	r.UpdatedAt = time.Now().UnixNano()
	return nil
}

type MessageRepo interface {
	Create(context.Context, MessageRecord) (MessageRecord, error)
	Update(context.Context, MessageRecord) error
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

func (r *messageRepo) Create(ctx context.Context, item MessageRecord) (MessageRecord, error) {
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

func (r *messageRepo) Update(ctx context.Context, record MessageRecord) error {
	var item MessageRecord
	if err := r.client.DB.WithContext(ctx).First(&item, "id = ?", record.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRepoNotFound
		}
		return err
	}
	item.Parts = record.Parts
	item.FinishedAt = record.FinishedAt
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
