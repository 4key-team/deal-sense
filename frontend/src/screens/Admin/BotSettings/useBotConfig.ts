import { useCallback, useEffect, useRef, useState } from "react";
import { getItem } from "../../../lib/storage";

const API_BASE =
  import.meta.env.VITE_API_URL ??
  `${location.protocol}//${location.hostname}:8080`;

interface StoredSettings {
  dealSenseApiKey?: string;
}

// BotConfigResponse mirrors backend AdminBotConfigResponse. When Configured
// is false the bot has no persisted config and falls back to env vars.
export interface BotConfigResponse {
  configured: boolean;
  masked_token?: string;
  allowlist_open?: boolean;
  allowlist_user_ids?: number[];
  log_level?: string;
}

// BotConfigValidationError carries the backend's structured 4xx body: a
// translated message plus the field that should be highlighted in the UI.
export interface BotConfigValidationError {
  error: string;
  field?: string;
}

// BotConfigUpdateInput is the JSON we PUT to the backend.
export interface BotConfigUpdateInput {
  token: string;
  allowlist_user_ids: number[];
  log_level: string;
}

export interface UseBotConfigResult {
  data: BotConfigResponse | null;
  error: string | null;
  loading: boolean;
  validation: BotConfigValidationError | null;
  saving: boolean;
  update: (input: BotConfigUpdateInput) => Promise<boolean>;
  refresh: () => void;
}

// useBotConfig loads the current /api/admin/bot-config on mount and exposes
// an `update` function that PUTs new values. Validation errors from the
// backend are surfaced separately (so the UI can highlight a specific
// field) from transport / 5xx errors (single message banner).
//
// The X-API-Key header is read from the same localStorage slot the rest of
// the admin UI uses (llm-settings → dealSenseApiKey).
export function useBotConfig(): UseBotConfigResult {
  const [data, setData] = useState<BotConfigResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [validation, setValidation] = useState<BotConfigValidationError | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const aborter = useRef<AbortController | null>(null);

  const authHeaders = useCallback((): Record<string, string> => {
    const s = getItem<StoredSettings>("llm-settings", {});
    return s.dealSenseApiKey ? { "X-API-Key": s.dealSenseApiKey } : {};
  }, []);

  useEffect(() => {
    let cancelled = false;
    aborter.current?.abort();
    const ac = new AbortController();
    aborter.current = ac;

    async function load() {
      setLoading(true);
      try {
        const res = await fetch(`${API_BASE}/api/admin/bot-config`, {
          method: "GET",
          signal: ac.signal,
          headers: authHeaders(),
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = (await res.json()) as BotConfigResponse;
        if (cancelled) return;
        setData(json);
        setError(null);
      } catch (e) {
        if (cancelled) return;
        const err = e as Error;
        if (err.name === "AbortError") return;
        setError(err.message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
      ac.abort();
    };
  }, [reloadTick, authHeaders]);

  const refresh = useCallback(() => setReloadTick((n) => n + 1), []);

  const update = useCallback(
    async (input: BotConfigUpdateInput): Promise<boolean> => {
      setSaving(true);
      setValidation(null);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/api/admin/bot-config`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            ...authHeaders(),
          },
          body: JSON.stringify(input),
        });
        if (res.status === 400) {
          const body = (await res.json()) as BotConfigValidationError;
          setValidation(body);
          return false;
        }
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`);
        }
        const json = (await res.json()) as BotConfigResponse;
        setData(json);
        return true;
      } catch (e) {
        const err = e as Error;
        setError(err.message);
        return false;
      } finally {
        setSaving(false);
      }
    },
    [authHeaders]
  );

  return { data, error, loading, validation, saving, update, refresh };
}
