package telegram

// User-facing bot messages live here so they can be reviewed and translated
// without grepping handler logic. Russian only for now; English is Session 2.5+.
const (
	msgAttachFile          = "Пришлите файл тендера ответом на эту команду."
	msgAnalysisErrorPrefix = "❌ Ошибка анализа:"

	// Exported strings are used by the cmd-level wiring (auth middleware,
	// runtime adapter). Keeping them all here means translation work
	// touches one file.
	MsgDenied              = "🚫 Доступ запрещён."
	MsgDownloadFailed      = "❌ Не удалось скачать файл:"
	MsgFallbackHint        = "Используйте /analyze ответом на файл тендера."
)
