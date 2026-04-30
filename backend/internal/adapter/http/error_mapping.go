package http

import "strings"

type errorMessages struct {
	ru string
	en string
}

var errorPatterns = []struct {
	match func(errStr string) bool
	msgs  errorMessages
}{
	{
		match: func(s string) bool {
			return strings.Contains(s, "status 429") || strings.Contains(s, "rate_limit") || strings.Contains(s, "quota")
		},
		msgs: errorMessages{
			ru: "Превышен лимит запросов к AI-провайдеру. Подождите и попробуйте снова.",
			en: "AI provider rate limit exceeded. Please wait and try again.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "status 401") || strings.Contains(s, "invalid_api_key") ||
				strings.Contains(s, "invalid x-api-key") || strings.Contains(s, "incorrect api key")
		},
		msgs: errorMessages{
			ru: "Неверный API-ключ. Проверьте ключ в настройках.",
			en: "Invalid API key. Please check your key in settings.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "status 404") || strings.Contains(s, "model_not_found") ||
				strings.Contains(s, "does not exist") || strings.Contains(s, "is blocked")
		},
		msgs: errorMessages{
			ru: "Модель не найдена или заблокирована у провайдера. Проверьте название модели.",
			en: "Model not found or blocked. Please check the model name.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "parse llm response") || strings.Contains(s, "unexpected end of json")
		},
		msgs: errorMessages{
			ru: "Ответ модели обрезан — попробуйте модель с большим контекстом или уменьшите объём документов.",
			en: "Model response was truncated — try a model with a larger context window or reduce document size.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "empty template") || strings.Contains(s, "too large")
		},
		msgs: errorMessages{
			ru: "Файл пустой или слишком большой для обработки.",
			en: "File is empty or too large to process.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "context deadline exceeded") || strings.Contains(s, "context canceled") ||
				strings.Contains(s, "client.timeout")
		},
		msgs: errorMessages{
			ru: "Превышено время ожидания ответа от AI. Попробуйте ещё раз или выберите более быструю модель.",
			en: "AI response timed out. Please try again or choose a faster model.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "send request") || strings.Contains(s, "connection refused") ||
				strings.Contains(s, "no such host") || strings.Contains(s, "dial tcp")
		},
		msgs: errorMessages{
			ru: "Не удалось подключиться к AI-провайдеру. Проверьте URL и интернет-соединение.",
			en: "Could not connect to AI provider. Check the URL and your internet connection.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "status 402") || strings.Contains(s, "payment required") ||
				strings.Contains(s, "insufficient") || strings.Contains(s, "billing")
		},
		msgs: errorMessages{
			ru: "Недостаточно средств на аккаунте AI-провайдера. Пополните баланс.",
			en: "Insufficient funds on your AI provider account. Please top up.",
		},
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "status 5") || strings.Contains(s, "internal server error") ||
				strings.Contains(s, "bad gateway") || strings.Contains(s, "service unavailable")
		},
		msgs: errorMessages{
			ru: "AI-провайдер временно недоступен. Попробуйте через пару минут.",
			en: "AI provider is temporarily unavailable. Try again in a few minutes.",
		},
	},
}

func mapErrorToUserMessage(errStr string, langName string) string {
	lower := strings.ToLower(errStr)
	for _, p := range errorPatterns {
		if p.match(lower) {
			if langName == "English" {
				return p.msgs.en
			}
			return p.msgs.ru
		}
	}
	if langName == "English" {
		return "An error occurred. Please try again."
	}
	return "Произошла ошибка. Попробуйте ещё раз."
}
