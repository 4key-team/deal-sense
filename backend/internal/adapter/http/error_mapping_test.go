package http

import "testing"

func TestMapErrorToUserMessage(t *testing.T) {
	tests := []struct {
		name    string
		err     string
		lang    string
		wantRu  string
		wantEn  string
	}{
		{
			name:   "rate limit",
			err:    "llm completion: status 429: rate_limit_exceeded",
			wantRu: "Превышен лимит запросов к AI-провайдеру. Подождите и попробуйте снова.",
			wantEn: "AI provider rate limit exceeded. Please wait and try again.",
		},
		{
			name:   "invalid api key",
			err:    "llm completion: status 401: Incorrect API key provided",
			wantRu: "Неверный API-ключ. Проверьте ключ в настройках.",
			wantEn: "Invalid API key. Please check your key in settings.",
		},
		{
			name:   "model not found",
			err:    "llm completion: status 404: model does not exist",
			wantRu: "Модель не найдена или заблокирована у провайдера. Проверьте название модели.",
			wantEn: "Model not found or blocked. Please check the model name.",
		},
		{
			name:   "truncated response",
			err:    "parse llm response: unexpected end of JSON input (raw: {\"sections\":[{\"ti...)",
			wantRu: "Ответ модели обрезан — попробуйте модель с большим контекстом или уменьшите объём документов.",
			wantEn: "Model response was truncated — try a model with a larger context window or reduce document size.",
		},
		{
			name:   "timeout",
			err:    "llm completion: context deadline exceeded",
			wantRu: "Превышено время ожидания ответа от AI. Попробуйте ещё раз или выберите более быструю модель.",
			wantEn: "AI response timed out. Please try again or choose a faster model.",
		},
		{
			name:   "connection refused",
			err:    "llm completion: send request: dial tcp 127.0.0.1:11434: connection refused",
			wantRu: "Не удалось подключиться к AI-провайдеру. Проверьте URL и интернет-соединение.",
			wantEn: "Could not connect to AI provider. Check the URL and your internet connection.",
		},
		{
			name:   "model blocked",
			err:    "llm completion: The model `meta-llama/llama-4` is blocked",
			wantRu: "Модель не найдена или заблокирована у провайдера. Проверьте название модели.",
			wantEn: "Model not found or blocked. Please check the model name.",
		},
		{
			name:   "unknown error",
			err:    "something completely unexpected happened",
			wantRu: "Произошла ошибка. Попробуйте ещё раз.",
			wantEn: "An error occurred. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (ru)", func(t *testing.T) {
			got := mapErrorToUserMessage(tt.err, "Russian")
			if got != tt.wantRu {
				t.Errorf("got %q, want %q", got, tt.wantRu)
			}
		})
		t.Run(tt.name+" (en)", func(t *testing.T) {
			got := mapErrorToUserMessage(tt.err, "English")
			if got != tt.wantEn {
				t.Errorf("got %q, want %q", got, tt.wantEn)
			}
		})
	}
}
