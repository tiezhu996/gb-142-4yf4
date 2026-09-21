package service

import (
	"context"
	"fmt"
	"github.com/blueship581/gbcarenotify/internal/model"
	"github.com/blueship581/gbcarenotify/internal/repository"
)

func SeedDefaultTemplates(ctx context.Context, templates *repository.TemplateRepository) error {
	items, err := templates.List(ctx, "")
	if err != nil {
		return fmt.Errorf("list templates for seed: %w", err)
	}
	if len(items) > 0 {
		return nil
	}
	for _, item := range []model.SMSTemplate{{Name: "默认日常问候", Category: "general", Content: "{{name}}，您好，愿您今天平安顺心。如方便，请回复“已阅”报个平安。", Active: true}, {Name: "天气关怀", Category: "weather", Content: "{{name}}，天气变化请注意添衣和出行安全，愿您一切安好。", Active: true}, {Name: "节日问候", Category: "festival", Content: "{{name}}，节日安康，家人牵挂您，愿您幸福顺遂。", Active: true}} {
		copy := item
		if err := templates.Create(ctx, &copy); err != nil {
			return fmt.Errorf("seed template: %w", err)
		}
	}
	return nil
}
