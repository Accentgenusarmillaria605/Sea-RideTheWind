package model

import (
	"context"

	"gorm.io/gorm"
)

type LikeOutboxEventModel interface {
	CreateTx(ctx context.Context, tx *gorm.DB, data *LikeOutboxEvent) error
	FetchPending(ctx context.Context, limit int) ([]*LikeOutboxEvent, error)
	MarkSent(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string) error
}

type defaultLikeOutboxEventModel struct {
	db *gorm.DB
}

func NewLikeOutboxEventModel(db *gorm.DB) LikeOutboxEventModel {
	return &defaultLikeOutboxEventModel{db: db}
}

func (m *defaultLikeOutboxEventModel) CreateTx(ctx context.Context, tx *gorm.DB, data *LikeOutboxEvent) error {
	return tx.WithContext(ctx).Create(data).Error
}

func (m *defaultLikeOutboxEventModel) FetchPending(ctx context.Context, limit int) ([]*LikeOutboxEvent, error) {
	var res []*LikeOutboxEvent
	err := m.db.WithContext(ctx).
		Where("status IN (0,2) AND retry_count < 3").
		Order("created_at asc").
		Limit(limit).
		Find(&res).Error
	return res, err
}

func (m *defaultLikeOutboxEventModel) MarkSent(ctx context.Context, eventID string) error {
	return m.db.WithContext(ctx).
		Model(&LikeOutboxEvent{}).
		Where("event_id = ?", eventID).
		Update("status", 1).Error
}

func (m *defaultLikeOutboxEventModel) MarkFailed(ctx context.Context, eventID string) error {
	return m.db.WithContext(ctx).
		Model(&LikeOutboxEvent{}).
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":      2,
			"retry_count": gorm.Expr("retry_count + 1"),
		}).Error
}
