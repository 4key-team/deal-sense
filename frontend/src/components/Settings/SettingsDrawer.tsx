import { useState } from "react";
import { useI18n } from "../../providers/useI18n";
import { Button } from "../../ui/Button";
import { Select } from "../../ui/Select";
import { Field } from "../../ui/Field";
import { Spinner } from "../../ui/Spinner";
import { XIcon, EyeIcon, CheckIcon } from "../../icons/Icons";
import { getItem, setItem } from "../../lib/storage";
import { checkConnection } from "../../lib/api";
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
    models: ["llama-3.3-70b", "mixtral-8x7b"],
    url: "https://api.groq.com/openai/v1",
  },
  {
    id: "ollama",
    label: () => "Ollama (local)",
    models: ["llama3.1:70b", "qwen2.5:32b"],
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
}

export function SettingsDrawer({ open, onClose }: SettingsDrawerProps) {
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

  if (!open) return null;

  const currentProvider = PROVIDERS.find((p) => p.id === providerId) ?? defaultProvider;

  function handleProviderPick(id: string) {
    const p = PROVIDERS.find((pr) => pr.id === id);
    if (!p) return;
    setProviderId(id);
    setUrl(p.url);
    setModel(p.models[0]);
    setTestState("idle");
    setProviderOpen(false);
  }

  function handleModelPick(id: string) {
    setModel(id);
    setModelOpen(false);
  }

  async function handleTest() {
    setTestState("testing");
    try {
      const result = await checkConnection();
      setTestState(result.ok ? "ok" : "fail");
    } catch {
      setTestState("fail");
    }
  }

  function handleSave() {
    setItem<LLMSettings>(STORAGE_KEY, { providerId, apiKey, url, model });
    onClose();
  }

  const providerOptions = PROVIDERS.map((p) => ({
    id: p.id,
    label: p.label(lang),
  }));

  const modelOptions = currentProvider.models.map((m) => ({
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
          <Field label={t.settings.provider}>
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
          <Field label={t.settings.key} hint={t.settings.key_hint}>
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
          <Field label={t.settings.url} hint={t.settings.url_hint}>
            <input
              className={styles.urlInput}
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              spellCheck={false}
            />
          </Field>

          {/* Model */}
          <Field label={t.settings.model}>
            <Select
              value={model}
              options={modelOptions}
              selected={model}
              open={modelOpen}
              onOpen={() => setModelOpen(true)}
              onClose={() => setModelOpen(false)}
              onPick={handleModelPick}
              mono
            />
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
