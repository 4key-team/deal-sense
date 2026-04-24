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
}

function apiHeaders(): Record<string, string> {
  const s = getItem<StoredSettings>("llm-settings", {});
  const headers: Record<string, string> = {};
  if (s.apiKey) headers["X-LLM-Key"] = s.apiKey;
  if (s.providerId) headers["X-LLM-Provider"] = s.providerId;
  if (s.url) headers["X-LLM-URL"] = s.url;
  if (s.model) headers["X-LLM-Model"] = s.model;
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

  const res = await fetch(`${BASE}/api/tender/analyze`, {
    method: "POST",
    headers: apiHeaders(),
    body: form,
  });

  if (!res.ok) {
    throw new Error(`Tender analyze failed: ${res.status}`);
  }

  return res.json() as Promise<TenderResult>;
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
  mode?: string; // "placeholder" | "generative" | "automarkup"
  usage?: TokenUsage;
}

export async function generateProposal(
  template: File,
  contextFiles: File[],
  lang = "ru",
  params?: Record<string, string>,
): Promise<ProposalResult> {
  const form = new FormData();
  form.append("template", template);
  form.append("lang", lang);
  contextFiles.forEach((f) => form.append("context", f));
  if (params) {
    form.append("params", JSON.stringify(params));
  }

  const res = await fetch(`${BASE}/api/proposal/generate`, {
    method: "POST",
    headers: apiHeaders(),
    body: form,
  });

  if (!res.ok) {
    throw new Error(`Proposal generate failed: ${res.status}`);
  }

  return res.json() as Promise<ProposalResult>;
}

export async function checkConnection(overrides?: {
  provider?: string;
  apiKey?: string;
  url?: string;
  model?: string;
}): Promise<CheckResult> {
  const headers: Record<string, string> = overrides
    ? {
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
