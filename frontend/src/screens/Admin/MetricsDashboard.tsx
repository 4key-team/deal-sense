import { useMetrics } from "./useMetrics";
import type { PromMetric } from "../../lib/promParser";
import { MiniSparkline } from "../../components/charts";
import styles from "./MetricsDashboard.module.css";

const INTERVAL_OPTIONS = [
  { value: 1000, label: "1s" },
  { value: 5000, label: "5s" },
  { value: 30000, label: "30s" },
] as const;

function findMetric(metrics: PromMetric[], name: string): PromMetric | undefined {
  return metrics.find((m) => m.name === name);
}

function humaniseError(msg: string): string {
  // Browser fetch surfaces "Failed to fetch" / "NetworkError" for transport
  // issues — translate to RU per CLAUDE.md i18n rule, leave others as-is so
  // operator can debug.
  if (/failed to fetch|network/i.test(msg)) {
    return "Не удалось загрузить метрики";
  }
  return msg;
}

export function MetricsDashboard() {
  const { metrics, error, intervalMs, setIntervalMs, lastUpdated } =
    useMetrics(5000);

  const requests = findMetric(metrics, "dealsense_requests_total");
  const llm = findMetric(metrics, "dealsense_llm_calls_total");
  const risk = findMetric(metrics, "dealsense_endpoint_risk");
  const declines = findMetric(metrics, "dealsense_security_decline_total");

  return (
    <div className={styles.dashboard}>
      <header className={styles.header}>
        <h1>Metrics</h1>
        <label className={styles.picker}>
          Refresh:
          <select
            aria-label="Refresh interval"
            value={intervalMs}
            onChange={(e) => setIntervalMs(Number(e.target.value))}
          >
            {INTERVAL_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </label>
        {lastUpdated !== null && (
          <span className={styles.updated}>
            Updated: {new Date(lastUpdated).toLocaleTimeString()}
          </span>
        )}
      </header>

      {error && (
        <div className={styles.error} role="alert">
          ❌ {humaniseError(error)}
        </div>
      )}

      <section className={styles.section} aria-label="Requests">
        <h2>Requests by path</h2>
        <SeriesList metric={requests} format={(s) => `${s.labels.path} (${s.labels.status})`} />
      </section>

      <section className={styles.section} aria-label="LLM">
        <h2>LLM calls by provider</h2>
        <SeriesList metric={llm} format={(s) => `${s.labels.provider} / ${s.labels.status}`} />
      </section>

      <section className={styles.section} aria-label="Endpoint risk">
        <h2>Endpoint risk</h2>
        {risk && risk.series.length > 0 ? (
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Path</th>
                <th>Risk level</th>
              </tr>
            </thead>
            <tbody>
              {risk.series.map((s, i) => (
                <tr key={`${s.labels.path}-${s.labels.level}-${i}`}>
                  <td>{s.labels.path}</td>
                  <td>{s.labels.level}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <span className={styles.empty}>no data yet</span>
        )}
      </section>

      <section className={styles.section} aria-label="Security declines">
        <h2>Security declines by kind</h2>
        <SeriesList
          metric={declines}
          format={(s) => s.labels.kind}
          showSparkline
        />
      </section>
    </div>
  );
}

interface SeriesListProps {
  metric: PromMetric | undefined;
  format: (s: PromMetric["series"][number]) => string;
  showSparkline?: boolean;
}

function SeriesList({ metric, format, showSparkline = false }: SeriesListProps) {
  if (!metric || metric.series.length === 0) {
    return <span className={styles.empty}>no data yet</span>;
  }
  return (
    <>
      {metric.series.map((s, i) => (
        <div key={`${format(s)}-${i}`} className={showSparkline ? styles.chartRow : styles.row}>
          <span>{format(s)}</span>
          {showSparkline ? (
            // Sparkline needs ≥2 points to render a line — the dashboard only
            // has one snapshot per fetch, so we feed a degenerate [0, value]
            // pair just to fill the SVG height bar; replace with rolling
            // history once the dashboard accumulates samples over time.
            <MiniSparkline data={[0, s.value]} height={20} />
          ) : (
            <span>{s.value}</span>
          )}
        </div>
      ))}
    </>
  );
}
