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

// ClaimForGreeting atomically marks a due recipient as being greeted by setting
// last_greeting_at to now. The update only succeeds while the stored value still
// equals previous, so concurrent scans cannot both claim the same recipient.
// Callers pass recipients already filtered as due by DueForGreeting. It returns
// true when this caller won the claim.
func (r *RecipientRepository) ClaimForGreeting(ctx context.Context, id uint, previous *time.Time, now time.Time) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.CareRecipient{}).
		Where("id = ? AND status = ? AND care_start_at <= ?", id, constants.RecipientStatusActive, now)
	if previous == nil {
		query = query.Where("last_greeting_at IS NULL")
	} else {
		query = query.Where("last_greeting_at = ?", *previous)
	}
	result := query.Update("last_greeting_at", now)
	if result.Error != nil {
		return false, fmt.Errorf("claim greeting for recipient %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

// ReleaseGreetingClaim restores the previous greeting time after a failed send,
// but only when the stored value is still this caller's claim timestamp. It
// returns true when the rollback was applied.
func (r *RecipientRepository) ReleaseGreetingClaim(ctx context.Context, id uint, claimedAt time.Time, previous *time.Time) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.CareRecipient{}).
		Where("id = ? AND last_greeting_at = ?", id, claimedAt)
	var result *gorm.DB
	if previous == nil {
		result = query.Update("last_greeting_at", gorm.Expr("NULL"))
	} else {
		result = query.Update("last_greeting_at", *previous)
	}
	if result.Error != nil {
		return false, fmt.Errorf("release greeting claim for recipient %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
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
