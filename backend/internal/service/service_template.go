package service

import (
	"context"
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
)

type TemplateService struct {
	repo *repository.TemplateRepository
}

func NewTemplateService(repo *repository.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}
func (s *TemplateService) Create(ctx context.Context, req dto.CreateTemplateRequest) (*model.SMSTemplate, error) {
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	item := &model.SMSTemplate{Name: req.Name, Category: req.Category, Content: req.Content, Active: active}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *TemplateService) List(ctx context.Context, category string) ([]model.SMSTemplate, error) {
	return s.repo.List(ctx, category)
}
func (s *TemplateService) Get(ctx context.Context, id uint) (*model.SMSTemplate, error) {
	return s.repo.Get(ctx, id)
}
func (s *TemplateService) Update(ctx context.Context, id uint, req dto.UpdateTemplateRequest) (*model.SMSTemplate, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	item.Name = req.Name
	item.Category = req.Category
	item.Content = req.Content
	item.Active = req.Active
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *TemplateService) Delete(ctx context.Context, id uint) error { return s.repo.Delete(ctx, id) }
