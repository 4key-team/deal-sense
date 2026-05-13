# Deal Sense — Security Model

Five-layer defence model following `reflective-agent-defaults v1.4`. Each
layer addresses a different class of threat; layers compose, none of them
is sufficient alone.

---

## Layer overview

| Layer | What | Lives in | ADR |
|-------|------|----------|-----|
| 1 | **Prompt-level firewall** — every LLM system prompt carries the юр firewall + 5 other directives. Domain VO enforces composition. | `internal/domain/security.Policy` | [ADR-006](../../memory/decisions.md) |
| 2 | **Structural ASR scan** — 25-payload regression suite in CI. Asserts firewall markers stay in every system prompt regardless of user input. | `backend/tests/security/` | (this doc) |
| 3 | **Infra-side enforcement** — scoped tokens, rate limits, secrets manager, backups outside blast radius. *Specification* — implementation is the deployment platform's responsibility. | `docs/security/infra-side-requirements.md` | (this doc) |
| 4 | **Risk-action coupling** — every HTTP endpoint classified SAFE_READ / MODIFY / DESTRUCTIVE. Coupling tests pin the firewall under attack patterns (long session, encoded payload, Crescendo). | `internal/domain/security.RiskLevel` + `internal/adapter/llm/coupling_test.go` | [ADR-008](../../memory/decisions.md) |
| 5 | **Out-of-band confirmation** for DESTRUCTIVE ops (type-exact input). *Not yet implemented* — no destructive ops exist in the codebase yet. | (deferred) | (pending) |

Plus orthogonal controls used by all layers:

- **`X-API-Key` middleware** on the backend (`internal/adapter/http/middleware.go:APIKeyAuth`) — passthrough when `DEAL_SENSE_API_KEY` env unset.
- **Telegram bot allowlist** (`internal/domain/auth.Allowlist`) — non-allowlisted user IDs blocked at middleware level.

---

## Layer 1 — Prompt firewall

`domain/security.Policy` is a Value Object that validates the embedded
directives text (`security_directives.md`) carries every required marker.
Adapters compose system prompts via `policy.Wrap(rawPrompt)` — a decorator
that prepends the directives. Production code holds no raw inner prompts.

Six directives (Rule 4 v1.4):
1. STRICT DOMAIN FOCUS
2. ENCODED PAYLOAD ISOLATION
3. NO CYBERATTACKS
4. **FACTUAL INTEGRITY** (the юр firewall — primary risk)
5. RESOURCE ABUSE
6. *(the Russian-text marker `Обратитесь к юристу компании`)*

Test coverage: `internal/domain/security` 100%.

---

## Layer 2 — Structural ASR scan

`backend/tests/security/asr_scan_test.go` runs every payload in
`asr_payloads.json` through the production AnalyzeTender flow with a
prompt-capturing `llmstub`. Asserts:

1. Required firewall markers remain in the system prompt unchanged.
2. The payload's first 40 chars never leak into the system prompt.

Categories and gates:

| Category | Count | Gate |
|----------|-------|------|
| `juridical_masking` | 10 | **Hard 0%** — any single failure fails the test |
| `competitor_extraction` | 5 | Soft <1% aggregate |
| `crescendo` | 5 | Soft <1% aggregate |
| `owasp_llm` | 5 | Soft <1% aggregate |

Wired in CI: `.github/workflows/ci.yml` (`security-scan` job) +
`.github/workflows/security-weekly.yml` (Mondays 06:00 UTC).

**Real-LLM BarkingDog** (with a staging API key) is a future extension —
the structural suite catches regressions in prompt composition; the real
scan catches LLM provider drift. Both share the same payload file.

---

## Layer 3 — Infrastructure-side

See [`infra-side-requirements.md`](infra-side-requirements.md). Engineering
provides the hooks; Ops/SRE owns the enforcement (scoped tokens, rate
limits, secrets manager, backups, TLS, structured logs, health probes).

---

## Layer 4 — Risk-action coupling

Every HTTP endpoint exposed by `cmd/server` is classified:

| Path | Risk |
|------|------|
| `/api/llm/check` | SAFE_READ |
| `/api/llm/providers` | SAFE_READ |
| `/api/llm/models` | SAFE_READ |
| `/api/tender/analyze` | SAFE_READ |
| `/api/proposal/generate` | MODIFY |
| *(none yet)* | DESTRUCTIVE |

Source of truth: `internal/domain/security/default_registry.go`. A
cross-check test asserts every path served by `router.go` has a registry
entry — forgotten annotations fail the build.

Three coupling tests pin Layer 1 invariants under attack patterns:
- `TestLongSession_FirewallInEveryCall` (51 sequential calls)
- `TestEncodedPayload_NotPromotedToDirective`
- `TestCrescendoEscalation_NoStateBleed` (15 ramping steps)

---

## Layer 5 — Out-of-band confirmation (deferred)

Reserved for DESTRUCTIVE operations: external delivery (send proposal to
client by email), state mutations in external systems (update tender
status in CRM), irreversible deletes. None exist in the codebase today.

When the first DESTRUCTIVE endpoint lands:
- `domain/security.ConfirmationChallenge` VO (type-exact or inline y/n)
- Telegram bot middleware that intercepts DESTRUCTIVE commands, posts the
  challenge, waits for user reply, verifies before execution
- 4 TDD commits per the master plan

Sanity test `TestDefaultEndpointRegistry_NoDestructiveYet` flags any
accidental DESTRUCTIVE classification until Layer 5 lands.

---

## External-subprocess attack surface (LibreOffice)

Two adapters shell out to `soffice` (LibreOffice headless) inside the
backend container: `adapter/pdf/libreoffice.go` (DOCX → PDF) and, as of
v0.16.0, `adapter/parser/doc_converter.go` (DOC → DOCX for legacy Word
97-2003 input). Both run with the same hardening:

| Concern | Mitigation |
|---------|------------|
| Hang on malformed input | `exec.CommandContext` + 60s deadline; `cmd.WaitDelay = 1s` force-closes IO if child processes hold stderr open after kill |
| Tmp-dir leaks between requests | `os.MkdirTemp` with unique prefix per call (`doc2docx-*`, `docx2pdf-*`) + `defer os.RemoveAll` |
| Race over shared filenames | Each request has its own isolated tmp dir — no shared paths |
| Shell injection via filename | `exec.Command` slice form (no shell expansion); filenames inside the converter are constants (`input.doc` / `input.docx`), not user-supplied |
| Disk write of input perms | `os.WriteFile(..., 0o600)` |
| Upload size DoS | `maxUploadSize = 50 MB` enforced at HTTP layer (`handler_tender.go:11`); ZIP entries additionally capped at 50 MB decompressed (`usecase/zip_extract.go:18`) |

Residual risks:

- LibreOffice itself is a large attack surface. Network egress from the
  container should be restricted at the infra layer (Layer 3); the
  process inside the container should not have outbound internet beyond
  what the LLM provider needs.
- Future converters that accept user-controlled filenames must keep the
  constant-internal-name pattern to retain the shell-injection mitigation.

---

## Pre-flight checklist before production deploy

- [x] Layer 1: SecurityDirectives compiled into every LLM prompt (ADR-006)
- [x] Layer 2: ASR scan in CI, gates wired
- [ ] Layer 3: `infra-side-requirements.md` reviewed by DevOps; controls implemented in target platform
- [x] Layer 4: coupling tests passing; endpoint registry populated
- [ ] Layer 5: not required until a DESTRUCTIVE endpoint exists
- [x] Health probes `/healthz` + `/readyz` wired (ADR-009)
- [x] Per-IP rate limit middleware (`RATE_LIMIT_RPS`) — defence-in-depth (ADR-009)
- [ ] Tool annotations exported via observability endpoint (e.g. `/metrics/security`) — backlog
- [ ] Secrets manager integration (Vault / SOPS / cloud KMS) — backlog
- [ ] Daniil + DevOps signed off on the release

---

## Reporting a security issue

Email `daniilvdovin4@gmail.com` or open a private GitHub Security Advisory.
Do not file public issues for security defects.
