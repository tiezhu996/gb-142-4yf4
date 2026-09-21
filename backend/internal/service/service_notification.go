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

// GreetingBatchResult summarizes one greeting scan. Every due recipient is
// processed exactly once; failures are isolated to the individual recipient.
type GreetingBatchResult struct {
	Due    int `json:"due"`
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
}

type NotificationService struct {
	recipients *repository.RecipientRepository
	templates  *repository.TemplateRepository
	logs       *repository.SMSLogRepository
	sms        *SMSService
	logger     *slog.Logger
}

func NewNotificationService(recipients *repository.RecipientRepository, templates *repository.TemplateRepository, logs *repository.SMSLogRepository, sms *SMSService, logger *slog.Logger) *NotificationService {
	return &NotificationService{recipients: recipients, templates: templates, logs: logs, sms: sms, logger: logger}
}
func (s *NotificationService) SendDueGreetings(ctx context.Context) (GreetingBatchResult, error) {
	items, err := s.recipients.DueForGreeting(ctx, time.Now().UTC())
	if err != nil {
		return GreetingBatchResult{}, err
	}
	result := GreetingBatchResult{Due: len(items)}
	for i := range items {
		sent, err := s.SendGreeting(ctx, &items[i], "")
		if err != nil {
			result.Failed++
			s.logger.Error("greeting failed, continue with next recipient", "recipient_id", items[i].ID, "phone", items[i].Phone, "error", err)
			continue
		}
		if sent {
			result.Sent++
		}
	}
	s.logger.Info("greeting batch finished", "due", result.Due, "sent", result.Sent, "failed", result.Failed)
	return result, nil
}

// SendGreeting claims the recipient before sending, so concurrent scans leave
// exactly one success result and one success log per recipient. On failure the
// claim is released (last greeting time stays unchanged) and the attempt is
// recorded as a failed log, so the next scan retries the recipient.
func (s *NotificationService) SendGreeting(ctx context.Context, recipient *model.CareRecipient, category string) (bool, error) {
	now := time.Now().UTC()
	claimed, err := s.recipients.ClaimForGreeting(ctx, recipient.ID, recipient.CareFrequency, now)
	if err != nil {
		return false, err
	}
	if !claimed {
		return false, nil
	}
	if err := s.deliverGreeting(ctx, recipient, category, now); err != nil {
		if releaseErr := s.recipients.ReleaseGreetingClaim(ctx, recipient.ID, now, recipient.LastGreetingAt); releaseErr != nil {
			s.logger.Error("release greeting claim", "recipient_id", recipient.ID, "error", releaseErr)
		}
		return false, err
	}
	recipient.LastGreetingAt = &now
	s.logger.Info("greeting dispatched", "recipient_id", recipient.ID)
	return true, nil
}
func (s *NotificationService) deliverGreeting(ctx context.Context, recipient *model.CareRecipient, category string, now time.Time) error {
	template, err := s.templates.RandomActive(ctx, category)
	if err == repository.ErrNotFound {
		template = &model.SMSTemplate{Content: "{{name}}，您好，愿您今天平安顺心。如方便，请回复“已阅”报个平安。"}
	} else if err != nil {
		return fmt.Errorf("select greeting template: %w", err)
	}
	content := strings.ReplaceAll(template.Content, "{{name}}", recipient.Name)
	recipientID := recipient.ID
	message := SMSMessage{CareRecipientID: &recipientID, RecipientPhone: recipient.Phone, Content: content, Kind: constants.SMSKindGreeting}
	if err := s.sms.Send(ctx, message); err != nil {
		s.recordFailure(ctx, message, now, err)
		return fmt.Errorf("send greeting to recipient %d: %w", recipient.ID, err)
	}
	return nil
}
func (s *NotificationService) recordFailure(ctx context.Context, message SMSMessage, now time.Time, sendErr error) {
	failure := &model.SMSLog{CareRecipientID: message.CareRecipientID, FamilySubscriptionID: message.FamilySubscriptionID, RecipientPhone: message.RecipientPhone, MessageContent: message.Content, Kind: message.Kind, Result: constants.SMSResultFailed, FailureReason: sendErr.Error(), SentAt: now}
	if err := s.logs.Create(ctx, failure); err != nil {
		s.logger.Error("record greeting failure log", "recipient_phone", message.RecipientPhone, "error", err)
	}
}
