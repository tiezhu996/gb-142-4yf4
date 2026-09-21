package repository

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/gorm"
)

type SMSLogRepository struct{ db *gorm.DB }

func NewSMSLogRepository(db *gorm.DB) *SMSLogRepository { return &SMSLogRepository{db: db} }
func (r *SMSLogRepository) Create(ctx context.Context, item *model.SMSLog) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return fmt.Errorf("create sms log: %w", err)
	}
	return nil
}
func (r *SMSLogRepository) List(ctx context.Context, page, pageSize int, recipientID *uint) ([]model.SMSLog, int64, error) {
	var items []model.SMSLog
	var total int64
	q := r.db.WithContext(ctx).Model(&model.SMSLog{})
	if recipientID != nil {
		q = q.Where("care_recipient_id = ?", *recipientID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sms logs: %w", err)
	}
	if err := q.Order("sent_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list sms logs: %w", err)
	}
	return items, total, nil
}
