package repo

import (
	"context"
	"errors"

	datadb "ferryman-agent/data/db"
	"gorm.io/gorm"
)

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) MessageRepo {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, params CreateMessageParams) (Message, error) {
	item := datadb.Message{
		ID:        params.ID,
		SessionID: params.SessionID,
		Role:      params.Role,
		Parts:     params.Parts,
		Model:     params.Model,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		return tx.Model(&datadb.Session{}).Where("id = ?", item.SessionID).
			Update("message_count", gorm.Expr("message_count + ?", 1)).Error
	})
	if err != nil {
		return Message{}, err
	}
	return fromDBMessage(item), nil
}

func (r *messageRepo) Update(ctx context.Context, params UpdateMessageParams) error {
	var item datadb.Message
	if err := r.db.WithContext(ctx).First(&item, "id = ?", params.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRepoNotFound
		}
		return err
	}
	item.Parts = params.Parts
	item.FinishedAt = params.FinishedAt
	return r.db.WithContext(ctx).Save(&item).Error
}

func (r *messageRepo) Get(ctx context.Context, id string) (Message, error) {
	var item datadb.Message
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Message{}, ErrRepoNotFound
		}
		return Message{}, err
	}
	return fromDBMessage(item), nil
}

func (r *messageRepo) ListBySession(ctx context.Context, sessionID string) ([]Message, error) {
	var rows []datadb.Message
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Message, len(rows))
	for i, row := range rows {
		items[i] = fromDBMessage(row)
	}
	return items, nil
}

func (r *messageRepo) Delete(ctx context.Context, id string) error {
	var item datadb.Message
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRepoNotFound
		}
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&datadb.Message{}, "id = ?", id).Error; err != nil {
			return err
		}
		return decrementMessageCount(tx, item.SessionID)
	})
}

func (r *messageRepo) DeleteBySession(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var deleted int64
		if err := tx.Model(&datadb.Message{}).Where("session_id = ?", sessionID).Count(&deleted).Error; err != nil {
			return err
		}
		if err := tx.Delete(&datadb.Message{}, "session_id = ?", sessionID).Error; err != nil {
			return err
		}
		if deleted == 0 {
			return nil
		}
		return tx.Model(&datadb.Session{}).Where("id = ?", sessionID).
			Update("message_count", gorm.Expr("CASE WHEN message_count - ? < 0 THEN 0 ELSE message_count - ? END", deleted, deleted)).Error
	})
}

func decrementMessageCount(tx *gorm.DB, sessionID string) error {
	return tx.Model(&datadb.Session{}).Where("id = ?", sessionID).
		Update("message_count", gorm.Expr("CASE WHEN message_count - 1 < 0 THEN 0 ELSE message_count - 1 END")).Error
}

func fromDBMessage(item datadb.Message) Message {
	return Message(item)
}
