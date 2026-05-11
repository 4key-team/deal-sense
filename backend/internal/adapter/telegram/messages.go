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

	// Exported strings are used by the cmd-level wiring (auth middleware,
	// runtime adapter). Keeping them all here means translation work
	// touches one file.
	MsgDenied         = "🚫 Доступ запрещён."
	MsgDownloadFailed = "❌ Не удалось скачать файл:"
	MsgFallbackHint   = "Используйте /analyze или /generate ответом на файл."
)
