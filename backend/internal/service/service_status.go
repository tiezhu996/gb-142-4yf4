package service

import (
	"context"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"time"
)

type RecipientStatus struct {
	Recipient model.CareRecipient `json:"recipient"`
	State     string              `json:"state"`
	Reason    string              `json:"reason"`
}
type StatusService struct {
	recipients *repository.RecipientRepository
	timeout    time.Duration
}

func NewStatusService(recipients *repository.RecipientRepository, timeout time.Duration) *StatusService {
	return &StatusService{recipients: recipients, timeout: timeout}
}
func (s *StatusService) List(ctx context.Context, page, pageSize int) ([]RecipientStatus, int64, error) {
	items, total, err := s.recipients.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	result := make([]RecipientStatus, 0, len(items))
	now := time.Now().UTC()
	cutoff := now.Add(-s.timeout)
	for _, item := range items {
		state, reason := "失联", "超过确认时限"
		switch {
		case item.Status == constants.RecipientStatusPaused:
			state, reason = "已暂停", "关怀已暂停"
		case item.LastConfirmedAt != nil && !item.LastConfirmedAt.Before(cutoff):
			state, reason = "平安", "近期已确认"
		case item.LastConfirmedAt == nil && item.CareStartAt.After(cutoff):
			state, reason = "待确认", "尚未确认"
		}
		result = append(result, RecipientStatus{Recipient: item, State: state, Reason: reason})
	}
	return result, total, nil
}
