import { useEffect, useState } from "react";
import { getItem } from "../../../lib/storage";

const API_BASE =
  import.meta.env.VITE_API_URL ??
  `${location.protocol}//${location.hostname}:8080`;

interface StoredSettings {
  dealSenseApiKey?: string;
}

export interface LLMInfoResult {
  providers: string[];
  models: string[];
  error: string | null;
}

interface ProvidersResponse {
  providers?: string[];
}

interface ModelsResponse {
  models?: string[];
  error?: string;
}

// useLLMInfo fetches the server-side LLM info (provider list + models for the
// active provider) for the /admin/settings view-only section. The endpoints
// already exist (GET /api/llm/providers, GET /api/llm/models) and are used
// by the main proposal/tender screens.
//
// Failure is surfaced as a single error string — UI degrades to "loading…"
// markers but does not block the rest of the settings page.
export function useLLMInfo(): LLMInfoResult {
  const [providers, setProviders] = useState<string[]>([]);
  const [models, setModels] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const ac = new AbortController();

    async function load() {
      const s = getItem<StoredSettings>("llm-settings", {});
      const headers: Record<string, string> = {};
      if (s.dealSenseApiKey) headers["X-API-Key"] = s.dealSenseApiKey;

      try {
        const [pRes, mRes] = await Promise.all([
          fetch(`${API_BASE}/api/llm/providers`, {
            signal: ac.signal,
            headers,
          }),
          fetch(`${API_BASE}/api/llm/models`, {
            signal: ac.signal,
            headers,
          }),
        ]);
        if (!pRes.ok) throw new Error(`providers: HTTP ${pRes.status}`);
        if (!mRes.ok) throw new Error(`models: HTTP ${mRes.status}`);
        const pBody = (await pRes.json()) as ProvidersResponse;
        const mBody = (await mRes.json()) as ModelsResponse;
        if (cancelled) return;
        setProviders(pBody.providers ?? []);
        setModels(mBody.models ?? []);
        setError(mBody.error ?? null);
      } catch (e) {
        if (cancelled) return;
        const err = e as Error;
        if (err.name === "AbortError") return;
        setError(err.message);
      }
    }
    void load();
    return () => {
      cancelled = true;
      ac.abort();
    };
  }, []);

  return { providers, models, error };
}
