import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { Card } from "./Card";

describe("Card", () => {
  it("renders children", () => {
    const { getByText } = render(<Card>Content</Card>);
    expect(getByText("Content")).toBeInTheDocument();
  });

  it("applies default padding", () => {
    const { container } = render(<Card>X</Card>);
    expect(container.firstElementChild).toHaveStyle({ padding: "20px" });
  });

  it("applies tight padding", () => {
    const { container } = render(<Card tight>X</Card>);
    expect(container.firstElementChild).toHaveStyle({ padding: "14px" });
  });

  it("applies custom padding", () => {
    const { container } = render(<Card padding={32}>X</Card>);
    expect(container.firstElementChild).toHaveStyle({ padding: "32px" });
  });
});
