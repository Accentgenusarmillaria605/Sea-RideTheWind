package model

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type LikeConsumeInboxModel interface {
	FindByMsgID(ctx context.Context, msgID string) (*LikeConsumeInbox, error)
	CreateTx(ctx context.Context, tx *gorm.DB, data *LikeConsumeInbox) error
	MarkDoneTx(ctx context.Context, tx *gorm.DB, msgID string) error
}

type defaultLikeConsumeInboxModel struct {
	db *gorm.DB
}

func NewLikeConsumeInboxModel(db *gorm.DB) LikeConsumeInboxModel {
	return &defaultLikeConsumeInboxModel{db: db}
}

func (m *defaultLikeConsumeInboxModel) FindByMsgID(ctx context.Context, msgID string) (*LikeConsumeInbox, error) {
	var res LikeConsumeInbox
	err := m.db.WithContext(ctx).Where("msg_id = ?", msgID).First(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (m *defaultLikeConsumeInboxModel) CreateTx(ctx context.Context, tx *gorm.DB, data *LikeConsumeInbox) error {
	return tx.WithContext(ctx).Create(data).Error
}

func (m *defaultLikeConsumeInboxModel) MarkDoneTx(ctx context.Context, tx *gorm.DB, msgID string) error {
	return tx.WithContext(ctx).
		Model(&LikeConsumeInbox{}).
		Where("msg_id = ?", msgID).
		Update("status", 1).Error
}
