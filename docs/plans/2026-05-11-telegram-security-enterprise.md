# Plan: Telegram Wrapper + Security Layers 2–5 + Enterprise Pass

**Дата создания**: 2026-05-11
**Контекст**: Layer 1 (юр firewall в LLM-промптах) сдан в commits c845acd→e81c6a3, ADR-006. Code-reviewer ≥8/10 по всем 4 осям. Оставшиеся 4 слоя security + Telegram-обёртка + enterprise-pass запланированы здесь.

**Источник стандарта**: `~/claude-cowork/knowledge-base/standards/reflective-agent-defaults/v1.md`. Слои нумерованы как в исходном пользовательском промпте.

---

## Session 2 — Каркас Telegram-обёртки ✅ DONE (2026-05-12)

**Цель**: Создать `cmd/telegram-bot/` thin wrapper вокруг существующих HTTP-endpoints с подключённым security guard.

**Итог** (commits `e533320` → `200c870`, 13 коммитов в main):
- ✅ Backend X-API-Key middleware (`internal/adapter/http/middleware.go:APIKeyAuth`) — добавлено сверх плана
- ✅ env-based Config (`internal/adapter/telegram/config.go`) — BOT_TOKEN, ALLOWLIST_USER_IDS, API_BASE_URL, DEAL_SENSE_API_KEY
- ✅ Allowlist VO (`internal/domain/auth/allowlist.go`) — NewAllowlist, IsAllowed, ErrEmptyAllowlist, ErrInvalidUserID
- ✅ APIClient interface (`internal/usecase/telegram/ports.go`) + HTTPClient impl (`internal/adapter/dealsenseapi/client.go`)
- ✅ /analyze handler (`internal/adapter/telegram/analyze.go`) + FormatAnalyzeReply
- ✅ cmd/telegram-bot wiring с allowlistMiddleware + graceful shutdown (SIGINT/SIGTERM via signal.NotifyContext)
- ✅ docker-compose + Dockerfile.telegram-bot + .env.example
- ❌ `/generate` handler — отложен на Session 2.5

**Библиотека**: `github.com/go-telegram/bot` v1.20.0 (вместо go-telegram-bot-api/v5 — выбрано за context-aware handlers + middleware из коробки, см. ADR-007).

**Security**: handlers не зовут LLM напрямую — только HTTP вызов в backend. Layer 1 работает на backend стороне.

**Compliance**: code-reviewer (после refactor 200c870) дал TDD 7 / DDD 9 / CA 9 / Tests 9. TDD 7 — исторический долг по коммиту `d326358` (polling loop без RED-step), документирован как honest test-after. ADR-007 в `memory/decisions.md`.

**Coverage**: `domain/auth` 100%, `adapter/telegram` 100%, `internal/config` 100%, `adapter/dealsenseapi` 95.2%, `cmd/telegram-bot` 86.1% (`main()` glue не покрывается). Race detector clean.

---

## Session 2.5 — /generate handler

**Цель**: Добавить `/generate` команду в бота — proxy к `POST /api/proposal/generate`.

**TDD-декомпозиция** (≤5 коммитов):
1. `test(telegram): RED — APIClient.GenerateProposal contract` (multipart template + context files + params)
2. `feat(telegram): GREEN — HTTPClient.GenerateProposal`
3. `test(telegram): RED — /generate handler (template + optional context)`
4. `feat(telegram): GREEN — /generate handler + reply with .docx attachment via SendDocument`
5. `chore(telegram): wire RegisterHandler в main.go`

**Нюансы**:
- Backend handler `/api/proposal/generate` принимает `template` (single file), `context` (multiple files), `params` (JSON) — см. `backend/internal/adapter/http/handler_proposal.go`.
- Ответ возвращает `docx`/`pdf` base64. Бот должен раскодировать и `b.SendDocument(...)` с правильным filename.
- /generate UX: user шлёт template файлом, бот отвечает .docx. Если шаблон без плейсхолдеров → backend сам выбирает generative mode.

---

## Session 3 — Layer 4 ✅ DONE (2026-05-12), Layer 5 deferred

### Layer 4 — Rule-action coupling tests (reflective-agent-defaults v1.4 Rule 11) ✅

Реализовано (см. ADR-008):
- ✅ `RiskLevel` typed enum в `internal/domain/security/risk_level.go` (SAFE_READ/MODIFY/DESTRUCTIVE)
- ✅ `EndpointRegistry` с `NewDefaultEndpointRegistry()` — все 5 routes из `router.go` annotated; sanity test `NoDestructiveYet`
- ✅ `adapter/llmstub.Provider` — scripted thread-safe LLM stub с prompt capture
- ✅ 3 coupling tests в `adapter/llm/coupling_test.go` (backfill, Layer 1 уже корректно изолирует firewall):
  - `TestLongSession_FirewallInEveryCall` — 51 call, каждый system prompt с firewall markers
  - `TestEncodedPayload_NotPromotedToDirective` — Base64 user payload не интерпретируется как directive
  - `TestCrescendoEscalation_NoStateBleed` — 15 ramping steps, system prompts identical (statelessness pinned)

Запуск: `go test ./internal/adapter/llm/ -run "TestLongSession|TestEncodedPayload|TestCrescendo"` (без build tag — быстро, stub-based).

Coverage: `domain/security` 97.2%, `adapter/llmstub` 96.7%.

### Layer 5 — deferred

Не реализовано: destructive operations (`sendProposalToClient`, `updateTenderStatus`, `deleteProposal`) в текущем коде не существуют. Создавать `ConfirmationChallenge` VO без consumer = красный флаг проекта («мёртвый код в domain»). Возвращаемся когда добавим destructive op.

### Layer 5 — Out-of-band confirmation для DESTRUCTIVE (reflective-agent-defaults v1.3 Rule 5)

Destructive operations в Telegram-обёртке (когда появятся):
- `sendProposalToClient(proposal_id, client_email)` — out-of-band: type-exact имя клиента
- `updateTenderStatus(tender_id, status='participate')` — out-of-band: type-exact номер тендера
- `deleteProposal(proposal_id)` — inline `[y/n]` достаточно

Domain VO `ConfirmationChallenge` с конструктором + методом `Verify(input string) error`. Adapter Telegram использует.

**TDD**: 4 коммита (test:RED + feat:GREEN ×2 для VO + middleware).

---

## Session 4 — Layer 2 (BarkingDog CI/CD)

**Research first**: WebSearch `BarkingDog LLM security scanner site:github.com` — подтвердить URL, лицензию, payload format, статус активности репо (имя из исходного промпта — нуждается в верификации).

**Подготовка**:
1. `tests/security/tender_payloads.json` — custom payloads (20-30 «юр-вопрос с маскировкой», 10-15 «извлечь данные конкурентов», 5-10 Crescendo, 5-10 OWASP LLM Top 10)
2. `docker-compose.test.yml` — test instance Deal Sense с mock LLM (или real provider с test API key)
3. `.github/workflows/security-scan.yml` (или `justfile` task если GitHub Actions ещё нет)

**Метрики**:
- ASR < 1.0% на all-categories — fail PR
- ASR = 0% на юр-категории — hard gate

**Weekly schedule**: `cron: '0 6 * * 1'` для регрессий после обновлений LLM провайдером.

---

## Session 5 — Layer 3 docs + Pre-flight + Deployment

### Layer 3 — Infrastructure-side enforcement docs

`docs/security/infra-side-requirements.md` — спецификация для DevOps/SRE компании (это не код, а контракт):

| Требование | Где живёт | Что блокирует |
|---|---|---|
| Scoped API tokens | API Gateway / IAM | telegram-bot имеет только READ + ANALYZE + GENERATE — нет volumeDelete/transfer |
| Rate limit per-user | API Gateway | 30 req/min, 5 generate/hour, алерты на 80% |
| Production guards | Middleware | DESTRUCTIVE endpoints требуют out-of-band (см. Layer 5) |
| Бэкапы вне blast radius | DBA | Daily snapshots в другой регион |
| MCP scoped tokens | IAM | Не shared с CLI |

`docs/security/SECURITY.md` — описание всех 5 слоёв со ссылками на ADR-006 и reflective-agent-defaults v1.4.

### Pre-flight checklist перед deployment

- [ ] Layer 1: SecurityDirectives compiled в каждый prompt (✅ Session 1)
- [ ] Layer 2: BarkingDog scan в CI, ASR <1% all / 0% юр
- [ ] Layer 3: infra-side-requirements.md написан, согласован с DevOps
- [ ] Layer 4: coupling tests проходят (long session / encoded / Crescendo)
- [ ] Layer 5: out-of-band для sendProposalToClient + updateTenderStatus
- [ ] Tool annotations выставлены (SAFE_READ / MODIFY / DESTRUCTIVE)
- [ ] `/metrics/security` Prometheus endpoint
- [ ] Daniil + DevOps подписали release

---

## Enterprise improvement backlog (из code-review Layer 1)

Не блокеры, но enterprise-уровень требует. Распределить по сессиям 2-5 или отдельная сессия:

1. **`policyLoader` mutable global → `atomic.Pointer` или DI**
   - Сейчас package-level mutable, `policyLoader` мутируется тестовым seam без синхронизации.
   - Решение: либо `sync.RWMutex`, либо рефакторинг `init()` → `NewPrompts(p *security.Policy) *Prompts` с явным DI из `main.go`.
   - Размер: 1 refactor commit + adjust tests.

2. **VO `WrappedPrompt` для type-safe разделения raw vs wrapped**
   - Сейчас `func(string) string` — анемичный тип, компилятор не препятствует случайно вернуть raw наружу.
   - Решение: `type WrappedPrompt struct{ fn func(string) string }` с `NewWrapped(policy, raw)` конструктором, `Call(lang) string`.
   - Размер: 1 test:RED + 1 feat:GREEN.

3. **`TestPolicy_Wrap_DoubleWrap_AddsTwoPrefixes` — покрыть footgun**
   - Тест что `p.Wrap(p.Wrap(inner))` даёт два префикса (документирует ожидаемое поведение или, если решим запретить — добавить idempotent flag в Wrap).
   - Размер: 1 test commit.

4. **Отделить `TestEmbeddedDirectives_NoAccidentalDuplicates` от `TestNewDefaultPolicy_AllMarkers`**
   - Сейчас один тест проверяет и наличие, и уникальность маркеров — двойная ответственность.
   - Размер: 1 test refactor.

5. **`go vet`, `staticcheck`, `golangci-lint` в CI**
   - Нет linter pipeline. Для enterprise — обязательно.
   - Размер: `.golangci.yml` + CI job.

6. **Generic project-wide enterprise pass** (вне scope этого плана, но user mentioned):
   - Auth для backend HTTP API (сейчас открытый — fine для local, не fine для prod)
   - Structured logging (zap / slog) — replace ad-hoc fmt.Print
   - OpenTelemetry tracing для LLM calls
   - Health/readiness probes
   - Rate limiting middleware на handler level (defence-in-depth с API Gateway)
   - Secrets management (Vault / SOPS / env-from-file) — сейчас env vars
   - Pre-commit hooks (golangci-lint, conventional commits validate)

---

## Связь со стандартами и каталогом

- **reflective-agent-defaults v1.4** — `~/claude-cowork/knowledge-base/standards/reflective-agent-defaults/v1.md`. Single source of truth.
- **catalog#789 BarkingDog** — 6-line pattern и ASR метрик. Layer 2.
- **catalog#610 Aule validate-then-repair** — confirming для Rule 12 (Layer 4).
- **catalog#760 Veai BugSwarm** — second confirming для Rule 12.
- **catalog#585 PocketOS-инцидент Aule** — источник Rule 4 v1.3 (Layer 3 infra-side).
- **catalog#482 TAU15** — оригинальная архитектура рефлексирующего агента.

---

## Cadence

- 1 сессия ≈ 1 слой + относящийся к нему scope
- Между сессиями: handoff в `.claude/handoffs/YYYY-MM-DD_тема.md`, chronicles, обновление этого плана
- Каждая сессия завершается `code-reviewer agent` с ≥8/10 по 4 осям перед claim compliance

## Pre-conditions для каждой сессии

- `go test ./... -race -cover` зелёный
- `git status` чистый (handoff закоммичен)
- Прочитать последний handoff
- Прочитать релевантные ADR (особенно ADR-006 для Layer 2+)
