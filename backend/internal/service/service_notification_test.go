package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type flakySMSProvider struct {
	inner      SMSProvider
	mu         sync.Mutex
	failPhones map[string]bool
	calls      []string
}

func (p *flakySMSProvider) Send(ctx context.Context, message SMSMessage) error {
	p.mu.Lock()
	p.calls = append(p.calls, message.RecipientPhone)
	fail := p.failPhones[message.RecipientPhone]
	p.mu.Unlock()
	if fail {
		return errors.New("gateway rejected phone " + message.RecipientPhone)
	}
	return p.inner.Send(ctx, message)
}
func (p *flakySMSProvider) allow(phone string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.failPhones, phone)
}
func newNotificationTestService(t *testing.T, dsn string) (*NotificationService, *repository.RecipientRepository, *repository.SMSLogRepository, *flakySMSProvider) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recipientRepo := repository.NewRecipientRepository(db)
	templateRepo := repository.NewTemplateRepository(db)
	smsLogRepo := repository.NewSMSLogRepository(db)
	if err := templateRepo.Create(context.Background(), &model.SMSTemplate{Name: "日常问候", Category: "general", Content: "{{name}}，愿您今天平安顺心。", Active: true}); err != nil {
		t.Fatal(err)
	}
	provider := &flakySMSProvider{inner: NewLogSMSSender(smsLogRepo, logger), failPhones: map[string]bool{}}
	svc := NewNotificationService(recipientRepo, templateRepo, smsLogRepo, NewSMSService(provider), logger)
	return svc, recipientRepo, smsLogRepo, provider
}
func countLogs(t *testing.T, logs *repository.SMSLogRepository, recipientID uint) ([]model.SMSLog, int64) {
	t.Helper()
	items, total, err := logs.List(context.Background(), 1, 100, &recipientID)
	if err != nil {
		t.Fatal(err)
	}
	return items, total
}

func TestNotificationServiceSendDueGreetingsFaultIsolation(t *testing.T) {
	svc, recipientRepo, smsLogRepo, provider := newNotificationTestService(t, ":memory:")
	ctx := context.Background()
	now := time.Now().UTC()
	start := now.Add(-72 * time.Hour)
	greeted := now.Add(-48 * time.Hour)
	recipients := []*model.CareRecipient{
		{Name: "甲", Phone: "100001", CareFrequency: constants.FrequencyDaily, CareStartAt: start, Status: constants.RecipientStatusActive},
		{Name: "乙", Phone: "100002", CareFrequency: constants.FrequencyDaily, CareStartAt: start, Status: constants.RecipientStatusActive, LastGreetingAt: &greeted},
		{Name: "丙", Phone: "100003", CareFrequency: constants.FrequencyDaily, CareStartAt: start, Status: constants.RecipientStatusActive},
		{Name: "丁", Phone: "100004", CareFrequency: constants.FrequencyDaily, CareStartAt: start, Status: constants.RecipientStatusPaused},
		{Name: "戊", Phone: "100005", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(time.Hour), Status: constants.RecipientStatusActive},
	}
	for _, item := range recipients {
		if err := recipientRepo.Create(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	provider.failPhones["100003"] = true

	result, err := svc.SendDueGreetings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Due != 3 || result.Sent != 2 || result.Failed != 1 {
		t.Fatalf("result=%+v, want due=3 sent=2 failed=1", result)
	}
	bad, err := recipientRepo.Get(ctx, recipients[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	if bad.LastGreetingAt != nil {
		t.Fatalf("failed recipient last greeting=%v, want nil", bad.LastGreetingAt)
	}
	failures, total := countLogs(t, smsLogRepo, bad.ID)
	if total != 1 {
		t.Fatalf("failure logs=%d, want 1", total)
	}
	if failures[0].Result != constants.SMSResultFailed || !strings.Contains(failures[0].FailureReason, "gateway rejected phone 100003") {
		t.Fatalf("failure log=%+v, want failed result with reason", failures[0])
	}
	for _, item := range recipients[:2] {
		stored, err := recipientRepo.Get(ctx, item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.LastGreetingAt == nil {
			t.Fatalf("%s was not stamped", item.Name)
		}
		if _, total := countLogs(t, smsLogRepo, item.ID); total != 1 {
			t.Fatalf("%s logs=%d, want 1", item.Name, total)
		}
	}
	sent, _ := countLogs(t, smsLogRepo, recipients[0].ID)
	if !strings.Contains(sent[0].MessageContent, "甲") {
		t.Fatalf("content=%q, want template rendered with name", sent[0].MessageContent)
	}
	for _, item := range recipients[3:] {
		if _, total := countLogs(t, smsLogRepo, item.ID); total != 0 {
			t.Fatalf("%s logs=%d, want 0", item.Name, total)
		}
	}
	for _, phone := range provider.calls {
		if phone == "100004" || phone == "100005" {
			t.Fatalf("unexpected send to %s", phone)
		}
	}

	result, err = svc.SendDueGreetings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Due != 1 || result.Sent != 0 || result.Failed != 1 {
		t.Fatalf("second result=%+v, want due=1 sent=0 failed=1", result)
	}
	if _, total := countLogs(t, smsLogRepo, bad.ID); total != 2 {
		t.Fatalf("failure logs after retry=%d, want 2", total)
	}
	for _, item := range recipients[:2] {
		if _, total := countLogs(t, smsLogRepo, item.ID); total != 1 {
			t.Fatalf("%s logs after retry=%d, want 1 (no resend)", item.Name, total)
		}
	}

	provider.allow("100003")
	result, err = svc.SendDueGreetings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Due != 1 || result.Sent != 1 || result.Failed != 0 {
		t.Fatalf("third result=%+v, want due=1 sent=1 failed=0", result)
	}
	bad, err = recipientRepo.Get(ctx, recipients[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	if bad.LastGreetingAt == nil {
		t.Fatal("recovered recipient was not stamped")
	}
	logs, total := countLogs(t, smsLogRepo, bad.ID)
	if total != 3 {
		t.Fatalf("logs for recovered recipient=%d, want 3", total)
	}
	successes := 0
	for _, item := range logs {
		if item.Result == constants.SMSResultSuccess {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("success logs=%d, want 1", successes)
	}
}

func TestNotificationServiceOverlappingScans(t *testing.T) {
	svc, recipientRepo, smsLogRepo, _ := newNotificationTestService(t, ":memory:")
	ctx := context.Background()
	now := time.Now().UTC()
	item := model.CareRecipient{Name: "甲", Phone: "100001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-72 * time.Hour), Status: constants.RecipientStatusActive}
	if err := recipientRepo.Create(ctx, &item); err != nil {
		t.Fatal(err)
	}
	first, err := recipientRepo.DueForGreeting(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := recipientRepo.DueForGreeting(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("due lists=%d/%d, want 1/1", len(first), len(second))
	}
	sent1, err := svc.SendGreeting(ctx, &first[0], "")
	if err != nil {
		t.Fatal(err)
	}
	sent2, err := svc.SendGreeting(ctx, &second[0], "")
	if err != nil {
		t.Fatal(err)
	}
	if sent1 == sent2 {
		t.Fatalf("sent1=%v sent2=%v, want exactly one success", sent1, sent2)
	}
	if _, total := countLogs(t, smsLogRepo, item.ID); total != 1 {
		t.Fatalf("success logs=%d, want exactly 1", total)
	}
}

func TestNotificationServiceConcurrentScans(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000", filepath.Join(t.TempDir(), "notify.db"))
	svc, recipientRepo, smsLogRepo, _ := newNotificationTestService(t, dsn)
	ctx := context.Background()
	now := time.Now().UTC()
	item := model.CareRecipient{Name: "甲", Phone: "100001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-72 * time.Hour), Status: constants.RecipientStatusActive}
	if err := recipientRepo.Create(ctx, &item); err != nil {
		t.Fatal(err)
	}
	const scans = 4
	results := make(chan GreetingBatchResult, scans)
	errs := make(chan error, scans)
	var wg sync.WaitGroup
	for i := 0; i < scans; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := svc.SendDueGreetings(ctx)
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	sentTotal := 0
	for result := range results {
		sentTotal += result.Sent
	}
	if sentTotal != 1 {
		t.Fatalf("concurrent scans sent=%d, want exactly 1", sentTotal)
	}
	if _, total := countLogs(t, smsLogRepo, item.ID); total != 1 {
		t.Fatalf("success logs after concurrent scans=%d, want exactly 1", total)
	}
}
