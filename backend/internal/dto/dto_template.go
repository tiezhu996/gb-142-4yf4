package dto

type CreateTemplateRequest struct {
	Name     string `json:"name" validate:"required,max=100"`
	Category string `json:"category" validate:"required,oneof=festival weather season general"`
	Content  string `json:"content" validate:"required,max=1000"`
	Active   *bool  `json:"active"`
}
type UpdateTemplateRequest struct {
	Name     string `json:"name" validate:"required,max=100"`
	Category string `json:"category" validate:"required,oneof=festival weather season general"`
	Content  string `json:"content" validate:"required,max=1000"`
	Active   bool   `json:"active"`
}
