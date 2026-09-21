package model

import "time"

type ReplyConfirm struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CareRecipientID uint      `gorm:"not null;index" json:"care_recipient_id"`
	Channel         string    `gorm:"size:32;not null" json:"channel"`
	Note            string    `gorm:"size:500" json:"note"`
	ConfirmedAt     time.Time `gorm:"not null" json:"confirmed_at"`
	CreatedAt       time.Time `json:"created_at"`
}
