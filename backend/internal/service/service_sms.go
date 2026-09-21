package service

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"log/slog"
	"time"
)

type SMSMessage struct {
	CareRecipientID      *uint
	FamilySubscriptionID *uint
	RecipientPhone       string
	Content              string
	Kind                 string
}
type SMSProvider interface {
	Send(context.Context, SMSMessage) error
}
type LogSMSSender struct {
	logs   *repository.SMSLogRepository
	logger *slog.Logger
}

func NewLogSMSSender(logs *repository.SMSLogRepository, logger *slog.Logger) *LogSMSSender {
	return &LogSMSSender{logs: logs, logger: logger}
}
func (s *LogSMSSender) Send(ctx context.Context, message SMSMessage) error {
	if message.RecipientPhone == "" {
		return fmt.Errorf("send sms: recipient phone is empty")
	}
	logItem := &model.SMSLog{CareRecipientID: message.CareRecipientID, FamilySubscriptionID: message.FamilySubscriptionID, RecipientPhone: message.RecipientPhone, MessageContent: message.Content, Kind: message.Kind, Result: constants.SMSResultSuccess, SentAt: time.Now().UTC()}
	if err := s.logs.Create(ctx, logItem); err != nil {
		return err
	}
	s.logger.Info("sms sent by log provider", "sms_log_id", logItem.ID, "kind", message.Kind, "recipient_phone", message.RecipientPhone)
	return nil
}

type SMSService struct{ provider SMSProvider }

func NewSMSService(provider SMSProvider) *SMSService { return &SMSService{provider: provider} }
func (s *SMSService) Send(ctx context.Context, message SMSMessage) error {
	return s.provider.Send(ctx, message)
}
