# Deal Sense — CLAUDE.md

## Проект
AI-система: генерация КП из .docx шаблонов + анализ тендерной документации.
Версия: 0.2.0

## Архитектура
Modular monolith, Clean Architecture, DDD.

```
deal-sense/
├── backend/
│   ├── cmd/server/             # main.go, DI, graceful shutdown
│   ├── internal/
│   │   ├── domain/             # entities, value objects, domain errors
│   │   ├── usecase/            # бизнес-логика, порты (interfaces)
│   │   ├── adapter/
│   │   │   ├── llm/            # openai_compatible, anthropic, gemini, factory
│   │   │   ├── parser/         # pdf_parser, docx_reader, docx_template, composite
│   │   │   └── http/           # handlers, middleware, router
│   │   └── config/             # конфигурация из env
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── styles/             # tokens.css, reset.css, typography.css
│   │   ├── providers/          # ThemeProvider, I18nProvider
│   │   ├── ui/                 # Button, Chip, Card, StatusPill, FitGauge, Select, Field, Spinner
│   │   ├── icons/              # SVG-иконки как React-компоненты
│   │   ├── components/
│   │   │   ├── Header/
│   │   │   ├── Tabs/
│   │   │   ├── Logo/           # 4 марки + Wordmark + Lockup
│   │   │   ├── charts/         # MiniHistogram, MiniDonut, MiniSparkline
│   │   │   └── Settings/       # SettingsDrawer
│   │   ├── screens/
│   │   │   ├── Tender/         # TenderReport, VerdictHero, ProConCard
│   │   │   └── Proposal/       # ProposalResult
│   │   ├── i18n/               # ru.ts, en.ts, types.ts
│   │   ├── mocks/              # tender.ts, proposal.ts
│   │   └── lib/                # storage.ts
│   ├── e2e/                    # Playwright smoke tests
│   ├── Dockerfile
│   ├── package.json
│   └── vite.config.ts
├── docker-compose.yml
├── justfile
├── VERSION                     # 0.2.0
└── .env.example
```

## Правила

### Go Backend
- Go 1.26, modern features (cmp.Or, t.Context)
- Interfaces в usecase/, не в domain/ (DIP)
- Domain entities только через конструкторы NewXxx
- Доменные ошибки: `var ErrXxx = errors.New(...)` в domain/
- Handler: парсинг → usecase → маппинг. Без uuid.New(), time.Now()
- Cross-module импорты запрещены
- Unreachable OS-errors (MkdirTemp, WriteFile) → panic или _, не return error
- Coverage: 100% internal/, race detector clean

### TDD
- RED → GREEN: два коммита на каждое поведение
- Table-driven tests при ≥3 вариантах
- Backfill честно называть `test: backfill coverage for X`

### LLM Provider
- Интерфейс LLMProvider в usecase/
- Адаптеры: openai_compatible (OpenAI, Groq, Ollama, Custom), gemini, anthropic
- Factory: NewLLMProvider(config) → LLMProvider
- Выбор провайдера через UI + конфиг (env)

### Frontend
- React + Vite + TypeScript (strict)
- CSS-модули + глобальные токены (tokens.css)
- Named exports только, без default
- Тема через data-theme на html, не через JS-классы
- Цвета только через var(--...), никаких хардкод hex
- i18n через React context (не через либу)
- Без UI-библиотек (Mantine, shadcn) — свои примитивы по дизайн-системе
