package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/gorm"
)

type TemplateRepository struct{ db *gorm.DB }

func NewTemplateRepository(db *gorm.DB) *TemplateRepository { return &TemplateRepository{db: db} }
func (r *TemplateRepository) Create(ctx context.Context, item *model.SMSTemplate) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	return nil
}
func (r *TemplateRepository) List(ctx context.Context, category string) ([]model.SMSTemplate, error) {
	var items []model.SMSTemplate
	q := r.db.WithContext(ctx).Order("id desc")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	return items, nil
}
func (r *TemplateRepository) Get(ctx context.Context, id uint) (*model.SMSTemplate, error) {
	var item model.SMSTemplate
	err := r.db.WithContext(ctx).First(&item, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	return &item, nil
}
func (r *TemplateRepository) Update(ctx context.Context, item *model.SMSTemplate) error {
	result := r.db.WithContext(ctx).Save(item)
	if result.Error != nil {
		return fmt.Errorf("update template: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *TemplateRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.SMSTemplate{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete template: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *TemplateRepository) RandomActive(ctx context.Context, category string) (*model.SMSTemplate, error) {
	var item model.SMSTemplate
	q := r.db.WithContext(ctx).Where("active = ?", true)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	err := q.Order("RAND()").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select random template: %w", err)
	}
	return &item, nil
}
