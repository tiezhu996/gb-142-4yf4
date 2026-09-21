package model

import "time"

type FamilySubscription struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CareRecipientID uint      `gorm:"not null;index" json:"care_recipient_id"`
	FamilyName      string    `gorm:"size:100;not null" json:"family_name"`
	FamilyPhone     string    `gorm:"size:32;not null" json:"family_phone"`
	Active          bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt       time.Time `json:"created_at"`
}
