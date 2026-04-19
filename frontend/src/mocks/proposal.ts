import type { Lang } from "../i18n/types";

export interface ProposalSection {
  id: number;
  title: string;
  status: "ai" | "review" | "filled";
  tokens: number;
}

export interface ProposalContext {
  kind: "tpl" | "ctx";
  name: string;
  size: string;
  role: string;
}

export interface ProposalMeta {
  client: string;
  project: string;
  price: string;
  term: string;
  created: string;
}

export interface ProposalLog {
  time: string;
  msg: string;
}

export function getSections(lang: Lang): ProposalSection[] {
  return [
    { id: 1, title: lang === "ru" ? "Резюме" : "Executive summary", status: "ai", tokens: 412 },
    { id: 2, title: lang === "ru" ? "Понимание задачи" : "Problem understanding", status: "ai", tokens: 684 },
    { id: 3, title: lang === "ru" ? "Подход и методология" : "Approach & methodology", status: "ai", tokens: 1120 },
    { id: 4, title: lang === "ru" ? "Состав команды" : "Team composition", status: "filled", tokens: 0 },
    { id: 5, title: lang === "ru" ? "Этапы и сроки" : "Phases & timeline", status: "ai", tokens: 540 },
    { id: 6, title: lang === "ru" ? "Стоимость" : "Pricing", status: "filled", tokens: 0 },
    { id: 7, title: lang === "ru" ? "Релевантные кейсы" : "Relevant case studies", status: "review", tokens: 820 },
    { id: 8, title: lang === "ru" ? "Условия работы" : "Terms & conditions", status: "filled", tokens: 0 },
  ];
}

export function getContext(
  lang: Lang,
  t: { context_brief: string; context_cases: string; context_prices: string },
): ProposalContext[] {
  return [
    { kind: "tpl", name: "proposal-tpl.docx", size: "24 KB", role: lang === "ru" ? "шаблон" : "template" },
    { kind: "ctx", name: lang === "ru" ? "brief-klient.txt" : "client-brief.txt", size: "3.1 KB", role: t.context_brief },
    { kind: "ctx", name: "cases-2025.docx", size: "142 KB", role: t.context_cases },
    { kind: "ctx", name: "rates-q2.docx", size: "18 KB", role: t.context_prices },
  ];
}

export function getMeta(lang: Lang): ProposalMeta {
  return {
    client: "Northwind Logistics",
    project: lang === "ru" ? "Портал для партнёров, 1 этап" : "Partner portal, phase 1",
    price: lang === "ru" ? "2 450 000 ₽" : "€24,500",
    term: lang === "ru" ? "9 недель" : "9 weeks",
    created: lang === "ru" ? "сегодня, 14:32" : "Today, 2:32 PM",
  };
}

export function getLog(lang: Lang): ProposalLog[] {
  return [
    { time: "14:31:04", msg: lang === "ru" ? "прочитан шаблон · 42 плейсхолдера" : "template parsed · 42 placeholders" },
    { time: "14:31:06", msg: lang === "ru" ? "индексирован контекст · 3 файла" : "context indexed · 3 files" },
    { time: "14:31:18", msg: lang === "ru" ? "заполнены статичные поля · 12" : "static fields filled · 12" },
    { time: "14:32:01", msg: lang === "ru" ? "сгенерированы секции · 5 из 8" : "sections generated · 5 of 8" },
    { time: "14:32:08", msg: lang === "ru" ? "собран .docx · 18 страниц" : ".docx assembled · 18 pages" },
  ];
}
