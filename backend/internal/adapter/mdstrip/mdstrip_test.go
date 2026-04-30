package mdstrip_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/mdstrip"
)

func TestStrip(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"### 2.1. Роли систем", "2.1. Роли систем"},
		{"## Заголовок", "Заголовок"},
		{"# Главный", "Главный"},
		{"**bold text**", "bold text"},
		{"Текст **с bold** внутри", "Текст с bold внутри"},
		{"*italic text*", "italic text"},
		{"Обычный текст", "Обычный текст"},
		{"|---------|----------|", ""},
		{"| Позиция | Стоимость |", "Позиция — Стоимость"},
		{"| Bitrix24 | 13 990 ₽ |", "Bitrix24 — 13 990 ₽"},
		{"- Пункт списка", "- Пункт списка"},
		{"* Пункт списка", "* Пункт списка"},
		{"[ваш email](mailto:x)", "ваш email"},
		{"[ваш телефон][ref]", "ваш телефон"},
		{"Контакты: [email][телефон]", "Контакты: email"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mdstrip.Strip(tt.input)
			if got != tt.want {
				t.Errorf("Strip(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
