import { useEffect, useState, type FormEvent } from "react";
import { Button } from "../../../ui/Button";
import {
  useBotConfig,
  type BotConfigUpdateInput,
} from "./useBotConfig";
import { useLLMInfo } from "./useLLMInfo";
import styles from "./BotSettings.module.css";

const LOG_LEVELS = ["debug", "info", "warn", "error"] as const;

// parseAllowlist accepts a CSV / whitespace-separated list and returns a
// deduplicated array of positive integers. Invalid entries surface as a
// thrown error so the caller can highlight the field.
function parseAllowlist(raw: string): number[] {
  const tokens = raw
    .split(/[\s,]+/)
    .map((t) => t.trim())
    .filter(Boolean);
  const ids: number[] = [];
  for (const t of tokens) {
    if (!/^\d+$/.test(t)) {
      throw new Error(`"${t}" is not a positive integer`);
    }
    ids.push(Number(t));
  }
  return ids;
}

export function BotSettings() {
  const { data, loading, error, validation, saving, update } = useBotConfig();
  const llmInfo = useLLMInfo();

  const [token, setToken] = useState("");
  const [allowlistRaw, setAllowlistRaw] = useState("");
  const [logLevel, setLogLevel] = useState("info");
  const [showToken, setShowToken] = useState(false);
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);

  // When the backend returns existing config, seed allowlist + log level so
  // the operator sees current values and can edit incrementally. The token
  // is intentionally NOT seeded: backend never returns the raw secret, only
  // the mask — an empty input means "keep current".
  useEffect(() => {
    if (!data) return;
    setAllowlistRaw((data.allowlist_user_ids ?? []).join(", "));
    setLogLevel(data.log_level ?? "info");
  }, [data]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLocalError(null);
    setSavedAt(null);

    let ids: number[];
    try {
      ids = parseAllowlist(allowlistRaw);
    } catch (err) {
      setLocalError((err as Error).message);
      return;
    }

    const payload: BotConfigUpdateInput = {
      token: token.trim(),
      allowlist_user_ids: ids,
      log_level: logLevel,
    };

    const ok = await update(payload);
    if (ok) {
      setToken("");
      setShowToken(false);
      setSavedAt(Date.now());
    }
  }

  const tokenFieldError = validation?.field === "token" ? validation.error : null;
  const allowlistFieldError =
    validation?.field === "allowlist_user_ids" ? validation.error : localError;
  const logLevelFieldError =
    validation?.field === "log_level" ? validation.error : null;
  const genericError =
    error ?? (validation && !validation.field ? validation.error : null);

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Bot configuration</h1>
      <p className={styles.subtitle}>
        Telegram bot token, access allowlist and log level. Changes take effect
        after restarting the <code>telegram-bot</code> container.
      </p>

      <div className={styles.policyCard} role="note" aria-label="LLM policy">
        <div className={styles.policyHeader}>
          <span className={styles.policyBadge}>BYOK</span>
          <strong>Per-chat LLM is required by default</strong>
        </div>
        <p className={styles.policyBody}>
          Since v0.23 the bot enforces «bring your own key»: every Telegram
          chat must run <code>/llm edit</code> in the bot before{" "}
          <code>/analyze</code> or <code>/generate</code> will work. The
          server-side env <code>LLM_*</code> keys are no longer a silent
          fallback for arbitrary users.
        </p>
        <p className={styles.policyBody}>
          To revert to single-tenant behaviour (env <code>LLM_*</code> shared
          by all chats), set{" "}
          <code>ALLOW_SERVER_LLM_FALLBACK=true</code> on the{" "}
          <code>telegram-bot</code> service and restart it.
        </p>
      </div>

      {loading && <p className={styles.loading}>Loading current configuration…</p>}

      {genericError && (
        <div className={`${styles.banner} ${styles.bannerError}`} role="alert">
          ❌ {genericError}
        </div>
      )}

      {savedAt !== null && (
        <div className={`${styles.banner} ${styles.bannerSuccess}`} role="status">
          ✅ Saved. Restart the <code>telegram-bot</code> container for the
          changes to take effect (<code>docker compose restart telegram-bot</code>).
        </div>
      )}

      {data && data.configured && data.masked_token && (
        <p className={styles.hint}>
          Currently saved token: <code>{data.masked_token}</code>. Leave the
          token field empty to keep it; type a new value to replace it.
        </p>
      )}

      <form className={styles.form} onSubmit={onSubmit} noValidate>
        <div
          className={`${styles.field} ${tokenFieldError ? styles.fieldError : ""}`}
        >
          <label className={styles.label} htmlFor="bot-token">
            Telegram bot token
            <span
              className={styles.tooltip}
              title="Format: <digits>:<secret>. Get it from @BotFather in Telegram."
            >
              ?
            </span>
          </label>
          <div className={styles.passwordRow}>
            <input
              id="bot-token"
              className={styles.input}
              type={showToken ? "text" : "password"}
              autoComplete="off"
              spellCheck={false}
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder={data?.configured ? "(keep current)" : "1234567:ABC-DEF…"}
              style={{ flex: 1 }}
            />
            <button
              type="button"
              className={styles.toggleVisibility}
              onClick={() => setShowToken((v) => !v)}
              aria-label={showToken ? "Hide token" : "Show token"}
            >
              {showToken ? "Hide" : "Show"}
            </button>
          </div>
          {tokenFieldError && (
            <span className={styles.fieldErrorText} role="alert">
              {tokenFieldError}
            </span>
          )}
        </div>

        <div
          className={`${styles.field} ${allowlistFieldError ? styles.fieldError : ""}`}
        >
          <label className={styles.label} htmlFor="bot-allowlist">
            Allowlist (Telegram user IDs)
            <span
              className={styles.tooltip}
              title="Comma or space separated positive integers. Leave empty for OPEN mode — any Telegram user can interact with the bot."
            >
              ?
            </span>
          </label>
          <textarea
            id="bot-allowlist"
            className={styles.textarea}
            value={allowlistRaw}
            onChange={(e) => setAllowlistRaw(e.target.value)}
            placeholder="e.g. 12345678, 98765432"
            aria-describedby="bot-allowlist-hint"
          />
          <p id="bot-allowlist-hint" className={styles.hint}>
            Empty list = <strong>open mode</strong> (any Telegram user can talk
            to the bot). For production, restrict to known IDs.
          </p>
          {allowlistFieldError && (
            <span className={styles.fieldErrorText} role="alert">
              {allowlistFieldError}
            </span>
          )}
        </div>

        <div
          className={`${styles.field} ${logLevelFieldError ? styles.fieldError : ""}`}
        >
          <label className={styles.label} htmlFor="bot-loglevel">
            Log level
            <span
              className={styles.tooltip}
              title="Use 'info' in production. 'debug' is verbose and may leak data into logs."
            >
              ?
            </span>
          </label>
          <select
            id="bot-loglevel"
            className={styles.select}
            value={logLevel}
            onChange={(e) => setLogLevel(e.target.value)}
          >
            {LOG_LEVELS.map((lvl) => (
              <option key={lvl} value={lvl}>
                {lvl}
              </option>
            ))}
          </select>
          {logLevelFieldError && (
            <span className={styles.fieldErrorText} role="alert">
              {logLevelFieldError}
            </span>
          )}
        </div>

        <div className={styles.actions}>
          <Button type="submit" disabled={saving} variant="primary">
            {saving ? "Saving…" : "Save configuration"}
          </Button>
        </div>
      </form>

      <section className={styles.llmSection} aria-label="Server-side LLM">
        <h2 className={styles.llmTitle}>Server-side LLM (read-only)</h2>
        <p className={styles.subtitle}>
          This is the LLM the backend uses when a chat has not configured its
          own via the <code>/llm</code> wizard in Telegram. To change these,
          edit the <code>LLM_*</code> env vars on the <code>backend</code>{" "}
          service and restart it.
        </p>
        {llmInfo.error && (
          <div className={`${styles.banner} ${styles.bannerError}`} role="alert">
            ❌ {llmInfo.error}
          </div>
        )}
        <dl className={styles.llmList}>
          <div className={styles.llmRow}>
            <dt>Active providers</dt>
            <dd>
              {llmInfo.providers.length > 0
                ? llmInfo.providers.join(", ")
                : <span className={styles.empty}>loading…</span>}
            </dd>
          </div>
          <div className={styles.llmRow}>
            <dt>Available models</dt>
            <dd>
              {llmInfo.models.length > 0 ? (
                <ul className={styles.modelList}>
                  {llmInfo.models.map((m: string) => (
                    <li key={m}>
                      <code>{m}</code>
                    </li>
                  ))}
                </ul>
              ) : (
                <span className={styles.empty}>loading…</span>
              )}
            </dd>
          </div>
        </dl>
        <p className={styles.hint}>
          To override per chat, the user sends <code>/llm edit</code> in the
          Telegram bot and walks through the 4-step wizard. Per-chat settings
          are stored in <code>/data/telegram-llm-settings.json</code> and
          forwarded to the backend as <code>X-LLM-*</code> headers on each
          analyze / generate call.
        </p>
      </section>
    </div>
  );
}
