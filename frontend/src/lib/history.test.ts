import { describe, it, expect, beforeEach } from "vitest";
import { getHistory, addEntry, getScores, getTrend, getStats } from "./history";

describe("history", () => {
  beforeEach(() => localStorage.clear());

  it("returns empty history initially", () => {
    expect(getHistory()).toEqual([]);
  });

  it("adds an entry with auto-date", () => {
    addEntry({ score: 82, verdict: "go", fileName: "tender.pdf" });

    const history = getHistory();
    expect(history).toHaveLength(1);
    expect(history[0].score).toBe(82);
    expect(history[0].verdict).toBe("go");
    expect(history[0].fileName).toBe("tender.pdf");
    expect(history[0].date).toBeTruthy();
  });

  it("preserves order across multiple entries", () => {
    addEntry({ score: 70, verdict: "go", fileName: "a.pdf" });
    addEntry({ score: 40, verdict: "no-go", fileName: "b.pdf" });
    addEntry({ score: 90, verdict: "go", fileName: "c.pdf" });

    const history = getHistory();
    expect(history).toHaveLength(3);
    expect(history[0].score).toBe(70);
    expect(history[2].score).toBe(90);
  });

  it("limits to 50 entries", () => {
    for (let i = 0; i < 55; i++) {
      addEntry({ score: i, verdict: "go", fileName: `f${i}.pdf` });
    }
    expect(getHistory()).toHaveLength(50);
    // oldest entries removed
    expect(getHistory()[0].score).toBe(5);
  });

  it("getScores returns last N scores", () => {
    addEntry({ score: 10, verdict: "go", fileName: "a.pdf" });
    addEntry({ score: 20, verdict: "go", fileName: "b.pdf" });
    addEntry({ score: 30, verdict: "go", fileName: "c.pdf" });

    expect(getScores(2)).toEqual([20, 30]);
    expect(getScores()).toEqual([10, 20, 30]);
  });

  it("getTrend returns last N scores", () => {
    addEntry({ score: 50, verdict: "go", fileName: "a.pdf" });
    addEntry({ score: 60, verdict: "go", fileName: "b.pdf" });

    expect(getTrend(1)).toEqual([60]);
    expect(getTrend()).toEqual([50, 60]);
  });

  it("getStats computes go/watch/no correctly", () => {
    addEntry({ score: 80, verdict: "go", fileName: "a.pdf" });    // go (score>=60)
    addEntry({ score: 50, verdict: "go", fileName: "b.pdf" });    // watch (go but score<60)
    addEntry({ score: 30, verdict: "no-go", fileName: "c.pdf" }); // no

    const stats = getStats();
    expect(stats.go).toBe(1);
    expect(stats.watch).toBe(1);
    expect(stats.no).toBe(1);
    expect(stats.avgScore).toBe(53); // (80+50+30)/3 = 53.3 → 53
  });

  it("getStats returns zeros for empty history", () => {
    const stats = getStats();
    expect(stats).toEqual({ go: 0, watch: 0, no: 0, avgScore: 0 });
  });
});
