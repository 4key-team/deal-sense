import React from "react";

interface WordmarkProps {
  size?: number;
  tone?: "ink" | "white";
}

export function Wordmark({ size = 22, tone = "ink" }: WordmarkProps): React.ReactElement {
  const inkColor = tone === "white" ? "#fff" : "var(--ink-1)";
  const accentColor = tone === "white" ? "rgba(255,255,255,0.85)" : "var(--brand)";

  return (
    <span
      style={{
        fontFamily: "var(--font-serif)",
        fontWeight: 500,
        fontSize: size,
        letterSpacing: "-0.005em",
        lineHeight: 1,
        color: inkColor,
        display: "inline-flex",
        alignItems: "baseline",
      }}
    >
      Deal
      <span style={{ color: accentColor }}>Sense</span>
    </span>
  );
}
