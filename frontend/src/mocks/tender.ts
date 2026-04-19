import type { Lang } from "../i18n/types";

export interface TenderProCon {
  t: string;
  d: string;
}

export interface TenderRequirement {
  label: string;
  status: "met" | "partial" | "miss";
}

export interface TenderFile {
  name: string;
  size: string;
  pages: number;
}

export interface TenderData {
  fit: number;
  tone: "go" | "no";
  title: string;
  sub: string;
  pros: TenderProCon[];
  cons: TenderProCon[];
}

export function getTenderData(verdict: "go" | "no", lang: Lang): TenderData {
  if (verdict === "go") {
    if (lang === "ru") {
      return {
        fit: 82,
        tone: "go",
        title: "Разработка платформы управления данными",
        sub: "Росатом · Государственный контракт · до 15 млн ₽",
        pros: [
          {
            t: "Стек полностью совпадает",
            d: "TypeScript, React, PostgreSQL — именно то, с чем ваша команда работает каждый день.",
          },
          {
            t: "Бюджет выше рынка",
            d: "Ставка 15 млн на 6 месяцев покрывает команду из 4 человек с хорошей маржой.",
          },
          {
            t: "Нет жёстких требований по сертификации",
            d: "Не нужны ФСБ-лицензии или ФСТЭК. Только опыт разработки госсистем.",
          },
          {
            t: "Знакомый заказчик",
            d: "Работали с этим контрагентом в 2023 году — положительный референс уже есть.",
          },
        ],
        cons: [
          {
            t: "Требуется Istio в проде",
            d: "У вас есть базовый опыт, но production-кейсов с Istio service mesh в резюме нет.",
          },
          {
            t: "Сжатые сроки сдачи",
            d: "MVP через 3 месяца при полном штате. Риск переработок при любых задержках.",
          },
        ],
      };
    } else {
      return {
        fit: 82,
        tone: "go",
        title: "Data management platform development",
        sub: "Rosatom · Government contract · up to ₽15M",
        pros: [
          {
            t: "Tech stack is a perfect match",
            d: "TypeScript, React, PostgreSQL — exactly what your team works with every day.",
          },
          {
            t: "Budget is above market rate",
            d: "₽15M for 6 months covers a 4-person team with a healthy margin.",
          },
          {
            t: "No strict certification requirements",
            d: "No FSB or FSTEC licenses required. Just experience with government systems.",
          },
          {
            t: "Known client",
            d: "You worked with this counterparty in 2023 — a positive reference already exists.",
          },
        ],
        cons: [
          {
            t: "Istio in production required",
            d: "You have basic experience, but no production Istio service mesh cases in the portfolio.",
          },
          {
            t: "Tight delivery schedule",
            d: "MVP in 3 months with full headcount. Risk of overtime if any delays occur.",
          },
        ],
      };
    }
  } else {
    if (lang === "ru") {
      return {
        fit: 34,
        tone: "no",
        title: "Создание системы информационной безопасности",
        sub: "Минобороны · Государственный контракт · до 8 млн ₽",
        pros: [
          {
            t: "Бюджет покрывает затраты",
            d: "8 млн рублей достаточно для небольшой команды на весь период контракта.",
          },
        ],
        cons: [
          {
            t: "Требуется лицензия ФСБ",
            d: "Обязательна лицензия ФСБ на разработку средств защиты. Получение займёт от 6 месяцев.",
          },
          {
            t: "SOC 2 Type II обязателен",
            d: "Заказчик требует действующий сертификат SOC 2 Type II. У вас его нет.",
          },
          {
            t: "Команда не менее 10 человек",
            d: "ТЗ требует минимум 10 специалистов в штате. В вашей команде 4 человека.",
          },
          {
            t: "Опыт в оборонной сфере",
            d: "Необходимы реализованные проекты для силовых структур. В портфолио таких нет.",
          },
          {
            t: "Жёсткие требования к безопасности",
            d: "Работа с секретными данными, специальный режим доступа, выездные проверки заказчика.",
          },
        ],
      };
    } else {
      return {
        fit: 34,
        tone: "no",
        title: "Information security system development",
        sub: "Ministry of Defence · Government contract · up to ₽8M",
        pros: [
          {
            t: "Budget covers costs",
            d: "₽8M is sufficient for a small team for the full contract period.",
          },
        ],
        cons: [
          {
            t: "FSB license required",
            d: "An FSB license for security software development is mandatory. Obtaining it takes 6+ months.",
          },
          {
            t: "SOC 2 Type II mandatory",
            d: "The client requires an active SOC 2 Type II certificate. You don't have one.",
          },
          {
            t: "Team of at least 10 required",
            d: "The spec requires a minimum of 10 specialists on staff. Your team has 4.",
          },
          {
            t: "Defence sector experience",
            d: "Completed projects for security agencies are required. No such cases in the portfolio.",
          },
          {
            t: "Strict security requirements",
            d: "Work with classified data, special access regime, and on-site inspections by the client.",
          },
        ],
      };
    }
  }
}

export function getRequirements(lang: Lang): TenderRequirement[] {
  return [
    {
      label: lang === "ru" ? "TypeScript, 3+ года" : "TypeScript, 3+ years",
      status: "met",
    },
    {
      label:
        lang === "ru" ? "PostgreSQL, репликация" : "PostgreSQL, replication",
      status: "met",
    },
    {
      label:
        lang === "ru" ? "React, дизайн-система" : "React, design system",
      status: "met",
    },
    {
      label:
        lang === "ru"
          ? "Kubernetes в проде"
          : "Kubernetes in production",
      status: "met",
    },
    {
      label: lang === "ru" ? "Istio service mesh" : "Istio service mesh",
      status: "partial",
    },
    { label: "SOC 2 Type II", status: "miss" },
    {
      label: lang === "ru" ? "Команда 5+ человек" : "Team of 5+",
      status: "met",
    },
    {
      label: lang === "ru" ? "Английский B2+" : "English B2+",
      status: "met",
    },
  ];
}

export function getFiles(lang: Lang): TenderFile[] {
  return [
    { name: "tender-tz.pdf", size: "2.1 MB", pages: 34 },
    {
      name:
        lang === "ru"
          ? "trebovaniya-bezopasnosti.pdf"
          : "security-requirements.pdf",
      size: "612 KB",
      pages: 8,
    },
    {
      name:
        lang === "ru" ? "dogovor-draft.docx" : "contract-draft.docx",
      size: "128 KB",
      pages: 12,
    },
  ];
}

export const fitHistory = [82, 45, 67, 91, 34, 78, 52, 88, 41, 73, 60, 55];
export const winTrend = [45, 52, 48, 63, 58, 72, 68, 82];
