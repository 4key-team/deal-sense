package telegram

// User-facing bot messages live here so they can be reviewed and translated
// without grepping handler logic. Russian only for now; multilingual support
// is backlog.
const (
	msgAttachFile          = "Пришлите файл тендера ответом на эту команду."
	msgAnalysisErrorPrefix = "❌ Ошибка анализа:"
	msgAttachTemplate      = "Пришлите шаблон коммерческого предложения ответом на эту команду."
	msgGenerationErrPrefix = "❌ Ошибка генерации:"
	msgGenerateCaptionFmt  = "✅ Готово. Mode: %s. Sections: %d."

	// /profile and its wizard. The wizard speaks the user as "Вы" and is
	// kept compact — one line per step plus a short instruction header.
	msgProfileEmpty       = "Профиль компании ещё не заполнен. Отправьте /profile edit чтобы заполнить."
	msgProfileShowFmt     = "Текущий профиль:\n\n%s\n\nКоманды: /profile edit — редактировать, /profile clear — сбросить."
	msgProfileCleared     = "Профиль удалён."
	msgProfileUnknownCmd  = "Неизвестная подкоманда. Доступно: /profile, /profile edit, /profile clear."
	msgProfileLoadError   = "❌ Профиль временно недоступен. Попробуйте позже."
	msgProfileSaveError   = "❌ Не удалось сохранить профиль. Попробуйте позже."
	msgWizardStart        = "Заполним профиль компании. На каждый вопрос отвечайте одним сообщением. Прервать — /cancel.\n\nКак называется ваша компания?"
	msgWizardTeamSize     = "Сколько человек в команде?"
	msgWizardExperience   = "Сколько лет опыта в разработке?"
	msgWizardTechStack    = "Технологический стек (через запятую): Go, React, Postgres…"
	msgWizardCerts        = "Сертификации (через запятую). Если нет — отправьте «-»."
	msgWizardSpecs        = "Специализации (через запятую): backend, мобильная разработка…"
	msgWizardClients      = "Ключевые клиенты или проекты (одной строкой). Если нет — «-»."
	msgWizardExtra        = "Что-то ещё важное про компанию? Если нет — «-»."
	msgWizardConfirmFmt   = "Готово, сохраняю профиль:\n\n%s\n\nЕсли что — /profile edit для повторного заполнения."
	msgWizardCancelled    = "Заполнение профиля отменено."
	msgWizardEmptyProfile = "Профиль получился пустым — заполните хотя бы название компании. Начните заново: /profile edit."

	// Exported strings are used by the cmd-level wiring (auth middleware,
	// runtime adapter). Keeping them all here means translation work
	// touches one file.
	MsgDenied         = "🚫 Доступ запрещён."
	MsgDownloadFailed = "❌ Не удалось скачать файл:"
	MsgFallbackHint   = "Используйте /analyze или /generate ответом на файл."

	// DefaultCompanyFallback is the placeholder profile fed to the LLM when
	// a chat has no per-chat company profile saved. It lives next to the
	// other user-visible strings so all bot semantics stay in one file.
	DefaultCompanyFallback = "Software development company"
)
