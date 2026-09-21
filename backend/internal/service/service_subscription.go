package service

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"log/slog"
)

type SubscriptionService struct {
	recipients *repository.RecipientRepository
	repo       *repository.SubscriptionRepository
	logger     *slog.Logger
}

func NewSubscriptionService(recipients *repository.RecipientRepository, repo *repository.SubscriptionRepository, logger *slog.Logger) *SubscriptionService {
	return &SubscriptionService{recipients: recipients, repo: repo, logger: logger}
}
func (s *SubscriptionService) Create(ctx context.Context, recipientID uint, req dto.CreateSubscriptionRequest) (*model.FamilySubscription, error) {
	if _, err := s.recipients.Get(ctx, recipientID); err != nil {
		return nil, fmt.Errorf("verify recipient: %w", err)
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	item := &model.FamilySubscription{CareRecipientID: recipientID, FamilyName: req.FamilyName, FamilyPhone: req.FamilyPhone, Active: active}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *SubscriptionService) List(ctx context.Context, recipientID uint) ([]model.FamilySubscription, error) {
	return s.repo.ListByRecipient(ctx, recipientID)
}
