import { getItem } from "./storage";

const BASE =
  import.meta.env.VITE_API_URL ??
  `${location.protocol}//${location.hostname}:8080`;

export interface TenderProCon {
  title: string;
  desc: string;
}

export interface TenderRequirement {
  label: string;
  status: "met" | "partial" | "miss";
}

export interface TokenUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface TenderResult {
  verdict: string;
  risk: string;
  score: number;
  summary: string;
  pros: TenderProCon[];
  cons: TenderProCon[];
  requirements: TenderRequirement[];
  effort: string;
  usage?: TokenUsage;
}

export interface CheckResult {
  ok: boolean;
  provider: string;
  error?: string;
}

export interface ProviderInfo {
  id: string;
  name: string;
  models: string[];
}

export interface ProvidersResult {
  providers: ProviderInfo[];
}

interface StoredSettings {
  providerId?: string;
  apiKey?: string;
  url?: string;
  model?: string;
  dealSenseApiKey?: string;
}

function apiHeaders(): Record<string, string> {
  const s = getItem<StoredSettings>("llm-settings", {});
  const headers: Record<string, string> = {};
  if (s.apiKey) headers["X-LLM-Key"] = s.apiKey;
  if (s.providerId) headers["X-LLM-Provider"] = s.providerId;
  if (s.url) headers["X-LLM-URL"] = s.url;
  if (s.model) headers["X-LLM-Model"] = s.model;
  if (s.dealSenseApiKey) headers["X-API-Key"] = s.dealSenseApiKey;
  return headers;
}

export async function analyzeTender(
  files: File[],
  companyProfile: string,
  lang = "ru",
): Promise<TenderResult> {
  const form = new FormData();
  form.append("company_profile", companyProfile);
  form.append("lang", lang);
  files.forEach((f) => form.append("files", f));

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5 * 60 * 1000);

  try {
    const res = await fetch(`${BASE}/api/tender/analyze`, {
      method: "POST",
      headers: apiHeaders(),
      body: form,
      signal: controller.signal,
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error((body as Record<string, string>).error || `Tender analyze failed: ${res.status}`);
    }

    return res.json() as Promise<TenderResult>;
  } finally {
    clearTimeout(timeout);
  }
}

export interface ProposalSection {
  title: string;
  status: "ai" | "filled" | "review";
  tokens: number;
}

export interface ProposalLogEntry {
  time: string;
  msg: string;
}

export interface ProposalResult {
  template: string;
  summary: string;
  meta: Record<string, string>;
  sections: ProposalSection[];
  log: ProposalLogEntry[];
  docx: string; // base64 encoded .docx
  pdf?: string; // base64 encoded .pdf
  md?: string; // markdown text
  mode?: string; // "placeholder" | "generative" | "clean"
  usage?: TokenUsage;
}

export async function generateProposal(
  template: File | null,
  contextFiles: File[],
  lang = "ru",
  params?: Record<string, string>,
): Promise<ProposalResult> {
  const form = new FormData();
  if (template) {
    form.append("template", template);
  }
  form.append("lang", lang);
  contextFiles.forEach((f) => form.append("context", f));
  if (params) {
    form.append("params", JSON.stringify(params));
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5 * 60 * 1000);

  try {
    const res = await fetch(`${BASE}/api/proposal/generate`, {
      method: "POST",
      headers: apiHeaders(),
      body: form,
      signal: controller.signal,
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error((body as Record<string, string>).error || `Proposal generate failed: ${res.status}`);
    }

    return res.json() as Promise<ProposalResult>;
  } finally {
    clearTimeout(timeout);
  }
}

/**
 * Streaming variant of generateProposal — uses the SSE endpoint that
 * keeps the TCP connection warm with periodic `progress` events while
 * the LLM is still running. Required for Opus-class generations on
 * Safari/Chrome where the browser fetch loop drops idle long
 * connections around the 60-120s mark.
 *
 * The promise resolves when the server emits `event: result` and
 * rejects on `event: error` or any transport-level failure.
 */
export async function generateProposalStream(
  template: File | null,
  contextFiles: File[],
  lang = "ru",
  params?: Record<string, string>,
  onProgress?: (event: { ts: number }) => void,
): Promise<ProposalResult> {
  const form = new FormData();
  if (template) {
    form.append("template", template);
  }
  form.append("lang", lang);
  contextFiles.forEach((f) => form.append("context", f));
  if (params) {
    form.append("params", JSON.stringify(params));
  }

  // Six-minute hard ceiling matching backend WriteTimeout. Each SSE
  // frame keeps the browser-side connection happy; the abort signal
  // only fires if the whole generation outpaces that ceiling.
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 6 * 60 * 1000);

  let res: Response;
  try {
    res = await fetch(`${BASE}/api/proposal/generate-stream`, {
      method: "POST",
      headers: apiHeaders(),
      body: form,
      signal: controller.signal,
    });
  } catch (err) {
    clearTimeout(timeout);
    throw err;
  }

  if (!res.ok || !res.body) {
    clearTimeout(timeout);
    throw new Error(`Proposal stream failed: ${res.status}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (value) buffer += decoder.decode(value, { stream: !done });

      // Drain complete events (\n\n delimited) from the buffer.
      while (true) {
        const sep = buffer.indexOf("\n\n");
        if (sep < 0) break;
        const raw = buffer.slice(0, sep);
        buffer = buffer.slice(sep + 2);

        let event = "message";
        let data = "";
        for (const line of raw.split("\n")) {
          if (line.startsWith("event: ")) event = line.slice("event: ".length);
          else if (line.startsWith("data: ")) data += line.slice("data: ".length);
        }
        if (!event) continue;

        if (event === "progress") {
          let parsed: { ts: number } = { ts: 0 };
          try {
            parsed = JSON.parse(data) as { ts: number };
          } catch {
            /* malformed progress frame — keep waiting */
          }
          onProgress?.(parsed);
          continue;
        }
        if (event === "error") {
          let msg = "Generation failed";
          try {
            const parsed = JSON.parse(data) as { error?: string };
            if (parsed.error) msg = parsed.error;
          } catch {
            msg = data || msg;
          }
          throw new Error(msg);
        }
        if (event === "result") {
          return JSON.parse(data) as ProposalResult;
        }
      }

      if (done) break;
    }
    throw new Error("Proposal stream ended without a result event");
  } finally {
    clearTimeout(timeout);
    try {
      reader.releaseLock();
    } catch {
      /* reader already released */
    }
  }
}

export async function checkConnection(overrides?: {
  provider?: string;
  apiKey?: string;
  url?: string;
  model?: string;
}): Promise<CheckResult> {
  // Start from apiHeaders() so the backend X-API-Key (dealSenseApiKey
  // from localStorage) is always attached. Without it the bot's
  // APIKeyAuth middleware returns 401 and the browser surfaces a
  // generic "Load failed" — not what the user expected to see.
  const headers: Record<string, string> = overrides
    ? {
        ...apiHeaders(),
        ...(overrides.provider && { "X-LLM-Provider": overrides.provider }),
        ...(overrides.apiKey && { "X-LLM-Key": overrides.apiKey }),
        ...(overrides.url && { "X-LLM-URL": overrides.url }),
        ...(overrides.model && { "X-LLM-Model": overrides.model }),
      }
    : apiHeaders();

  const res = await fetch(`${BASE}/api/llm/check`, {
    method: "POST",
    headers,
  });

  return res.json() as Promise<CheckResult>;
}

export async function listProviders(): Promise<ProvidersResult> {
  const res = await fetch(`${BASE}/api/llm/providers`, {
    method: "GET",
    headers: apiHeaders(),
  });

  if (!res.ok) {
    throw new Error(`List providers failed: ${res.status}`);
  }

  return res.json() as Promise<ProvidersResult>;
}

export interface ModelsResult {
  models: string[];
  error?: string;
}

export async function listModels(
  provider: string,
  apiKey: string,
  url: string,
): Promise<ModelsResult> {
  const res = await fetch(`${BASE}/api/llm/models`, {
    method: "GET",
    headers: {
      "X-LLM-Provider": provider,
      "X-LLM-Key": apiKey,
      "X-LLM-URL": url,
    },
  });

  return res.json() as Promise<ModelsResult>;
}

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
