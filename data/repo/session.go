package repo

import (
	"context"
	"errors"

	datadb "ferryman-agent/data/db"
	"gorm.io/gorm"
)

type sessionRepo struct {
	db *gorm.DB
}

func NewSessionRepo(db *gorm.DB) SessionRepo {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, params CreateSessionParams) (Session, error) {
	item := datadb.Session{
		ID:               params.ID,
		ParentSessionID:  params.ParentSessionID,
		Title:            params.Title,
		MessageCount:     params.MessageCount,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		Cost:             params.Cost,
	}
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		return Session{}, err
	}
	return fromDBSession(item), nil
}

func (r *sessionRepo) Get(ctx context.Context, id string) (Session, error) {
	var item datadb.Session
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Session{}, ErrRepoNotFound
		}
		return Session{}, err
	}
	return fromDBSession(item), nil
}

func (r *sessionRepo) ListRoot(ctx context.Context) ([]Session, error) {
	var rows []datadb.Session
	if err := r.db.WithContext(ctx).Where("parent_session_id = ?", "").Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Session, len(rows))
	for i, row := range rows {
		items[i] = fromDBSession(row)
	}
	return items, nil
}

func (r *sessionRepo) Update(ctx context.Context, params UpdateSessionParams) (Session, error) {
	var item datadb.Session
	if err := r.db.WithContext(ctx).First(&item, "id = ?", params.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Session{}, ErrRepoNotFound
		}
		return Session{}, err
	}
	item.Title = params.Title
	item.PromptTokens = params.PromptTokens
	item.CompletionTokens = params.CompletionTokens
	item.SummaryMessageID = params.SummaryMessageID
	item.Cost = params.Cost
	if err := r.db.WithContext(ctx).Save(&item).Error; err != nil {
		return Session{}, err
	}
	return fromDBSession(item), nil
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&datadb.Session{}, "id = ?", id)
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
	return r.db.WithContext(ctx).Model(&datadb.Session{}).Where("id = ?", id).Update("message_count", next).Error
}

func fromDBSession(item datadb.Session) Session {
	return Session(item)
}
