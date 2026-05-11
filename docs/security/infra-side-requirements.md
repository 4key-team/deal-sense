# Layer 3 — Infrastructure-Side Enforcement Requirements

**Status:** specification. The application code cannot enforce these from
inside its own process — they live in the deployment platform (API
gateway, IAM, secrets manager, observability stack). This document is the
contract between Engineering and Ops/SRE.

Source: `reflective-agent-defaults v1.4 §Layer 3`. Companion to ADR-006
(Layer 1), ADR-007 (Telegram bot), ADR-008 (Layer 4).

---

## Required controls

| # | Requirement | Where it lives | What it blocks | Verification |
|---|-------------|----------------|----------------|--------------|
| 1 | **Scoped API tokens** | IAM / API gateway | A leaked `telegram-bot` token cannot perform admin actions; it has READ + ANALYZE + GENERATE scopes only. No volume/delete/transfer scopes are issued to that token. | Token issue process documented in `docs/security/SECURITY.md`; periodic audit of issued scopes. |
| 2 | **Per-user rate limit** | API gateway (Nginx limit_req, AWS API Gateway throttling, Cloudflare, etc.) | Single user or single bot account cannot drain the LLM budget. | 30 req/min and 5 generate/hour soft caps; pager alert at 80% of cap. |
| 3 | **Production guard for DESTRUCTIVE** | Middleware in front of cmd/server | DESTRUCTIVE endpoints (none today — see Layer 5) require out-of-band confirmation; the middleware refuses without a verified token. | Smoke test on staging that calls DESTRUCTIVE without confirmation and expects 403. |
| 4 | **Backups outside the blast radius** | DBA / DevOps | A compromised production DB cannot also wipe its own backups. | Daily snapshots in a different region/account. Restore drill once per quarter. |
| 5 | **MCP scoped tokens** | IAM | If the project gains MCP tools, those tokens are not shared with the CLI/dev workflow. | Each MCP server has its own service account; audit via IAM inventory. |
| 6 | **Secrets manager (not env vars)** | Vault / SOPS / cloud KMS | Plaintext secrets in container env are observable by anyone with `docker inspect` or pod-exec. | Secrets injected at runtime via secrets manager; `.env.example` documents structure but never holds real values. |
| 7 | **TLS termination + HSTS** | Load balancer / Cloudflare | Plaintext HTTP carries `X-API-Key` and `X-LLM-Key` — both sensitive. | All public endpoints redirect HTTP→HTTPS; HSTS header set with ≥1 year max-age. |
| 8 | **Structured access logs** | Log aggregator | Without logs, an incident cannot be reconstructed. | Every request logged with timestamp, route, status, latency, user/bot ID, request ID. Retained ≥ 90 days. |
| 9 | **Health & readiness probes** | Container orchestrator | A wedged process keeps receiving traffic. | `/healthz` (liveness) and `/readyz` (readiness) endpoints; orchestrator reschedules on failure. |

---

## Non-requirements (do **not** put these in app code)

- DDOS protection — belongs at CDN/edge.
- WAF rules for SQLi/XSS — belongs at WAF.
- Geo-fencing of Telegram users — Telegram Bot API doesn't expose IPs reliably; rely on allowlist instead.
- Egress filtering of LLM provider hostnames — belongs at the egress proxy.

---

## How the app code already supports this

| Control | App-side hook |
|---------|---------------|
| Scoped tokens | `X-API-Key` middleware (`internal/adapter/http/middleware.go:APIKeyAuth`). Configured per-deployment via `DEAL_SENSE_API_KEY`. |
| Rate limit | Application-level rate limit middleware is defence-in-depth (see backlog) but the primary mitigation is the gateway. |
| DESTRUCTIVE guard | `domain/security.RiskLevel` + `EndpointRegistry` (ADR-008). When DESTRUCTIVE endpoints land they will be wired with Layer 5 out-of-band confirmation. |
| Access logs | `apphttp.Logger` middleware in `cmd/server/main.go` emits structured slog records. |
| Health probes | Not yet present — see backlog in master plan. |

---

## Open items before production deployment

- [ ] Health/readiness probes (`/healthz`, `/readyz`) wired into `cmd/server`.
- [ ] Rate limit middleware on handler level (defence-in-depth with the gateway).
- [ ] Secrets manager integration (currently plain env vars).
- [ ] Pre-flight checklist sign-off — see `docs/security/SECURITY.md`.

---

## Related

- [ADR-006 — SecurityPolicy](../../memory/decisions.md) — Layer 1 (prompt-level firewall)
- [ADR-008 — RiskLevel + EndpointRegistry](../../memory/decisions.md) — Layer 4
- [Master plan](../plans/2026-05-11-telegram-security-enterprise.md)
