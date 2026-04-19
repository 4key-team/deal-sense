import { describe, it, expect } from "vitest";
import { getTenderData, getRequirements, getFiles } from "./tender";
import { getSections, getContext, getMeta, getLog } from "./proposal";

describe("tender mocks", () => {
  for (const lang of ["ru", "en"] as const) {
    for (const verdict of ["go", "no"] as const) {
      it(`getTenderData(${verdict}, ${lang})`, () => {
        const data = getTenderData(verdict, lang);
        expect(data.fit).toBeGreaterThan(0);
        expect(data.pros.length).toBeGreaterThan(0);
        expect(data.tone).toBe(verdict);
      });
    }

    it(`getRequirements(${lang})`, () => {
      expect(getRequirements(lang).length).toBe(8);
    });

    it(`getFiles(${lang})`, () => {
      expect(getFiles(lang).length).toBe(3);
    });
  }
});

describe("proposal mocks", () => {
  for (const lang of ["ru", "en"] as const) {
    it(`getSections(${lang})`, () => {
      expect(getSections(lang).length).toBe(8);
    });

    it(`getContext(${lang})`, () => {
      const t = { context_brief: "brief", context_cases: "cases", context_prices: "prices" };
      expect(getContext(lang, t).length).toBe(4);
    });

    it(`getMeta(${lang})`, () => {
      const meta = getMeta(lang);
      expect(meta.client).toBeTruthy();
      expect(meta.price).toBeTruthy();
    });

    it(`getLog(${lang})`, () => {
      expect(getLog(lang).length).toBe(5);
    });
  }
});
