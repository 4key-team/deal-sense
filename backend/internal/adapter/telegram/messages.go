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
	MsgFallbackHint   = "Не понимаю эту команду. Наберите /help чтобы увидеть все команды."

	// MsgStart greets the user. The shape mirrors MsgHelp's command list
	// so the very first interaction with the bot shows what's possible.
	MsgStart = "👋 Deal Sense — AI-помощник по тендерам.\n\n" +
		"📋 /analyze — анализ тендера (PDF, DOCX, DOC, MD или ZIP-архив)\n" +
		"📝 /generate — генерация КП по шаблону (DOCX, MD или ZIP)\n" +
		"👤 /profile — профиль вашей компании (контекст для LLM)\n" +
		"❓ /help — все команды\n\n" +
		"Чтобы начать: пришлите команду, затем файл — или файл с подписью команды."

	// MsgHelp is the full command reference. Should match the actual
	// registered handlers; update both when wiring changes.
	MsgHelp = "📚 Команды бота:\n\n" +
		"📋 /analyze — анализ тендера.\n" +
		"   Форматы: PDF, DOCX, DOC (Word 97-2003), MD, ZIP-архив.\n" +
		"   Пришлите команду, затем файл отдельным сообщением.\n" +
		"   Или: файл с подписью /analyze в одном сообщении.\n\n" +
		"📝 /generate — генерация КП по шаблону.\n" +
		"   Форматы шаблона: DOCX (плейсхолдеры или генеративный),\n" +
		"   MD (smart+plain), ZIP с шаблоном внутри.\n" +
		"   Пришлите команду, затем шаблон отдельным сообщением.\n" +
		"   Или: шаблон с подписью /generate в одном сообщении.\n\n" +
		"👤 /profile — текущий профиль компании.\n" +
		"   /profile edit — заполнить / обновить.\n" +
		"   /profile clear — удалить.\n\n" +
		"❌ /cancel — прервать заполнение профиля.\n" +
		"❓ /help — это сообщение."

	// DefaultCompanyFallback is the placeholder profile fed to the LLM when
	// a chat has no per-chat company profile saved. It lives next to the
	// other user-visible strings so all bot semantics stay in one file.
	DefaultCompanyFallback = "Software development company"

	// Reply-keyboard button labels. Tapping a button sends its text as a
	// regular message; the bot routes these via MatchTypeExact aliases.
	ButtonAnalyze  = "📋 Анализ тендера"
	ButtonGenerate = "📝 Создать КП"
	ButtonProfile  = "👤 Профиль компании"
	ButtonHelp     = "❓ Помощь"
)
