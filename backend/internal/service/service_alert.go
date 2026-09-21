package service

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"log/slog"
	"time"
)

type AlertService struct {
	recipients    *repository.RecipientRepository
	subscriptions *repository.SubscriptionRepository
	sms           *SMSService
	timeout       time.Duration
	logger        *slog.Logger
}

func NewAlertService(recipients *repository.RecipientRepository, subscriptions *repository.SubscriptionRepository, sms *SMSService, timeout time.Duration, logger *slog.Logger) *AlertService {
	return &AlertService{recipients: recipients, subscriptions: subscriptions, sms: sms, timeout: timeout, logger: logger}
}
func (s *AlertService) SendOverdueAlerts(ctx context.Context) (int, error) {
	items, err := s.recipients.Overdue(ctx, time.Now().UTC().Add(-s.timeout))
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, recipient := range items {
		subs, err := s.subscriptions.ListActiveByRecipient(ctx, recipient.ID)
		if err != nil {
			return sent, fmt.Errorf("list subscriptions for recipient %d: %w", recipient.ID, err)
		}
		for _, sub := range subs {
			rid, sid := recipient.ID, sub.ID
			content := fmt.Sprintf("关怀提醒：%s 已超过确认时限未回复，请尽快联系确认。", recipient.Name)
			if err := s.sms.Send(ctx, SMSMessage{CareRecipientID: &rid, FamilySubscriptionID: &sid, RecipientPhone: sub.FamilyPhone, Content: content, Kind: constants.SMSKindAlert}); err != nil {
				return sent, fmt.Errorf("send alert to subscription %d: %w", sub.ID, err)
			}
			sent++
		}
	}
	s.logger.Info("overdue alerts dispatched", "count", sent)
	return sent, nil
}
