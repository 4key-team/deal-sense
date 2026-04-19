import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import type { Copy, Lang } from "../i18n/types";
import { ru } from "../i18n/ru";
import { en } from "../i18n/en";

const copies: Record<Lang, Copy> = { ru, en };

interface I18nContextValue {
  lang: Lang;
  setLang: (lang: Lang) => void;
  t: Copy;
}

const I18nContext = createContext<I18nContextValue | null>(null);

function getInitialLang(): Lang {
  const saved = localStorage.getItem("ds:lang");
  if (saved === "ru" || saved === "en") return saved;
  return "ru";
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(getInitialLang);

  useEffect(() => {
    localStorage.setItem("ds:lang", lang);
  }, [lang]);

  const setLang = useCallback((l: Lang) => setLangState(l), []);

  return (
    <I18nContext.Provider value={{ lang, setLang, t: copies[lang] }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
