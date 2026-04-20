import { getItem, setItem } from "./storage";

const STORAGE_KEY = "tender-history";
const MAX_ENTRIES = 50;

export interface HistoryEntry {
  date: string;      // ISO date
  score: number;     // 0-100
  verdict: string;   // "go" | "no-go"
  fileName: string;  // first uploaded file name
}

export function getHistory(): HistoryEntry[] {
  return getItem<HistoryEntry[]>(STORAGE_KEY, []);
}

export function addEntry(entry: Omit<HistoryEntry, "date">): void {
  const history = getHistory();
  history.push({
    ...entry,
    date: new Date().toISOString(),
  });
  // keep last MAX_ENTRIES
  if (history.length > MAX_ENTRIES) {
    history.splice(0, history.length - MAX_ENTRIES);
  }
  setItem(STORAGE_KEY, history);
}

/** Last N scores for histogram */
export function getScores(n = 12): number[] {
  return getHistory().slice(-n).map((e) => e.score);
}

/** Last N scores for sparkline (trend over time) */
export function getTrend(n = 8): number[] {
  return getHistory().slice(-n).map((e) => e.score);
}

/** Stats from last N entries */
export function getStats(n = 12): { go: number; watch: number; no: number; avgScore: number } {
  const entries = getHistory().slice(-n);
  if (entries.length === 0) return { go: 0, watch: 0, no: 0, avgScore: 0 };

  let go = 0;
  let watch = 0;
  let no = 0;
  let total = 0;

  for (const e of entries) {
    total += e.score;
    if (e.verdict === "go" && e.score >= 60) go++;
    else if (e.verdict === "go") watch++;
    else no++;
  }

  return {
    go,
    watch,
    no,
    avgScore: Math.round(total / entries.length),
  };
}
