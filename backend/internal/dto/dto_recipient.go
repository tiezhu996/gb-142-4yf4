package dto

type CreateRecipientRequest struct {
	Name          string `json:"name" validate:"required,max=100"`
	Phone         string `json:"phone" validate:"required,min=6,max=32"`
	CareFrequency string `json:"care_frequency" validate:"required,oneof=daily weekly monthly"`
	CareStartAt   string `json:"care_start_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	Status        string `json:"status" validate:"omitempty,oneof=Active Paused"`
}
type UpdateRecipientRequest struct {
	Name          string `json:"name" validate:"required,max=100"`
	Phone         string `json:"phone" validate:"required,min=6,max=32"`
	CareFrequency string `json:"care_frequency" validate:"required,oneof=daily weekly monthly"`
	CareStartAt   string `json:"care_start_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	Status        string `json:"status" validate:"required,oneof=Active Paused"`
}
type ConfirmRecipientRequest struct {
	Channel string `json:"channel" validate:"omitempty,oneof=sms manual webhook"`
	Note    string `json:"note" validate:"omitempty,max=500"`
}
type CreateSubscriptionRequest struct {
	FamilyName  string `json:"family_name" validate:"required,max=100"`
	FamilyPhone string `json:"family_phone" validate:"required,min=6,max=32"`
	Active      *bool  `json:"active"`
}
