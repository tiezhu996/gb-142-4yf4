package service

import (
	"context"
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestRecipientServiceCreateAndConfirm(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewRecipientService(repository.NewRecipientRepository(db), repository.NewReplyConfirmRepository(db), logger)
	item, err := service.Create(context.Background(), dto.CreateRecipientRequest{Name: "王阿姨", Phone: "13800138000", CareFrequency: "daily", CareStartAt: time.Now().UTC().Format(time.RFC3339)})
	if err != nil {
		t.Fatal(err)
	}
	confirmed, err := service.Confirm(context.Background(), item.ID, dto.ConfirmRecipientRequest{Channel: "manual", Note: "phone call"})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.LastConfirmedAt == nil {
		t.Fatal("confirmation timestamp was not stored")
	}
}
