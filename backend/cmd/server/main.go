package main

import (
	"context"
	"errors"
	"github.com/blueship581/gbcarenotify/internal/config"
	"github.com/blueship581/gbcarenotify/internal/handler"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"github.com/blueship581/gbcarenotify/internal/router"
	"github.com/blueship581/gbcarenotify/internal/scheduler"
	"github.com/blueship581/gbcarenotify/internal/service"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}
	db, err := connect(cfg)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	if err := model.AutoMigrate(db); err != nil {
		logger.Error("migrate database", "error", err)
		os.Exit(1)
	}
	recipientRepo := repository.NewRecipientRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	templateRepo := repository.NewTemplateRepository(db)
	smsLogRepo := repository.NewSMSLogRepository(db)
	confirmRepo := repository.NewReplyConfirmRepository(db)
	if err := service.SeedDefaultTemplates(context.Background(), templateRepo); err != nil {
		logger.Error("seed templates", "error", err)
		os.Exit(1)
	}
	recipientService := service.NewRecipientService(recipientRepo, confirmRepo, logger)
	subscriptionService := service.NewSubscriptionService(recipientRepo, subscriptionRepo, logger)
	templateService := service.NewTemplateService(templateRepo)
	provider := service.NewLogSMSSender(smsLogRepo, logger)
	smsService := service.NewSMSService(provider)
	notificationService := service.NewNotificationService(recipientRepo, templateRepo, smsService, logger)
	alertService := service.NewAlertService(recipientRepo, subscriptionRepo, smsService, cfg.ConfirmTimeout, logger)
	statusService := service.NewStatusService(recipientRepo, cfg.ConfirmTimeout)
	jobs, err := scheduler.New(notificationService, alertService, cfg.GreetingCronExpression, cfg.AlertCronExpression, logger)
	if err != nil {
		logger.Error("create scheduler", "error", err)
		os.Exit(1)
	}
	jobs.Start()
	engine := router.New(cfg.APIKey, router.Handlers{Recipients: handler.NewRecipientHandler(handler.NewBaseHandler(), recipientService), Subscriptions: handler.NewSubscriptionHandler(handler.NewBaseHandler(), subscriptionService), Templates: handler.NewTemplateHandler(handler.NewBaseHandler(), templateService), SMSLogs: handler.NewSMSLogHandler(handler.NewBaseHandler(), smsLogRepo), Alerts: handler.NewAlertHandler(handler.NewBaseHandler(), notificationService, alertService), Admin: handler.NewAdminHandler(handler.NewBaseHandler(), statusService)})
	server := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("http server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	jobs.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown server", "error", err)
	}
}
func connect(cfg config.Config) (*gorm.DB, error) {
	var last error
	for i := 0; i < 20; i++ {
		db, err := gorm.Open(mysql.Open(cfg.MySQLDSN()), &gorm.Config{})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil {
				pingErr := sqlDB.Ping()
				if pingErr == nil {
					return db, nil
				}
				last = pingErr
			} else {
				last = sqlErr
			}
		} else {
			last = err
		}
		time.Sleep(2 * time.Second)
	}
	return nil, last
}
