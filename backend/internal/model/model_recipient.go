package model

import "time"

type CareRecipient struct {
	ID                  uint                 `gorm:"primaryKey" json:"id"`
	Name                string               `gorm:"size:100;not null" json:"name"`
	Phone               string               `gorm:"size:32;not null;uniqueIndex" json:"phone"`
	CareFrequency       string               `gorm:"size:16;not null" json:"care_frequency"`
	CareStartAt         time.Time            `gorm:"not null" json:"care_start_at"`
	Status              string               `gorm:"size:16;not null;index" json:"status"`
	LastConfirmedAt     *time.Time           `json:"last_confirmed_at"`
	LastGreetingAt      *time.Time           `json:"last_greeting_at"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	FamilySubscriptions []FamilySubscription `json:"family_subscriptions,omitempty"`
}
