import { useEffect, useRef, useState } from "react";
import { getItem } from "../../lib/storage";
import { parsePrometheus, type PromMetric } from "../../lib/promParser";

const API_BASE =
  import.meta.env.VITE_API_URL ??
  `${location.protocol}//${location.hostname}:8080`;

interface StoredSettings {
  dealSenseApiKey?: string;
}

export interface UseMetricsResult {
  metrics: PromMetric[];
  error: string | null;
  intervalMs: number;
  setIntervalMs: (n: number) => void;
  lastUpdated: number | null;
}

// useMetrics polls /metrics and parses the Prometheus exposition. The first
// fetch runs synchronously on mount; subsequent fetches fire every intervalMs.
// Switching intervalMs replaces the timer immediately (no double-fire).
//
// On unmount or interval change the in-flight request is aborted so a slow
// response can't land state after the component is gone.
export function useMetrics(initialIntervalMs = 5000): UseMetricsResult {
  const [metrics, setMetrics] = useState<PromMetric[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [intervalMs, setIntervalMs] = useState(initialIntervalMs);
  const [lastUpdated, setLastUpdated] = useState<number | null>(null);
  const aborter = useRef<AbortController | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function tick() {
      aborter.current?.abort();
      const ac = new AbortController();
      aborter.current = ac;

      try {
        const s = getItem<StoredSettings>("llm-settings", {});
        const headers: Record<string, string> = {};
        if (s.dealSenseApiKey) headers["X-API-Key"] = s.dealSenseApiKey;

        const res = await fetch(`${API_BASE}/metrics`, {
          method: "GET",
          signal: ac.signal,
          headers,
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const text = await res.text();
        if (cancelled) return;
        setMetrics(parsePrometheus(text));
        setError(null);
        setLastUpdated(Date.now());
      } catch (e) {
        if (cancelled) return;
        const err = e as Error;
        if (err.name === "AbortError") return;
        setError(err.message);
      }
    }

    void tick();
    const id = window.setInterval(tick, intervalMs);

    return () => {
      cancelled = true;
      window.clearInterval(id);
      aborter.current?.abort();
    };
  }, [intervalMs]);

  return { metrics, error, intervalMs, setIntervalMs, lastUpdated };
}
