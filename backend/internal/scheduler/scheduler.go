package scheduler

import (
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/service"
	"github.com/robfig/cron/v3"
	"log/slog"
)

type Scheduler struct {
	cron   *cron.Cron
	logger *slog.Logger
}

func New(notification *service.NotificationService, alerts *service.AlertService, greetingSpec, alertSpec string, logger *slog.Logger) (*Scheduler, error) {
	c := cron.New()
	if _, err := c.AddFunc(greetingSpec, func() {
		count, err := notification.SendDueGreetings(contextBackground())
		if err != nil {
			logger.Error("scheduled greetings failed", "error", err)
		} else {
			logger.Info("scheduled greetings finished", "count", count)
		}
	}); err != nil {
		return nil, fmt.Errorf("register greeting cron: %w", err)
	}
	if _, err := c.AddFunc(alertSpec, func() {
		count, err := alerts.SendOverdueAlerts(contextBackground())
		if err != nil {
			logger.Error("scheduled alerts failed", "error", err)
		} else {
			logger.Info("scheduled alerts finished", "count", count)
		}
	}); err != nil {
		return nil, fmt.Errorf("register alert cron: %w", err)
	}
	return &Scheduler{cron: c, logger: logger}, nil
}
func (s *Scheduler) Start() { s.cron.Start(); s.logger.Info("scheduler started") }
func (s *Scheduler) Stop()  { ctx := s.cron.Stop(); <-ctx.Done(); s.logger.Info("scheduler stopped") }
