package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/gorm"
)

type RecipientRepository struct{ db *gorm.DB }

func NewRecipientRepository(db *gorm.DB) *RecipientRepository { return &RecipientRepository{db: db} }
func (r *RecipientRepository) Create(ctx context.Context, recipient *model.CareRecipient) error {
	if err := r.db.WithContext(ctx).Create(recipient).Error; err != nil {
		return fmt.Errorf("create recipient: %w", err)
	}
	return nil
}
func (r *RecipientRepository) List(ctx context.Context, page, pageSize int) ([]model.CareRecipient, int64, error) {
	var items []model.CareRecipient
	var total int64
	db := r.db.WithContext(ctx).Model(&model.CareRecipient{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count recipients: %w", err)
	}
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list recipients: %w", err)
	}
	return items, total, nil
}
func (r *RecipientRepository) Get(ctx context.Context, id uint) (*model.CareRecipient, error) {
	var item model.CareRecipient
	err := r.db.WithContext(ctx).Preload("FamilySubscriptions").First(&item, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get recipient: %w", err)
	}
	return &item, nil
}
func (r *RecipientRepository) Update(ctx context.Context, recipient *model.CareRecipient) error {
	result := r.db.WithContext(ctx).Save(recipient)
	if result.Error != nil {
		return fmt.Errorf("update recipient: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *RecipientRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.CareRecipient{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete recipient: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *RecipientRepository) DueForGreeting(ctx context.Context, now time.Time) ([]model.CareRecipient, error) {
	var active []model.CareRecipient
	if err := r.db.WithContext(ctx).Where("status = ? AND care_start_at <= ?", constants.RecipientStatusActive, now).Find(&active).Error; err != nil {
		return nil, fmt.Errorf("list active recipients: %w", err)
	}
	due := make([]model.CareRecipient, 0)
	for _, item := range active {
		interval := frequencyInterval(item.CareFrequency)
		if item.LastGreetingAt == nil || !item.LastGreetingAt.Add(interval).After(now) {
			due = append(due, item)
		}
	}
	return due, nil
}
func (r *RecipientRepository) Overdue(ctx context.Context, cutoff time.Time) ([]model.CareRecipient, error) {
	var items []model.CareRecipient
	err := r.db.WithContext(ctx).Where("status = ? AND care_start_at <= ? AND (last_confirmed_at IS NULL OR last_confirmed_at < ?)", constants.RecipientStatusActive, cutoff, cutoff).Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("list overdue recipients: %w", err)
	}
	return items, nil
}
func frequencyInterval(frequency string) time.Duration {
	switch frequency {
	case constants.FrequencyWeekly:
		return 7 * 24 * time.Hour
	case constants.FrequencyMonthly:
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}
