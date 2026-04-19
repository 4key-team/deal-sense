import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import {
  SettingsIcon, EyeIcon, ChevIcon, CheckIcon, XIcon, MinusIcon,
  DocIcon, DownloadIcon, PlusIcon, SparkIcon, TrendIcon, SunIcon, MoonIcon,
} from "./Icons";

describe("Icons", () => {
  const icons = [
    ["SettingsIcon", SettingsIcon],
    ["CheckIcon", CheckIcon],
    ["XIcon", XIcon],
    ["MinusIcon", MinusIcon],
    ["DocIcon", DocIcon],
    ["DownloadIcon", DownloadIcon],
    ["PlusIcon", PlusIcon],
    ["SparkIcon", SparkIcon],
    ["TrendIcon", TrendIcon],
    ["SunIcon", SunIcon],
    ["MoonIcon", MoonIcon],
  ] as const;

  for (const [name, Icon] of icons) {
    it(`renders ${name}`, () => {
      const { container } = render(<Icon />);
      expect(container.querySelector("svg")).toBeTruthy();
    });
  }

  it("renders EyeIcon open", () => {
    const { container } = render(<EyeIcon />);
    expect(container.querySelector("svg")).toBeTruthy();
  });

  it("renders EyeIcon closed", () => {
    const { container } = render(<EyeIcon off />);
    expect(container.querySelectorAll("path").length).toBeGreaterThan(1);
  });

  it("renders ChevIcon all directions", () => {
    for (const dir of ["down", "up", "right", "left"] as const) {
      const { container } = render(<ChevIcon dir={dir} />);
      expect(container.querySelector("svg")).toBeTruthy();
    }
  });
});
