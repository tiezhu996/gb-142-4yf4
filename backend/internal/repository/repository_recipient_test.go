package repository

import (
	"context"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
