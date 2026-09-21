package service

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"log/slog"
	"strings"
	"time"
)

type NotificationService struct {
	recipients *repository.RecipientRepository
	templates  *repository.TemplateRepository
	sms        *SMSService
	logger     *slog.Logger
}

func NewNotificationService(recipients *repository.RecipientRepository, templates *repository.TemplateRepository, sms *SMSService, logger *slog.Logger) *NotificationService {
	return &NotificationService{recipients: recipients, templates: templates, sms: sms, logger: logger}
}
func (s *NotificationService) SendDueGreetings(ctx context.Context) (int, error) {
	items, err := s.recipients.DueForGreeting(ctx, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	sent := 0
	for i := range items {
		if err := s.SendGreeting(ctx, &items[i], ""); err != nil {
			return sent, fmt.Errorf("send greeting to recipient %d: %w", items[i].ID, err)
		}
		sent++
	}
	return sent, nil
}
func (s *NotificationService) SendGreeting(ctx context.Context, recipient *model.CareRecipient, category string) error {
	template, err := s.templates.RandomActive(ctx, category)
	if err == repository.ErrNotFound {
		template = &model.SMSTemplate{Content: "{{name}}，您好，愿您今天平安顺心。如方便，请回复“已阅”报个平安。"}
	} else if err != nil {
		return err
	}
	content := strings.ReplaceAll(template.Content, "{{name}}", recipient.Name)
	recipientID := recipient.ID
	if err := s.sms.Send(ctx, SMSMessage{CareRecipientID: &recipientID, RecipientPhone: recipient.Phone, Content: content, Kind: constants.SMSKindGreeting}); err != nil {
		return err
	}
	now := time.Now().UTC()
	recipient.LastGreetingAt = &now
	if err := s.recipients.Update(ctx, recipient); err != nil {
		return err
	}
	s.logger.Info("greeting dispatched", "recipient_id", recipient.ID)
	return nil
}
