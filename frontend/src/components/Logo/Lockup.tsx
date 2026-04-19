import React from "react";
import { MarkSeal, MarkWax, MarkPlumb, MarkPrism } from "./marks";
import { Wordmark } from "./Wordmark";

type Variant = "seal" | "wax" | "plumb" | "prism";
type Tone = "ink" | "white" | "mono";

interface LockupProps {
  variant?: Variant;
  size?: number;
  tone?: Tone;
}

const markMap: Record<Variant, React.ComponentType<{ size?: number; tone?: Tone }>> = {
  seal: MarkSeal,
  wax: MarkWax,
  plumb: MarkPlumb,
  prism: MarkPrism,
};

export function Lockup({ variant = "seal", size = 22, tone = "ink" }: LockupProps): React.ReactElement {
  const Mark = markMap[variant];
  const gap = size * 0.45;
  const wordmarkTone = tone === "mono" ? "ink" : tone;

  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap,
      }}
    >
      <Mark size={size} tone={tone} />
      <Wordmark size={size} tone={wordmarkTone} />
    </span>
  );
}
