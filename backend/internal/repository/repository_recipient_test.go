package repository

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestRecipientRepositoryDueAndOverdue(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRecipientRepository(db)
	now := time.Now().UTC()
	cases := []struct {
		name        string
		recipient   model.CareRecipient
		wantDue     bool
		wantOverdue bool
	}{{"new active", model.CareRecipient{Name: "A", Phone: "100001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-time.Hour), Status: constants.RecipientStatusActive}, true, true}, {"confirmed", model.CareRecipient{Name: "B", Phone: "100002", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-time.Hour), Status: constants.RecipientStatusActive, LastConfirmedAt: &now}, true, false}, {"paused", model.CareRecipient{Name: "C", Phone: "100003", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-time.Hour), Status: constants.RecipientStatusPaused}, false, false}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := repo.Create(context.Background(), &tt.recipient); err != nil {
				t.Fatal(err)
			}
		})
	}
	due, err := repo.DueForGreeting(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 2 {
		t.Fatalf("due length=%d, want 2", len(due))
	}
	overdue, err := repo.Overdue(context.Background(), now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(overdue) != 1 || overdue[0].Name != "A" {
		t.Fatalf("overdue=%+v, want A", overdue)
	}
}

func TestRecipientRepositoryClaimForGreeting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRecipientRepository(db)
	now := time.Now().UTC()
	greeted := now.Add(-48 * time.Hour)
	active := model.CareRecipient{Name: "A", Phone: "100001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusActive}
	recent := model.CareRecipient{Name: "B", Phone: "100002", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusActive, LastGreetingAt: &now}
	paused := model.CareRecipient{Name: "C", Phone: "100003", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusPaused}
	stale := model.CareRecipient{Name: "D", Phone: "100004", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-72 * time.Hour), Status: constants.RecipientStatusActive, LastGreetingAt: &greeted}
	for _, item := range []*model.CareRecipient{&active, &recent, &paused, &stale} {
		if err := repo.Create(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	claimed, err := repo.ClaimForGreeting(context.Background(), active.ID, active.CareFrequency, now)
	if err != nil || !claimed {
		t.Fatalf("first claim=%v err=%v, want true", claimed, err)
	}
	claimed, err = repo.ClaimForGreeting(context.Background(), active.ID, active.CareFrequency, now)
	if err != nil || claimed {
		t.Fatalf("second claim=%v err=%v, want false", claimed, err)
	}
	stored, err := repo.Get(context.Background(), active.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastGreetingAt == nil || !stored.LastGreetingAt.Equal(now) {
		t.Fatalf("last greeting=%v, want %v", stored.LastGreetingAt, now)
	}
	if err := repo.ReleaseGreetingClaim(context.Background(), active.ID, now, nil); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.Get(context.Background(), active.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastGreetingAt != nil {
		t.Fatalf("last greeting after release=%v, want nil", stored.LastGreetingAt)
	}
	claimed, err = repo.ClaimForGreeting(context.Background(), active.ID, active.CareFrequency, now)
	if err != nil || !claimed {
		t.Fatalf("claim after release=%v err=%v, want true", claimed, err)
	}
	claimed, err = repo.ClaimForGreeting(context.Background(), stale.ID, stale.CareFrequency, now)
	if err != nil || !claimed {
		t.Fatalf("stale claim=%v err=%v, want true", claimed, err)
	}
	if err := repo.ReleaseGreetingClaim(context.Background(), stale.ID, now, &greeted); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.Get(context.Background(), stale.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastGreetingAt == nil || !stored.LastGreetingAt.Equal(greeted) {
		t.Fatalf("stale last greeting=%v, want %v", stored.LastGreetingAt, greeted)
	}
	for _, item := range []model.CareRecipient{recent, paused} {
		claimed, err := repo.ClaimForGreeting(context.Background(), item.ID, item.CareFrequency, now)
		if err != nil || claimed {
			t.Fatalf("claim for %s=%v err=%v, want false", item.Name, claimed, err)
		}
	}
}

func TestRecipientRepositoryClaimForGreetingConcurrent(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000", filepath.Join(t.TempDir(), "claim.db"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRecipientRepository(db)
	now := time.Now().UTC()
	item := model.CareRecipient{Name: "A", Phone: "100001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-48 * time.Hour), Status: constants.RecipientStatusActive}
	if err := repo.Create(context.Background(), &item); err != nil {
		t.Fatal(err)
	}
	const scans = 8
	results := make(chan bool, scans)
	errs := make(chan error, scans)
	var wg sync.WaitGroup
	for i := 0; i < scans; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := repo.ClaimForGreeting(context.Background(), item.ID, item.CareFrequency, now)
			if err != nil {
				errs <- err
				return
			}
			results <- claimed
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	claims := 0
	for claimed := range results {
		if claimed {
			claims++
		}
	}
	if claims != 1 {
		t.Fatalf("concurrent claims=%d, want exactly 1", claims)
	}
}
