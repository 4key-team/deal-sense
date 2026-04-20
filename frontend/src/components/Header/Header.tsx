import { useState, useRef, useEffect } from "react";
import { useI18n } from "../../providers/useI18n";
import { useTheme } from "../../providers/useTheme";
import { SunIcon, MoonIcon } from "../../icons/Icons";
import { Lockup } from "../Logo";
import { StatusPill } from "../../ui/StatusPill";
import { getItem } from "../../lib/storage";
import type { Lang } from "../../i18n/types";
import styles from "./Header.module.css";

type LogoVariant = "seal" | "wax" | "plumb" | "prism";

const LOGO_VARIANTS: LogoVariant[] = ["seal", "wax", "plumb", "prism"];

const VARIANT_LABELS: Record<LogoVariant, { ru: string; en: string }> = {
  seal:  { ru: "Печать", en: "Seal" },
  wax:   { ru: "Медаль", en: "Wax" },
  plumb: { ru: "Отвес",  en: "Plumb" },
  prism: { ru: "Призма", en: "Prism" },
};

function getStoredVariant(): LogoVariant {
  const v = localStorage.getItem("ds:logo");
  if (v === "seal" || v === "wax" || v === "plumb" || v === "prism") return v;
  return "seal";
}

export interface HeaderProps {
  onOpenSettings: () => void;
  settingsVersion?: number;
}

const LLM_DEFAULTS = { providerId: "anthropic", apiKey: "", model: "claude-sonnet-4-5" };

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function Header({ onOpenSettings, settingsVersion: _v }: HeaderProps) {
  const { lang, setLang } = useI18n();
  const { theme, toggleTheme } = useTheme();
  const llmSettings = getItem<typeof LLM_DEFAULTS>("llm-settings", LLM_DEFAULTS);

  const [logoVariant, setLogoVariant] = useState<LogoVariant>(getStoredVariant);
  const [dropdownOpen, setDropdownOpen] = useState(false);

  const dropdownRef = useRef<HTMLDivElement>(null);
  const logoBtnRef = useRef<HTMLButtonElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    if (!dropdownOpen) return;
    function handleClick(e: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        logoBtnRef.current &&
        !logoBtnRef.current.contains(e.target as Node)
      ) {
        setDropdownOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [dropdownOpen]);

  function selectVariant(v: LogoVariant) {
    setLogoVariant(v);
    localStorage.setItem("ds:logo", v);
    setDropdownOpen(false);
  }

  return (
    <header className={styles.header}>
      <div className={styles.inner}>
        {/* Logo button with dropdown */}
        <div style={{ position: "relative" }}>
          <button
            ref={logoBtnRef}
            className={styles.logoBtn}
            onClick={() => setDropdownOpen((v) => !v)}
            aria-label="Choose logo variant"
            aria-expanded={dropdownOpen}
            aria-haspopup="true"
          >
            <Lockup variant={logoVariant} size={18} />
          </button>

          {dropdownOpen && (
            <div ref={dropdownRef} className={styles.dropdown} role="menu">
              {LOGO_VARIANTS.map((v) => (
                <button
                  key={v}
                  className={
                    v === logoVariant
                      ? `${styles.variantBtn} ${styles.variantBtnActive}`
                      : styles.variantBtn
                  }
                  onClick={() => selectVariant(v)}
                  role="menuitem"
                  aria-pressed={v === logoVariant}
                >
                  <Lockup variant={v} size={20} />
                  <span className={styles.variantLabel}>
                    {VARIANT_LABELS[v][lang]}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className={styles.spacer} />

        {/* Theme toggle */}
        <button
          className={styles.themeBtn}
          onClick={toggleTheme}
          aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
        >
          {theme === "dark" ? <SunIcon /> : <MoonIcon />}
        </button>

        {/* Language switcher */}
        <div className={styles.langSwitch} role="group" aria-label="Language">
          {(["ru", "en"] as Lang[]).map((l) => (
            <button
              key={l}
              className={
                l === lang
                  ? `${styles.langBtn} ${styles.langBtnActive}`
                  : styles.langBtn
              }
              onClick={() => setLang(l)}
              aria-pressed={l === lang}
            >
              {l}
            </button>
          ))}
        </div>

        {/* Status pill — opens settings */}
        <StatusPill
          provider={llmSettings.providerId}
          model={llmSettings.model}
          status={llmSettings.apiKey ? "ok" : "err"}
          onClick={onOpenSettings}
        />
      </div>
    </header>
  );
}
