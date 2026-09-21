package repository

import (
	"context"
	"github.com/blueship581/gbcarenotify/internal/constants"
	"github.com/blueship581/gbcarenotify/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
	"time"
)

func newClaimDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "claim.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

// A second claim against the stale value must lose, and a claim after rollback
// must win again so the next scan can retry.
func TestClaimForGreetingIsExclusiveAndReclaimableAfterRelease(t *testing.T) {
	db := newClaimDB(t)
	repo := NewRecipientRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	r := model.CareRecipient{Name: "A", Phone: "400001", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-time.Hour), Status: constants.RecipientStatusActive}
	if err := repo.Create(context.Background(), &r); err != nil {
		t.Fatal(err)
	}
	claimedAt := now

	won, err := repo.ClaimForGreeting(context.Background(), r.ID, nil, claimedAt)
	if err != nil || !won {
		t.Fatalf("first claim won=%v err=%v, want true", won, err)
	}
	// Concurrent scan still sees no previous greeting time and must lose.
	won, err = repo.ClaimForGreeting(context.Background(), r.ID, nil, claimedAt)
	if err != nil {
		t.Fatal(err)
	}
	if won {
		t.Fatal("second claim with stale previous value must lose")
	}

	released, err := repo.ReleaseGreetingClaim(context.Background(), r.ID, claimedAt, nil)
	if err != nil || !released {
		t.Fatalf("release released=%v err=%v, want true", released, err)
	}
	got, err := repo.Get(context.Background(), r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastGreetingAt != nil {
		t.Fatalf("last_greeting_at=%v, want NULL after rollback", got.LastGreetingAt)
	}
	won, err = repo.ClaimForGreeting(context.Background(), r.ID, nil, claimedAt.Add(time.Second))
	if err != nil || !won {
		t.Fatalf("reclaim after rollback won=%v err=%v, want true", won, err)
	}
}

// Releasing with a non-matching claim timestamp must not overwrite a newer
// successful greeting time.
func TestReleaseGreetingClaimKeepsNewerSuccess(t *testing.T) {
	db := newClaimDB(t)
	repo := NewRecipientRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	previous := now.Add(-48 * time.Hour)
	r := model.CareRecipient{Name: "B", Phone: "400002", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-72 * time.Hour), Status: constants.RecipientStatusActive, LastGreetingAt: &previous}
	if err := repo.Create(context.Background(), &r); err != nil {
		t.Fatal(err)
	}
	claimAt := now
	if won, err := repo.ClaimForGreeting(context.Background(), r.ID, &previous, claimAt); err != nil || !won {
		t.Fatalf("claim won=%v err=%v", won, err)
	}
	// Someone else's later successful run replaced the greeting time.
	later := now.Add(time.Minute)
	if err := db.Model(&model.CareRecipient{}).Where("id = ?", r.ID).Update("last_greeting_at", later).Error; err != nil {
		t.Fatal(err)
	}
	released, err := repo.ReleaseGreetingClaim(context.Background(), r.ID, claimAt, &previous)
	if err != nil {
		t.Fatal(err)
	}
	if released {
		t.Fatal("rollback with stale claim must not touch the newer success")
	}
	got, err := repo.Get(context.Background(), r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastGreetingAt == nil || !got.LastGreetingAt.Equal(later) {
		t.Fatalf("last_greeting_at=%v, want %v", got.LastGreetingAt, later)
	}
}

// Paused recipients cannot be claimed even if a stale due snapshot is supplied.
func TestClaimForGreetingRejectsPaused(t *testing.T) {
	db := newClaimDB(t)
	repo := NewRecipientRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	r := model.CareRecipient{Name: "P", Phone: "400003", CareFrequency: constants.FrequencyDaily, CareStartAt: now.Add(-time.Hour), Status: constants.RecipientStatusPaused}
	if err := repo.Create(context.Background(), &r); err != nil {
		t.Fatal(err)
	}
	won, err := repo.ClaimForGreeting(context.Background(), r.ID, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if won {
		t.Fatal("paused recipient must not be claimable")
	}
}
