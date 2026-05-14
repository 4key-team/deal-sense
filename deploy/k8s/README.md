# Deal Sense — Kubernetes deployment

Reference manifests для backend (`cmd/server`). Каждый файл — отдельный
ресурс, нумерация в имени = apply order (`kubectl apply -f deploy/k8s/`
сортирует alphabetically).

## Apply order

```bash
kubectl apply -f deploy/k8s/00-namespace.yaml
kubectl apply -f deploy/k8s/10-configmap.yaml

# Секреты — TEMPLATE в 20-secret-llm.example.yaml. В dev можно создать
# напрямую:
kubectl create secret generic deal-sense-llm \
  --namespace=deal-sense \
  --from-literal=LLM_API_KEY="$LLM_API_KEY" \
  --from-literal=DEAL_SENSE_API_KEY="$DEAL_SENSE_API_KEY"

kubectl apply -f deploy/k8s/30-deployment.yaml
kubectl apply -f deploy/k8s/40-service.yaml
kubectl apply -f deploy/k8s/50-ingress.yaml
```

## Что внутри

| Файл | Ресурс | Описание |
|---|---|---|
| `00-namespace.yaml` | Namespace | Изолированный namespace `deal-sense` |
| `10-configmap.yaml` | ConfigMap | Не-secret env (LLM_PROVIDER, RATE_LIMIT_RPS, …) |
| `20-secret-llm.example.yaml` | Secret | **Template**; production = Sealed Secrets / External Secrets |
| `30-deployment.yaml` | Deployment | Backend, 2 replicas, probes, securityContext, *_FILE secret mount, Prometheus scrape annotations |
| `40-service.yaml` | Service | ClusterIP, port 80 → container port 8080 |
| `50-ingress.yaml` | Ingress | nginx ingress с SSE-friendly timeouts (600s) + cert-manager TLS hint |

## Production checklist

- [ ] Заменить image tag `:latest` в `30-deployment.yaml` на immutable digest (`@sha256:…`).
- [ ] Установить Sealed Secrets controller (`bitnami-labs/sealed-secrets`) или External Secrets с Vault / AWS Secrets Manager. Не применять `20-secret-llm.example.yaml` напрямую.
- [ ] Заменить host `api.deal-sense.example.com` в `50-ingress.yaml` на реальный.
- [ ] Установить cert-manager + ClusterIssuer `letsencrypt-prod` (или удалить annotation и привести свой TLS секрет).
- [ ] Настроить Prometheus scrape job (target `:8080/metrics`) или установить PodMonitor CRD для kube-prometheus-stack.
- [ ] Заимпортить Grafana dashboard под `dealsense_*` метрики (см. `internal/adapter/metrics/collector.go`).
- [ ] PodDisruptionBudget (минимум 1 available) — вне scope reference.
- [ ] NetworkPolicy ingress=nginx-controller, egress=LLM provider + DNS — вне scope reference.
- [ ] HorizontalPodAutoscaler по CPU/memory или custom metric — вне scope reference.

## Что НЕ включено

- **Frontend** (React + Vite). Статика — обычно отдельный ingress или CDN (Cloudflare / S3+CloudFront). См. `frontend/Dockerfile` если хотите подать через тот же кластер.
- **Telegram bot** (`cmd/telegram-bot`). Polling-only клиент Telegram API; не нуждается в ingress. Достаточно отдельного Deployment без Service.
- **Мониторинг** (Prometheus / Grafana). Используйте `kube-prometheus-stack` chart.
- **Backup**. Backend stateless — нечего бэкапить (за исключением telegram-bot-data volume, если включён TG bot).

## Verification после deploy

```bash
# Pods Running + Ready
kubectl get pods -n deal-sense

# Logs не паникуют, init завершился
kubectl logs -n deal-sense deploy/deal-sense-backend --tail=50

# Bypass endpoints (no auth)
curl -k https://api.deal-sense.example.com/healthz
curl -k https://api.deal-sense.example.com/readyz
curl -k https://api.deal-sense.example.com/metrics

# Gated endpoint (API key required)
curl -k -H "X-API-Key: $DEAL_SENSE_API_KEY" \
  https://api.deal-sense.example.com/api/llm/providers
```

В `cmd/server/bypass_chain_test.go` зафиксирован тот же контракт на уровне
кода — `/healthz`, `/readyz`, `/metrics` отвечают 200 без X-API-Key,
`/api/llm/*` требует header.

## Customization

- **Реплики**: измените `spec.replicas` в `30-deployment.yaml`.
- **Resource limits**: подкрутите requests/limits под реальную нагрузку. Опыт показывает 100m CPU / 128Mi memory достаточно для idle, 1000m CPU peaks при Opus LLM call.
- **Rate limit**: правьте `RATE_LIMIT_RPS` и `RATE_LIMIT_BURST` в `10-configmap.yaml`.
- **CORS**: `CORS_ORIGIN` — в production не `*`, ставьте конкретный домен фронтенда.
- **LLM provider**: переключайте через `LLM_PROVIDER` в ConfigMap (anthropic / openai / gemini / groq / ollama / custom). См. `internal/adapter/llm/` для списка поддерживаемых провайдеров.

## kustomize / helm

Эти manifests — точка отсчёта. Для production с несколькими environment
(staging / prod / region-1 / region-2) типично:

```
deploy/k8s/
├── base/                  # текущий flat layout
└── overlays/
    ├── staging/
    │   ├── kustomization.yaml
    │   └── replicas-patch.yaml
    └── prod/
        ├── kustomization.yaml
        ├── replicas-patch.yaml
        └── ingress-host-patch.yaml
```

Если кластер на Helm — превратите в chart `deploy/helm/deal-sense/` с
values.yaml для replicas/image/host/etc. Reference остаётся flat для
читаемости.
