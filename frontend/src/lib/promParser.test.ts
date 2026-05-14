import { describe, it, expect } from "vitest";
import { parsePrometheus, type PromMetric } from "./promParser";

describe("parsePrometheus", () => {
  describe("empty / blank input", () => {
    it.each([
      { name: "empty string", input: "", expected: [] as PromMetric[] },
      { name: "whitespace only", input: "\n\n   \n", expected: [] as PromMetric[] },
      { name: "comments only", input: "# unrelated\n#unrelated2\n", expected: [] as PromMetric[] },
    ])("returns [] for $name", ({ input, expected }) => {
      expect(parsePrometheus(input)).toEqual(expected);
    });
  });

  describe("data line shapes", () => {
    it.each([
      {
        name: "single counter with HELP and TYPE",
        input: `# HELP dealsense_requests_total Total HTTP requests.
# TYPE dealsense_requests_total counter
dealsense_requests_total{path="/api/x",status="200"} 42
`,
        name0: "dealsense_requests_total",
        type0: "counter",
        help0: "Total HTTP requests.",
        series0: [{ labels: { path: "/api/x", status: "200" }, value: 42 }],
      },
      {
        name: "metric line without labels",
        input: `# TYPE up counter
up 1
`,
        name0: "up",
        type0: "counter",
        help0: "",
        series0: [{ labels: {}, value: 1 }],
      },
      {
        name: "tolerates extra whitespace",
        input: `# TYPE m counter
m{path="/x"}    99
`,
        name0: "m",
        type0: "counter",
        help0: "",
        series0: [{ labels: { path: "/x" }, value: 99 }],
      },
    ])("parses $name", ({ input, name0, type0, help0, series0 }) => {
      const out = parsePrometheus(input);
      expect(out).toHaveLength(1);
      expect(out[0]).toMatchObject({
        name: name0,
        type: type0,
        help: help0,
        series: series0,
      });
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

  describe("value parsing", () => {
    it.each([
      { name: "integer 1", line: 'g{n="a"} 1', expected: 1 },
      { name: "zero", line: 'g{n="b"} 0', expected: 0 },
      { name: "float pi", line: 'g{n="c"} 3.14', expected: 3.14 },
      { name: "negative", line: 'g{n="d"} -2.5', expected: -2.5 },
      { name: "scientific", line: 'g{n="e"} 1.5e3', expected: 1500 },
    ])("parses $name as $expected", ({ line, expected }) => {
      const text = `# TYPE g gauge\n${line}\n`;
      const out = parsePrometheus(text);
      expect(out[0].series[0].value).toBe(expected);
    });
  });

  describe("malformed input tolerance", () => {
    it.each([
      {
        name: "skips garbage lines but keeps valid ones",
        input: `# TYPE a counter
a{x="ok"} 1
garbage_line_with_no_value
a{x="also-ok"} 2
`,
        expectedCount: 2,
      },
      {
        name: "drops NaN value lines",
        input: `# TYPE a counter
a{x="ok"} 1
a{x="bad"} not_a_number
a{x="ok2"} 2
`,
        expectedCount: 2,
      },
    ])("$name", ({ input, expectedCount }) => {
      const out = parsePrometheus(input);
      expect(out[0].series).toHaveLength(expectedCount);
    });
  });

  it("unescapes backslash and quote in label values", () => {
    const text = `# TYPE m counter
m{path="a\\\\b\\"c"} 1
`;
    const out = parsePrometheus(text);
    expect(out[0].series[0].labels).toEqual({ path: 'a\\b"c' });
  });

  it("creates a metric entry even if HELP comes without TYPE (type='unknown')", () => {
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
