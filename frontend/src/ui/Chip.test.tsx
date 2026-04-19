import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Chip } from "./Chip";

describe("Chip", () => {
  it("renders text content", () => {
    render(<Chip>Ready</Chip>);
    expect(screen.getByText("Ready")).toBeInTheDocument();
  });

  it("renders icon when provided", () => {
    render(<Chip icon={<span data-testid="icon" />}>Status</Chip>);
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });
});
