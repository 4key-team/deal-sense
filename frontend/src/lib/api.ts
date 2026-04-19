import { getItem } from "./storage";

const BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export interface TenderResult {
  verdict: string;
  risk: string;
  score: number;
  summary: string;
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

function apiHeaders(): Record<string, string> {
  const settings = getItem<{ apiKey?: string }>("llm-settings", {});
  const headers: Record<string, string> = {};
  if (settings.apiKey) {
    headers["X-API-Key"] = settings.apiKey;
  }
  return headers;
}

export async function analyzeTender(
  files: File[],
  companyProfile: string,
): Promise<TenderResult> {
  const form = new FormData();
  form.append("company_profile", companyProfile);
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

export async function generateProposal(
  template: File,
  params?: Record<string, string>,
): Promise<Blob> {
  const form = new FormData();
  form.append("template", template);
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

  return res.blob();
}

export async function checkConnection(): Promise<CheckResult> {
  const res = await fetch(`${BASE}/api/llm/check`, {
    method: "POST",
    headers: apiHeaders(),
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

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
