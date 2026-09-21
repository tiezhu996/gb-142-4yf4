package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"log/slog"
	"time"
)

type RecipientService struct {
	repo     *repository.RecipientRepository
	confirms *repository.ReplyConfirmRepository
	logger   *slog.Logger
}

func NewRecipientService(repo *repository.RecipientRepository, confirms *repository.ReplyConfirmRepository, logger *slog.Logger) *RecipientService {
	return &RecipientService{repo: repo, confirms: confirms, logger: logger}
}
func (s *RecipientService) Create(ctx context.Context, req dto.CreateRecipientRequest) (*model.CareRecipient, error) {
	start, err := time.Parse(time.RFC3339, req.CareStartAt)
	if err != nil {
		return nil, fmt.Errorf("parse care start time: %w", err)
	}
	status := req.Status
	if status == "" {
		status = constants.RecipientStatusActive
	}
	item := &model.CareRecipient{Name: req.Name, Phone: req.Phone, CareFrequency: req.CareFrequency, CareStartAt: start, Status: status}
	if err = s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *RecipientService) List(ctx context.Context, page, pageSize int) ([]model.CareRecipient, int64, error) {
	return s.repo.List(ctx, page, pageSize)
}
func (s *RecipientService) Get(ctx context.Context, id uint) (*model.CareRecipient, error) {
	return s.repo.Get(ctx, id)
}
func (s *RecipientService) Update(ctx context.Context, id uint, req dto.UpdateRecipientRequest) (*model.CareRecipient, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	start, err := time.Parse(time.RFC3339, req.CareStartAt)
	if err != nil {
		return nil, fmt.Errorf("parse care start time: %w", err)
	}
	item.Name = req.Name
	item.Phone = req.Phone
	item.CareFrequency = req.CareFrequency
	item.CareStartAt = start
	item.Status = req.Status
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *RecipientService) Delete(ctx context.Context, id uint) error { return s.repo.Delete(ctx, id) }
func (s *RecipientService) Confirm(ctx context.Context, id uint, req dto.ConfirmRecipientRequest) (*model.CareRecipient, error) {
	item, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item.LastConfirmedAt = &now
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	channel := req.Channel
	if channel == "" {
		channel = "manual"
	}
	if err := s.confirms.Create(ctx, &model.ReplyConfirm{CareRecipientID: id, Channel: channel, Note: req.Note, ConfirmedAt: now}); err != nil {
		return nil, err
	}
	s.logger.Info("recipient confirmed", "recipient_id", id, "channel", channel)
	return item, nil
}
func (s *RecipientService) MarkGreeted(ctx context.Context, item *model.CareRecipient, now time.Time) error {
	if item == nil {
		return errors.New("recipient is nil")
	}
	item.LastGreetingAt = &now
	return s.repo.Update(ctx, item)
}
