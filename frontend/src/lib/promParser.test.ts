import { describe, it, expect } from "vitest";
import { parsePrometheus } from "./promParser";

describe("parsePrometheus", () => {
  it("returns empty array for empty input", () => {
    expect(parsePrometheus("")).toEqual([]);
  });

  it("ignores blank lines and stray comments", () => {
    const out = parsePrometheus("\n\n# unrelated comment\n\n");
    expect(out).toEqual([]);
  });

  it("parses a single counter with HELP and TYPE", () => {
    const text = `# HELP dealsense_requests_total Total HTTP requests.
# TYPE dealsense_requests_total counter
dealsense_requests_total{path="/api/x",status="200"} 42
`;
    const out = parsePrometheus(text);
    expect(out).toHaveLength(1);
    expect(out[0]).toMatchObject({
      name: "dealsense_requests_total",
      help: "Total HTTP requests.",
      type: "counter",
      series: [
        { labels: { path: "/api/x", status: "200" }, value: 42 },
      ],
    });
  });

  it("accumulates multiple series under the same metric", () => {
    const text = `# TYPE dealsense_requests_total counter
dealsense_requests_total{path="/a",status="200"} 1
dealsense_requests_total{path="/a",status="500"} 2
dealsense_requests_total{path="/b",status="200"} 7
`;
    const out = parsePrometheus(text);
    expect(out).toHaveLength(1);
    expect(out[0].series).toHaveLength(3);
    expect(out[0].series.map((s) => s.value)).toEqual([1, 2, 7]);
  });

  it("parses multiple metrics separated by HELP/TYPE headers", () => {
    const text = `# HELP a A counter.
# TYPE a counter
a{x="1"} 10
# HELP b A gauge.
# TYPE b gauge
b{y="2"} 0.5
`;
    const out = parsePrometheus(text);
    expect(out.map((m) => m.name)).toEqual(["a", "b"]);
    expect(out[0].type).toBe("counter");
    expect(out[1].type).toBe("gauge");
  });

  it("parses gauge values including floats and zero", () => {
    const text = `# TYPE g gauge
g{path="/x",level="modify"} 1
g{path="/y",level="safe_read"} 0
g{n="float"} 3.14
`;
    const out = parsePrometheus(text);
    expect(out[0].series.map((s) => s.value)).toEqual([1, 0, 3.14]);
  });

  it("parses metric lines without labels", () => {
    const text = `# TYPE up counter
up 1
`;
    const out = parsePrometheus(text);
    expect(out[0].series).toEqual([{ labels: {}, value: 1 }]);
  });

  it("unescapes backslash and quote in label values", () => {
    const text = `# TYPE m counter
m{path="a\\\\b\\"c"} 1
`;
    const out = parsePrometheus(text);
    expect(out[0].series[0].labels).toEqual({ path: 'a\\b"c' });
  });

  it("tolerates extra whitespace between tokens", () => {
    const text = `# TYPE m counter
m{path="/x"}    99
`;
    const out = parsePrometheus(text);
    expect(out[0].series[0]).toEqual({ labels: { path: "/x" }, value: 99 });
  });

  it("skips malformed data lines without losing valid ones", () => {
    const text = `# TYPE a counter
a{x="ok"} 1
garbage_line_with_no_value
a{x="also-ok"} 2
`;
    const out = parsePrometheus(text);
    expect(out[0].series).toHaveLength(2);
  });

  it("creates a metric entry even if HELP comes without TYPE", () => {
    // Real Prometheus exposition always emits TYPE; the parser must still
    // surface help-only entries with type='unknown' rather than dropping them.
    const text = `# HELP orphan No type line.
orphan 5
`;
    const out = parsePrometheus(text);
    expect(out[0].name).toBe("orphan");
    expect(out[0].help).toBe("No type line.");
    expect(out[0].type).toBe("unknown");
  });
});
