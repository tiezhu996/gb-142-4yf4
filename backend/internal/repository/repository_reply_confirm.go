package repository

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/gorm"
)

type ReplyConfirmRepository struct{ db *gorm.DB }

func NewReplyConfirmRepository(db *gorm.DB) *ReplyConfirmRepository {
	return &ReplyConfirmRepository{db: db}
}
func (r *ReplyConfirmRepository) Create(ctx context.Context, item *model.ReplyConfirm) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return fmt.Errorf("create reply confirm: %w", err)
	}
	return nil
}
