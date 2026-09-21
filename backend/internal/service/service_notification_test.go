package service

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// controlledProvider wraps the real log-writing sender so successful sends keep
// producing success logs, while configured phones fail with a recorded reason.
type controlledProvider struct {
	delegate SMSProvider
	mu       sync.Mutex
	fail     map[string]bool
	calls    map[string]int
}

func newControlledProvider(logs *repository.SMSLogRepository, logger *slog.Logger) *controlledProvider {
	return &controlledProvider{
		delegate: NewLogSMSSender(logs, logger),
		fail:     map[string]bool{},
		calls:    map[string]int{},
	}
}

func (p *controlledProvider) Send(ctx context.Context, message SMSMessage) error {
	p.mu.Lock()
	p.calls[message.RecipientPhone]++
	shouldFail := p.fail[message.RecipientPhone]
	p.mu.Unlock()
	if shouldFail {
		return fmt.Errorf("provider rejected phone %s", message.RecipientPhone)
	}
	return p.delegate.Send(ctx, message)
}

func (p *controlledProvider) callCount(phone string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls[phone]
}

func newNotificationFixture(t *testing.T) (*gorm.DB, *NotificationService, *controlledProvider) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "greeting.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(4)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recipients := repository.NewRecipientRepository(db)
	templates := repository.NewTemplateRepository(db)
	logs := repository.NewSMSLogRepository(db)
	if err := templates.Create(context.Background(), &model.SMSTemplate{Name: "问候", Category: "general", Content: "{{name}}，您好，今天也请保重身体。", Active: true}); err != nil {
		t.Fatal(err)
	}
	provider := newControlledProvider(logs, logger)
	svc := NewNotificationService(recipients, templates, logs, NewSMSService(provider), logger)
	return db, svc, provider
}

func createRecipient(t *testing.T, db *gorm.DB, r model.CareRecipient) uint {
	t.Helper()
	if err := db.Create(&r).Error; err != nil {
		t.Fatal(err)
	}
	return r.ID
}

func greetingLogs(t *testing.T, db *gorm.DB, recipientID uint) []model.SMSLog {
	t.Helper()
	var items []model.SMSLog
	if err := db.Where("care_recipient_id = ? AND kind = ?", recipientID, constants.SMSKindGreeting).Order("id asc").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	return items
}

// A failing number is isolated: the failure and reason are logged, its greeting
// time stays untouched, and the scan still sends to the other due recipients.
func TestSendDueGreetingsIsolatesFailureAndContinues(t *testing.T) {
	db, svc, provider := newNotificationFixture(t)
	now := time.Now().UTC()
	badID := createRecipient(t, db, model.CareRecipient{Name: "坏号码", Phone: "100001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusActive})
	goodID := createRecipient(t, db, model.CareRecipient{Name: "好号码", Phone: "100002", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusActive})
	pausedID := createRecipient(t, db, model.CareRecipient{Name: "暂停", Phone: "100003", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusPaused})
	provider.fail["100001"] = true

	sent, err := svc.SendDueGreetings(context.Background())
	if err != nil {
		t.Fatalf("batch returned error: %v", err)
	}
	if sent != 1 {
		t.Fatalf("sent=%d, want 1 (failure must not abort the batch)", sent)
	}
	if provider.callCount("100001") != 1 || provider.callCount("100002") != 1 || provider.callCount("100003") != 0 {
		t.Fatalf("calls bad=%d good=%d paused=%d", provider.callCount("100001"), provider.callCount("100002"), provider.callCount("100003"))
	}

	var bad, good, paused model.CareRecipient
	db.First(&bad, badID)
	db.First(&good, goodID)
	db.First(&paused, pausedID)
	if bad.LastGreetingAt != nil {
		t.Fatal("failed recipient must not update last_greeting_at")
	}
	if good.LastGreetingAt == nil {
		t.Fatal("successful recipient must update last_greeting_at")
	}
	if paused.LastGreetingAt != nil {
		t.Fatal("paused recipient must never be sent to")
	}

	badLogs := greetingLogs(t, db, badID)
	if len(badLogs) != 1 || badLogs[0].Result != constants.SMSResultFailed || badLogs[0].FailureReason == "" {
		t.Fatalf("bad logs=%+v, want one failed log with reason", badLogs)
	}
	goodLogs := greetingLogs(t, db, goodID)
	if len(goodLogs) != 1 || goodLogs[0].Result != constants.SMSResultSuccess {
		t.Fatalf("good logs=%+v, want one success log", goodLogs)
	}
}

// The failed object is retried on the next scan; the already successful object
// is not sent again once the frequency window has not elapsed.
func TestSendDueGreetingsRetriesFailedAndSkipsSucceeded(t *testing.T) {
	db, svc, provider := newNotificationFixture(t)
	now := time.Now().UTC()
	badID := createRecipient(t, db, model.CareRecipient{Name: "坏号码", Phone: "200001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusActive})
	goodID := createRecipient(t, db, model.CareRecipient{Name: "好号码", Phone: "200002", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusActive})
	provider.fail["200001"] = true

	if _, err := svc.SendDueGreetings(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Provider recovers before the next scan.
	provider.fail["200001"] = false
	sent, err := svc.SendDueGreetings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sent != 1 {
		t.Fatalf("second scan sent=%d, want 1 (retry only)", sent)
	}
	if provider.callCount("200001") != 2 || provider.callCount("200002") != 1 {
		t.Fatalf("calls bad=%d good=%d, want 2 and 1", provider.callCount("200001"), provider.callCount("200002"))
	}

	var bad model.CareRecipient
	db.First(&bad, badID)
	if bad.LastGreetingAt == nil {
		t.Fatal("retried recipient should be marked after success")
	}
	badLogs := greetingLogs(t, db, badID)
	if len(badLogs) != 2 || badLogs[0].Result != constants.SMSResultFailed || badLogs[1].Result != constants.SMSResultSuccess {
		t.Fatalf("bad logs=%+v, want failed then success", badLogs)
	}
	if logs := greetingLogs(t, db, goodID); len(logs) != 1 {
		t.Fatalf("good logs=%d, want exactly 1 (no resend)", len(logs))
	}
}

// Concurrent scans of the same due objects leave one success and one success
// log per recipient.
func TestSendDueGreetingsConcurrentScansDeduplicate(t *testing.T) {
	db, svc, _ := newNotificationFixture(t)
	now := time.Now().UTC()
	const n = 5
	for i := 0; i < n; i++ {
		createRecipient(t, db, model.CareRecipient{
			Name:          fmt.Sprintf("并发%d", i),
			Phone:         fmt.Sprintf("300%03d", i),
			CareFrequency: constants.FrequencyDaily,
			CareStartAt:   now.Add(-48 * time.Hour),
			Status:        constants.RecipientStatusActive,
		})
	}

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.SendDueGreetings(context.Background()); err != nil {
				t.Errorf("concurrent scan: %v", err)
			}
		}()
	}
	wg.Wait()

	var recipients []model.CareRecipient
	if err := db.Where("status = ?", constants.RecipientStatusActive).Find(&recipients).Error; err != nil {
		t.Fatal(err)
	}
	if len(recipients) != n {
		t.Fatalf("recipients=%d", len(recipients))
	}
	var logs []model.SMSLog
	if err := db.Where("kind = ? AND result = ?", constants.SMSKindGreeting, constants.SMSResultSuccess).Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != n {
		t.Fatalf("success logs=%d, want exactly %d", len(logs), n)
	}
	seen := map[uint]int{}
	for _, l := range logs {
		if l.CareRecipientID == nil {
			t.Fatal("success log missing recipient id")
		}
		seen[*l.CareRecipientID]++
	}
	for _, r := range recipients {
		if seen[r.ID] != 1 {
			t.Fatalf("recipient %d success logs=%d, want 1", r.ID, seen[r.ID])
		}
	}
}
