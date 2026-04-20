import { useState } from "react";
import { useI18n } from "../../providers/useI18n";
import { Button } from "../../ui/Button";
import { Select } from "../../ui/Select";
import { Field } from "../../ui/Field";
import { Spinner } from "../../ui/Spinner";
import { XIcon, EyeIcon, CheckIcon, ChevIcon } from "../../icons/Icons";
import { getItem, setItem } from "../../lib/storage";
import { checkConnection, listModels } from "../../lib/api";
import styles from "./SettingsDrawer.module.css";

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

export interface LLMSettings {
  providerId: string;
  apiKey: string;
  url: string;
  model: string;
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

export interface SettingsDrawerProps {
  open: boolean;
  onClose: () => void;
  onSave?: () => void;
}

export function SettingsDrawer({ open, onClose, onSave }: SettingsDrawerProps) {
  const { lang, t } = useI18n();

  const defaultProvider = PROVIDERS[0];
  const saved = loadSettings();

  const [providerId, setProviderId] = useState<string>(saved.providerId);
  const [apiKey, setApiKey] = useState<string>(saved.apiKey);
  const [showKey, setShowKey] = useState<boolean>(false);
  const [url, setUrl] = useState<string>(saved.url);
  const [model, setModel] = useState<string>(saved.model);
  const [testState, setTestState] = useState<TestState>("idle");
  const [providerOpen, setProviderOpen] = useState<boolean>(false);
  const [modelOpen, setModelOpen] = useState<boolean>(false);
  const [testError, setTestError] = useState("");
  const [remoteModels, setRemoteModels] = useState<string[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);

  if (!open) return null;

  const currentProvider = PROVIDERS.find((p) => p.id === providerId) ?? defaultProvider;

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
    fetchModels(id, p.url);
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
    setItem<LLMSettings>(STORAGE_KEY, { providerId, apiKey, url, model });
    onSave?.();
    onClose();
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
    <>
      <div
        className={styles.backdrop}
        onClick={onClose}
        aria-hidden="true"
      />
      <div
        className={styles.drawer}
        role="dialog"
        aria-modal="true"
        aria-label={t.settings.title}
      >
        {/* Header */}
        <div className={styles.header}>
          <div className={styles.headerText}>
            <div className={`${styles.headerTitle} font-serif`}>{t.settings.title}</div>
            <div className={`t-small muted ${styles.headerSubtitle}`}>{t.settings.subtitle}</div>
          </div>
          <button
            className={styles.closeBtn}
            onClick={onClose}
            type="button"
            aria-label="Close"
          >
            <XIcon />
          </button>
        </div>

        {/* Body */}
        <div className={styles.body}>
          {/* Provider */}
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

          {/* API Key */}
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

          {/* API URL */}
          <Field label={t.settings.url} hint={t.settings.url_hint} tooltip={t.settings.url_tip}>
            <input
              className={styles.urlInput}
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              spellCheck={false}
            />
          </Field>

          {/* Model */}
          <Field label={t.settings.model} hint={t.settings.model_hint} tooltip={t.settings.model_tip}>
            <div className={styles.modelWrap}>
              <input
                className={`${styles.keyInput} ${styles.modelInput}`}
                type="text"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                spellCheck={false}
                placeholder="e.g. llama-3.3-70b-versatile"
              />
              <button
                type="button"
                className={styles.modelPickBtn}
                onClick={() => {
                  const opening = !modelOpen;
                  setModelOpen(opening);
                  if (opening && remoteModels.length === 0) {
                    fetchModels(providerId, url);
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

          {/* Test connection */}
          <Button
            variant="secondary"
            onClick={handleTest}
            disabled={testState === "testing"}
            icon={
              testState === "testing" ? (
                <Spinner />
              ) : testState === "ok" ? (
                <span className={styles.iconGo}><CheckIcon /></span>
              ) : testState === "fail" ? (
                <span className={styles.iconNo}><XIcon /></span>
              ) : undefined
            }
          >
            {testState === "idle" && t.settings.test}
            {testState === "testing" && t.settings.testing}
            {testState === "ok" && t.settings.test_ok}
            {testState === "fail" && t.settings.test_fail}
          </Button>
          {testError && (
            <p className={`t-small ${styles.testError}`}>{testError}</p>
          )}
        </div>

        {/* Footer */}
        <div className={styles.footer}>
          <Button variant="ghost" onClick={onClose} type="button">
            {t.settings.cancel}
          </Button>
          <Button variant="primary" type="button" onClick={handleSave}>
            {t.settings.save}
          </Button>
        </div>
      </div>
    </>
  );
}
