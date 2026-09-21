package repository

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/gorm"
)

type SubscriptionRepository struct{ db *gorm.DB }

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}
func (r *SubscriptionRepository) Create(ctx context.Context, item *model.FamilySubscription) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	return nil
}
func (r *SubscriptionRepository) ListByRecipient(ctx context.Context, recipientID uint) ([]model.FamilySubscription, error) {
	var items []model.FamilySubscription
	if err := r.db.WithContext(ctx).Where("care_recipient_id = ?", recipientID).Order("id desc").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	return items, nil
}
func (r *SubscriptionRepository) ListActiveByRecipient(ctx context.Context, recipientID uint) ([]model.FamilySubscription, error) {
	var items []model.FamilySubscription
	if err := r.db.WithContext(ctx).Where("care_recipient_id = ? AND active = ?", recipientID, true).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	return items, nil
}
