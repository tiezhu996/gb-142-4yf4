package model

import "time"

type SMSTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Category  string    `gorm:"size:32;not null;index" json:"category"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Active    bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
