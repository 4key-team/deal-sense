import { useCallback, useEffect, useState, type ReactNode } from "react";
import type { Lang } from "../i18n/types";
import { ru } from "../i18n/ru";
import { en } from "../i18n/en";
import { I18nContext } from "./I18nContext";
import type { Copy } from "../i18n/types";

const copies: Record<Lang, Copy> = { ru, en };

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
