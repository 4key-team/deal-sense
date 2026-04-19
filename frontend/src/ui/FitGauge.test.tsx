import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { FitGauge } from "./FitGauge";

describe("FitGauge", () => {
  it("renders with correct width percentage", () => {
    const { container } = render(<FitGauge value={75} tone="go" />);
    const fill = container.querySelector('[class*="fill"]') as HTMLElement;
    expect(fill.style.width).toBe("75%");
  });

  it("shows 0 and 100 labels", () => {
    const { container } = render(<FitGauge value={50} />);
    expect(container.textContent).toContain("0");
    expect(container.textContent).toContain("100");
  });
});
