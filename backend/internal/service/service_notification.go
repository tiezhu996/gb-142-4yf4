package service

import (
	"context"
	"errors"
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
	logs       *repository.SMSLogRepository
	sms        *SMSService
	logger     *slog.Logger
}

func NewNotificationService(recipients *repository.RecipientRepository, templates *repository.TemplateRepository, logs *repository.SMSLogRepository, sms *SMSService, logger *slog.Logger) *NotificationService {
	return &NotificationService{recipients: recipients, templates: templates, logs: logs, sms: sms, logger: logger}
}

// SendDueGreetings scans every recipient that is due for a greeting in one pass.
// A failure for a single number is recorded and the scan continues with the
// remaining recipients; failed recipients keep their previous greeting time so
// the next scan retries them. Concurrent scans claiming the same recipient only
// produce one successful send and one success log.
func (s *NotificationService) SendDueGreetings(ctx context.Context) (int, error) {
	now := time.Now().UTC().Truncate(time.Second)
	items, err := s.recipients.DueForGreeting(ctx, now)
	if err != nil {
		return 0, err
	}
	sent, failed := 0, 0
	for i := range items {
		switch err := s.sendOneGreeting(ctx, &items[i], "", now); {
		case err == nil:
			sent++
		case errors.Is(err, errGreetingClaimed):
			s.logger.Info("greeting already claimed by another scan", "recipient_id", items[i].ID)
		default:
			failed++
			s.logger.Error("greeting failed", "recipient_id", items[i].ID, "error", err)
		}
	}
	s.logger.Info("greeting batch finished", "due", len(items), "sent", sent, "failed", failed)
	return sent, nil
}

var errGreetingClaimed = errors.New("greeting already claimed by another scan")

func (s *NotificationService) sendOneGreeting(ctx context.Context, recipient *model.CareRecipient, category string, now time.Time) error {
	template, err := s.templates.RandomActive(ctx, category)
	if errors.Is(err, repository.ErrNotFound) {
		template = &model.SMSTemplate{Content: "{{name}}，您好，愿您今天平安顺心。如方便，请回复“已阅”报个平安。"}
	} else if err != nil {
		return err
	}
	content := strings.ReplaceAll(template.Content, "{{name}}", recipient.Name)

	// Atomically claim the recipient before sending so concurrent scans cannot
	// dispatch the same greeting twice.
	claimed, err := s.recipients.ClaimForGreeting(ctx, recipient.ID, recipient.LastGreetingAt, now)
	if err != nil {
		return err
	}
	if !claimed {
		return errGreetingClaimed
	}

	recipientID := recipient.ID
	if err := s.sms.Send(ctx, SMSMessage{CareRecipientID: &recipientID, RecipientPhone: recipient.Phone, Content: content, Kind: constants.SMSKindGreeting}); err != nil {
		s.recordFailure(ctx, recipient, content, err)
		// Restore the previous greeting time so the next scan retries.
		if released, releaseErr := s.recipients.ReleaseGreetingClaim(ctx, recipient.ID, now, recipient.LastGreetingAt); releaseErr != nil {
			return fmt.Errorf("roll back greeting time for recipient %d after send failure: %w", recipient.ID, releaseErr)
		} else if !released {
			s.logger.Warn("greeting claim was replaced before rollback", "recipient_id", recipient.ID)
		}
		return err
	}
	recipient.LastGreetingAt = &now
	s.logger.Info("greeting dispatched", "recipient_id", recipient.ID)
	return nil
}

func (s *NotificationService) recordFailure(ctx context.Context, recipient *model.CareRecipient, content string, cause error) {
	recipientID := recipient.ID
	logItem := &model.SMSLog{
		CareRecipientID: &recipientID,
		RecipientPhone:  recipient.Phone,
		MessageContent:  content,
		Kind:            constants.SMSKindGreeting,
		Result:          constants.SMSResultFailed,
		FailureReason:   cause.Error(),
		SentAt:          time.Now().UTC(),
	}
	if err := s.logs.Create(ctx, logItem); err != nil {
		s.logger.Error("record failed greeting log", "recipient_id", recipient.ID, "error", err)
	}
}
