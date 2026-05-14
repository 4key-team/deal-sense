// Prometheus text exposition format parser (version 0.0.4 subset, no histograms /
// summaries). Built for the admin metrics dashboard — narrow scope, dependency-
// free, exposes a flat shape that's easy to render.
//
// Spec we care about: https://prometheus.io/docs/instrumenting/exposition_formats/#text-format-details
// Only the lines emitted by adapter/metrics.Collector.Render are supported:
//   # HELP <name> <text>
//   # TYPE <name> counter|gauge
//   <name>{label="value",...} <number>

export type PromMetricType = "counter" | "gauge" | "unknown";

export interface PromSeries {
  labels: Record<string, string>;
  value: number;
}

export interface PromMetric {
  name: string;
  help: string;
  type: PromMetricType;
  series: PromSeries[];
}

const DATA_LINE = /^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{([^}]*)\})?\s+([-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)\s*$/;
const LABEL_PAIR = /([a-zA-Z_][a-zA-Z0-9_]*)="((?:[^"\\]|\\.)*)"/g;

export function parsePrometheus(text: string): PromMetric[] {
  if (!text) return [];

  const metricsByName = new Map<string, PromMetric>();
  const order: string[] = [];

  function getOrCreate(name: string): PromMetric {
    let m = metricsByName.get(name);
    if (!m) {
      m = { name, help: "", type: "unknown", series: [] };
      metricsByName.set(name, m);
      order.push(name);
    }
    return m;
  }

  for (const rawLine of text.split("\n")) {
    const line = rawLine.trim();
    if (!line) continue;

    if (line.startsWith("# HELP ")) {
      // # HELP <name> <help text>
      const rest = line.slice("# HELP ".length);
      const space = rest.indexOf(" ");
      if (space < 0) continue;
      const name = rest.slice(0, space);
      const help = rest.slice(space + 1);
      getOrCreate(name).help = help;
      continue;
    }

    if (line.startsWith("# TYPE ")) {
      const rest = line.slice("# TYPE ".length);
      const space = rest.indexOf(" ");
      if (space < 0) continue;
      const name = rest.slice(0, space);
      const type = rest.slice(space + 1).trim();
      if (type === "counter" || type === "gauge") {
        getOrCreate(name).type = type;
      }
      continue;
    }

    if (line.startsWith("#")) continue;

    const match = DATA_LINE.exec(line);
    if (!match) continue;
    const [, name, , labelsStr, valueStr] = match;
    const value = Number(valueStr);
    if (Number.isNaN(value)) continue;

    const labels: Record<string, string> = {};
    if (labelsStr) {
      LABEL_PAIR.lastIndex = 0;
      let pair: RegExpExecArray | null;
      while ((pair = LABEL_PAIR.exec(labelsStr)) !== null) {
        labels[pair[1]] = unescapeLabel(pair[2]);
      }
    }

    getOrCreate(name).series.push({ labels, value });
  }

  return order.map((n) => metricsByName.get(n)!);
}

function unescapeLabel(s: string): string {
  return s.replace(/\\(.)/g, (_, c: string) => {
    if (c === "n") return "\n";
    if (c === "\\") return "\\";
    if (c === '"') return '"';
    return c;
  });
}
