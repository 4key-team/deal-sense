import { createContext } from "react";
import type { Copy, Lang } from "../i18n/types";

export interface I18nContextValue {
  lang: Lang;
  setLang: (lang: Lang) => void;
  t: Copy;
}

export const I18nContext = createContext<I18nContextValue | null>(null);
