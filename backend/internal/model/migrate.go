package model

import "gorm.io/gorm"

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&CareRecipient{}, &FamilySubscription{}, &SMSTemplate{}, &SMSLog{}, &ReplyConfirm{})
}
