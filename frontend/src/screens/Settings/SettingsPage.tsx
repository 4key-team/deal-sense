import { useState, useEffect } from "react";
import { useI18n } from "../../providers/useI18n";
import { Button } from "../../ui/Button";
import { Select } from "../../ui/Select";
import { Field } from "../../ui/Field";
import { Spinner } from "../../ui/Spinner";
import { XIcon, EyeIcon, CheckIcon, ChevIcon } from "../../icons/Icons";
import { getItem, setItem } from "../../lib/storage";
import { checkConnection, listModels } from "../../lib/api";
import styles from "./SettingsPage.module.css";

interface Provider {
  id: string;
  label: (lang: string) => string;
  models: string[];
  url: string;
}

const PROVIDERS: Provider[] = [
  {
    id: "anthropic",
    label: () => "Anthropic",
    models: ["claude-haiku-4-5", "claude-sonnet-4-5", "claude-opus-4-1"],
    url: "https://api.anthropic.com/v1",
  },
  {
    id: "openai",
    label: () => "OpenAI",
    models: ["gpt-4o", "gpt-4o-mini", "o3-mini"],
    url: "https://api.openai.com/v1",
  },
  {
    id: "google",
    label: () => "Google Gemini",
    models: ["gemini-2.5-pro", "gemini-2.5-flash"],
    url: "https://generativelanguage.googleapis.com/v1beta",
  },
  {
    id: "groq",
    label: () => "Groq",
    models: ["llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"],
    url: "https://api.groq.com/openai/v1",
  },
  {
    id: "openrouter",
    label: () => "OpenRouter",
    models: ["anthropic/claude-sonnet-4", "anthropic/claude-opus-4", "openai/gpt-4o"],
    url: "https://openrouter.ai/api/v1",
  },
  {
    id: "ollama",
    label: () => "Ollama (local)",
    models: ["llama3.1:70b", "qwen2.5:32b", "mistral:7b"],
    url: "http://localhost:11434/v1",
  },
  {
    id: "custom",
    label: (lang: string) => (lang === "ru" ? "Свой endpoint" : "Custom"),
    models: ["your-model"],
    url: "",
  },
];

type TestState = "idle" | "testing" | "ok" | "fail";

interface LLMSettings {
  providerId: string;
  apiKey: string;
  url: string;
  model: string;
  // dealSenseApiKey is the backend's X-API-Key. Admin-level: backend
  // sets DEAL_SENSE_API_KEY env on deploy, operator pastes the same
  // value here. Required for any /api/* call (analyze, generate,
  // check, list providers/models) when the backend has auth on.
  dealSenseApiKey?: string;
}

const STORAGE_KEY = "llm-settings";

function loadSettings(): LLMSettings {
  const defaultProvider = PROVIDERS[0];
  return getItem<LLMSettings>(STORAGE_KEY, {
    providerId: defaultProvider.id,
    apiKey: "",
    url: defaultProvider.url,
    model: defaultProvider.models[0],
  });
}

// SettingsPage is the unified user-facing settings screen accessible at
// /settings. It hosts everything the user can configure for themselves:
// LLM credentials saved to localStorage (used by the web frontend's own
// /api/llm/* calls) and a read-only view of the server-side default
// LLM. Operator-only knobs (bot token, allowlist) live on a separate
// /admin/* route since they need backend X-API-Key.
export function SettingsPage() {
  const { lang, t } = useI18n();

  const defaultProvider = PROVIDERS[0];
  const saved = loadSettings();

  const [providerId, setProviderId] = useState<string>(saved.providerId);
  const [apiKey, setApiKey] = useState<string>(saved.apiKey);
  const [showKey, setShowKey] = useState<boolean>(false);
  const [url, setUrl] = useState<string>(saved.url);
  const [model, setModel] = useState<string>(saved.model);
  const [dealSenseApiKey, setDealSenseApiKey] = useState<string>(saved.dealSenseApiKey ?? "");
  const [showBackendKey, setShowBackendKey] = useState<boolean>(false);
  const [testState, setTestState] = useState<TestState>("idle");
  const [providerOpen, setProviderOpen] = useState<boolean>(false);
  const [modelOpen, setModelOpen] = useState<boolean>(false);
  const [testError, setTestError] = useState("");
  const [remoteModels, setRemoteModels] = useState<string[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  const currentProvider =
    PROVIDERS.find((p) => p.id === providerId) ?? defaultProvider;

  // Fetch the remote model list once on mount when an api key already exists.
  useEffect(() => {
    if (!apiKey) return;
    void fetchModels(providerId, url);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Any change to the credentials invalidates the previous test result
  // and the saved confirmation. UX intent: a stale ✓ next to changed
  // fields would lie about the current state, and the «Сохранено»
  // confirmation must not survive an edit that hasn't been re-saved.
  useEffect(() => {
    setTestState("idle");
    setTestError("");
    setSavedAt(null);
  }, [providerId, apiKey, url, model, dealSenseApiKey]);

  async function fetchModels(provId: string, provUrl: string) {
    if (!apiKey) return;
    setLoadingModels(true);
    try {
      const result = await listModels(provId, apiKey, provUrl);
      if (result.models.length > 0) {
        setRemoteModels(result.models);
      }
    } catch {
      // keep local list
    } finally {
      setLoadingModels(false);
    }
  }

  function handleProviderPick(id: string) {
    const p = PROVIDERS.find((pr) => pr.id === id);
    if (!p) return;
    setProviderId(id);
    setUrl(p.url);
    setModel(p.models[0]);
    setTestState("idle");
    setProviderOpen(false);
    setRemoteModels([]);
    void fetchModels(id, p.url);
  }

  function handleModelPick(id: string) {
    setModel(id);
    setModelOpen(false);
  }

  async function handleTest() {
    setTestState("testing");
    setTestError("");
    try {
      const result = await checkConnection({
        provider: providerId,
        apiKey,
        url,
        model,
      });
      setTestState(result.ok ? "ok" : "fail");
      if (!result.ok && result.error) {
        setTestError(result.error);
      }
    } catch (err) {
      setTestState("fail");
      setTestError(err instanceof Error ? err.message : String(err));
    }
  }

  function handleSave() {
    setItem<LLMSettings>(STORAGE_KEY, {
      providerId,
      apiKey,
      url,
      model,
      dealSenseApiKey: dealSenseApiKey.trim() || undefined,
    });
    setSavedAt(Date.now());
  }

  const providerOptions = PROVIDERS.map((p) => ({
    id: p.id,
    label: p.label(lang),
  }));

  const modelList = remoteModels.length > 0 ? remoteModels : currentProvider.models;
  const modelOptions = modelList.map((m) => ({
    id: m,
    label: m,
    mono: true,
  }));

  const providerLabel =
    PROVIDERS.find((p) => p.id === providerId)?.label(lang) ?? providerId;

  return (
    <div className={styles.page}>
      <div className={styles.head}>
        <h1 className={styles.title}>{t.settings.title}</h1>
        <p className={styles.subtitle}>{t.settings.subtitle}</p>
      </div>

      {savedAt !== null && (
        <div className={styles.savedBanner} role="status">
          ✅ {lang === "ru" ? "Сохранено" : "Saved"}
        </div>
      )}

      <section className={styles.card} aria-label="LLM credentials">
        <h2 className={styles.cardTitle}>
          {lang === "ru" ? "Ваш LLM ключ" : "Your LLM key"}
        </h2>
        <p className={styles.cardHint}>
          {lang === "ru"
            ? "Эти настройки используются для веб-вызовов из этого браузера. Хранятся в localStorage. Для Telegram-бота настройте /llm edit в чате."
            : "Used by web calls from this browser. Stored in localStorage. For the Telegram bot, run /llm edit there."}
        </p>

        <Field label={t.settings.provider} tooltip={t.settings.provider_tip}>
          <Select
            value={providerLabel}
            options={providerOptions}
            selected={providerId}
            open={providerOpen}
            onOpen={() => setProviderOpen(true)}
            onClose={() => setProviderOpen(false)}
            onPick={handleProviderPick}
          />
        </Field>

        <Field label={t.settings.key} hint={t.settings.key_hint} tooltip={t.settings.key_tip}>
          <div className={styles.keyWrap}>
            <input
              className={styles.keyInput}
              type={showKey ? "text" : "password"}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              autoComplete="off"
              spellCheck={false}
            />
            <button
              className={styles.keyToggle}
              type="button"
              onClick={() => setShowKey((v) => !v)}
              aria-label={showKey ? t.settings.hide : t.settings.show}
            >
              <EyeIcon off={showKey} />
            </button>
          </div>
        </Field>

        <Field label={t.settings.url} hint={t.settings.url_hint} tooltip={t.settings.url_tip}>
          <input
            className={styles.urlInput}
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            spellCheck={false}
          />
        </Field>

        <Field label={t.settings.model} hint={t.settings.model_hint} tooltip={t.settings.model_tip}>
          <div className={styles.modelWrap}>
            <input
              className={`${styles.keyInput} ${styles.modelInput}`}
              type="text"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              spellCheck={false}
              placeholder="e.g. claude-sonnet-4-5"
            />
            <button
              type="button"
              className={styles.modelPickBtn}
              onClick={() => {
                const opening = !modelOpen;
                setModelOpen(opening);
                if (opening && remoteModels.length === 0) {
                  void fetchModels(providerId, url);
                }
              }}
              aria-label="Pick model"
            >
              {loadingModels ? <Spinner /> : <ChevIcon dir={modelOpen ? "up" : "down"} />}
            </button>
          </div>
          {modelOpen && (
            <div className={styles.modelDropdown}>
              {modelOptions.map((o) => (
                <button
                  key={o.id}
                  type="button"
                  className={`${styles.modelOption} ${o.id === model ? styles.modelOptionActive : ""}`}
                  onClick={() => handleModelPick(o.id)}
                >
                  {o.label}
                </button>
              ))}
            </div>
          )}
        </Field>

        <div className={styles.actions}>
          <Button
            variant="secondary"
            onClick={handleTest}
            disabled={testState === "testing"}
            icon={
              testState === "testing" ? (
                <Spinner />
              ) : testState === "ok" ? (
                <span className={styles.iconGo}>
                  <CheckIcon />
                </span>
              ) : testState === "fail" ? (
                <span className={styles.iconNo}>
                  <XIcon />
                </span>
              ) : undefined
            }
          >
            {testState === "idle" && t.settings.test}
            {testState === "testing" && t.settings.testing}
            {testState === "ok" && t.settings.test_ok}
            {testState === "fail" && t.settings.test_fail}
          </Button>
          <Button
            variant="primary"
            onClick={handleSave}
            icon={savedAt !== null ? <CheckIcon /> : undefined}
          >
            {savedAt !== null
              ? lang === "ru"
                ? "Сохранено"
                : "Saved"
              : t.settings.save}
          </Button>
        </div>
        {testError && <p className={styles.testError}>{testError}</p>}
      </section>

      <section className={styles.card} aria-label="Backend access">
        <h2 className={styles.cardTitle}>
          {lang === "ru" ? "Доступ к серверу Deal Sense" : "Deal Sense backend access"}
        </h2>
        <p className={styles.cardHint}>
          {lang === "ru" ? (
            <>
              Backend защищён <code>X-API-Key</code>. Этот ключ задаёт администратор
              при деплое (<code>DEAL_SENSE_API_KEY</code> в env). Спросите у того, кто
              развернул бэкенд, и вставьте сюда — без него тест подключения и
              запросы к бэкенду возвращают 401 unauthorized.
            </>
          ) : (
            <>
              The backend is protected by <code>X-API-Key</code>. Set on deploy via
              <code> DEAL_SENSE_API_KEY</code>. Ask the operator who deployed the
              backend; without it tests and API calls return 401 unauthorized.
            </>
          )}
        </p>

        <Field
          label={lang === "ru" ? "Backend API ключ" : "Backend API key"}
          hint={lang === "ru" ? "Хранится только в этом браузере." : "Stored in this browser only."}
        >
          <div className={styles.keyWrap}>
            <input
              className={styles.keyInput}
              type={showBackendKey ? "text" : "password"}
              value={dealSenseApiKey}
              onChange={(e) => setDealSenseApiKey(e.target.value)}
              autoComplete="off"
              spellCheck={false}
              placeholder={lang === "ru" ? "dev-smoke-test-key-…" : "dev-smoke-test-key-…"}
            />
            <button
              className={styles.keyToggle}
              type="button"
              onClick={() => setShowBackendKey((v) => !v)}
              aria-label={showBackendKey ? t.settings.hide : t.settings.show}
            >
              <EyeIcon off={showBackendKey} />
            </button>
          </div>
        </Field>
      </section>

      <section className={styles.card} aria-label="Telegram bot">
        <h2 className={styles.cardTitle}>
          {lang === "ru" ? "Telegram бот" : "Telegram bot"}
        </h2>
        <p className={styles.cardHint}>
          {lang === "ru" ? (
            <>
              С v0.23 каждый пользователь бота настраивает свой LLM сам — отправьте
              <code> /llm edit</code> в чате с ботом (4 коротких шага: provider,
              base URL, API key, model). Серверный LLM используется только если
              включен фолбек.
            </>
          ) : (
            <>
              Since v0.23 each Telegram chat configures its own LLM —
              run <code>/llm edit</code> in the bot (4 short steps). The server
              default is only used when the operator enables fallback mode.
            </>
          )}
        </p>
        <p className={styles.cardHint}>
          {lang === "ru" ? "Также в боту доступно: " : "Available in the bot: "}
          <code>/profile edit</code>
          {lang === "ru" ? " (профиль компании), " : " (company profile), "}
          <code>/analyze</code>, <code>/generate</code>
          {lang === "ru"
            ? " (с multi-file сбором контекста и /go для запуска)."
            : " (with multi-file context collection and /go to run)."}
        </p>
      </section>
    </div>
  );
}
