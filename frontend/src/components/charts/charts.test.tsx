import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { MiniHistogram } from "./MiniHistogram";
import { MiniDonut } from "./MiniDonut";
import { MiniSparkline } from "./MiniSparkline";

describe("MiniHistogram", () => {
  it("renders bars for data", () => {
    const { container } = render(<MiniHistogram data={[80, 40, 60]} />);
    expect(container.querySelectorAll("rect").length).toBe(3);
  });
});

describe("MiniDonut", () => {
  it("renders with data", () => {
    const { container } = render(<MiniDonut met={5} partial={1} miss={2} />);
    expect(container.querySelector("svg")).toBeTruthy();
  });

  it("handles zero total", () => {
    const { container } = render(<MiniDonut met={0} partial={0} miss={0} />);
    expect(container.querySelector("svg")).toBeTruthy();
  });
});

describe("MiniSparkline", () => {
  it("renders with data", () => {
    const { container } = render(<MiniSparkline data={[10, 20, 30, 40]} />);
    expect(container.querySelector("polyline")).toBeTruthy();
  });

  it("handles single data point", () => {
    const { container } = render(<MiniSparkline data={[42]} />);
    expect(container.querySelector("svg")).toBeTruthy();
  });

  it("handles flat data (all same values)", () => {
    const { container } = render(<MiniSparkline data={[50, 50, 50]} />);
    expect(container.querySelector("polyline")).toBeTruthy();
  });

  it("handles empty data", () => {
    const { container } = render(<MiniSparkline data={[]} />);
    expect(container.querySelector("svg")).toBeTruthy();
  });
});
