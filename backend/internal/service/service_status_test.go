package service

import (
	"context"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestStatusServiceList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRecipientRepository(db)
	now := time.Now().UTC()
	paused := model.CareRecipient{Name: "P", Phone: "100001", CareFrequency: "daily", CareStartAt: now.Add(-time.Hour), Status: "Paused"}
	recent := model.CareRecipient{Name: "R", Phone: "100002", CareFrequency: "daily", CareStartAt: now.Add(-time.Hour), Status: "Active", LastConfirmedAt: &now}
	never := model.CareRecipient{Name: "N", Phone: "100003", CareFrequency: "daily", CareStartAt: now.Add(-time.Minute), Status: "Active"}
	overdue := model.CareRecipient{Name: "O", Phone: "100004", CareFrequency: "daily", CareStartAt: now.Add(-48 * time.Hour), Status: "Active"}
	for _, r := range []model.CareRecipient{paused, recent, never, overdue} {
		item := r
		if err := repo.Create(context.Background(), &item); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewStatusService(repo, time.Hour)
	items, total, err := svc.List(context.Background(), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || len(items) != 4 {
		t.Fatalf("total=%d len=%d, want 4", total, len(items))
	}
	states := map[string]string{}
	for _, item := range items {
		states[item.Recipient.Name] = item.State
	}
	if states["P"] != "已暂停" {
		t.Fatalf("P state=%q", states["P"])
	}
	if states["R"] != "平安" {
		t.Fatalf("R state=%q", states["R"])
	}
	if states["N"] != "待确认" {
		t.Fatalf("N state=%q", states["N"])
	}
	if states["O"] != "失联" {
		t.Fatalf("O state=%q", states["O"])
	}
}
