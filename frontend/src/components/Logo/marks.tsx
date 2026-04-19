import React from "react";

type Tone = "ink" | "white" | "mono";

interface MarkProps {
  size?: number;
  tone?: Tone;
}

function toneColors(tone: string) {
  if (tone === "white") return { fg: "#fff", fgSoft: "rgba(255,255,255,0.6)", softOpacity: 1 };
  if (tone === "mono") return { fg: "currentColor", fgSoft: "currentColor", softOpacity: 0.55 };
  return { fg: "var(--brand-ink)", fgSoft: "var(--brand)", softOpacity: 1 };
}

export function MarkSeal({ size = 48, tone = "ink" }: MarkProps): React.ReactElement {
  const { fg, fgSoft, softOpacity } = toneColors(tone);
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M14 2 H34 L46 14 V34 L34 46 H14 L2 34 V14 Z"
        stroke={fg}
        strokeWidth={1.6}
        strokeLinejoin="miter"
      />
      <circle
        cx={24}
        cy={24}
        r={13.5}
        stroke={fgSoft}
        strokeOpacity={softOpacity * 0.55}
        strokeWidth={1}
        strokeDasharray="1 2.2"
      />
      <path
        d="M24 4 V7 M24 41 V44 M4 24 H7 M41 24 H44"
        stroke={fg}
        strokeWidth={1.4}
        strokeLinecap="square"
      />
      <path
        d="M18 16 H26.5 a4.5 4.5 0 0 1 0 9 H18 V16 Z M18 23.5 H27.2 a4.7 4.7 0 0 1 0 9 H18 V23.5 Z"
        stroke={fg}
        strokeWidth={1.7}
        strokeLinejoin="miter"
        fill="none"
      />
      <path
        d="M14 24 L34 24"
        stroke={fg}
        strokeWidth={1.7}
        strokeLinecap="square"
      />
    </svg>
  );
}

export function MarkWax({ size = 48, tone = "ink" }: MarkProps): React.ReactElement {
  const { fg, fgSoft, softOpacity } = toneColors(tone);
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx={24} cy={24} r={22} stroke={fg} strokeWidth={1.5} />
      <circle
        cx={24}
        cy={24}
        r={17}
        stroke={fgSoft}
        strokeOpacity={softOpacity * 0.45}
        strokeWidth={1}
      />
      <path
        d="M24 9 L35 24 L24 39 L13 24 Z"
        stroke={fg}
        strokeWidth={1.6}
        strokeLinejoin="miter"
      />
      <path d="M24 9 L13 24 L24 39 Z" fill={fg} />
      <path
        d="M24 9 L24 39 M28 12.5 L28 35.5 M32 18 L32 30"
        stroke={fgSoft}
        strokeOpacity={softOpacity * 0.45}
        strokeWidth={0.9}
      />
    </svg>
  );
}

export function MarkPlumb({ size = 48, tone = "ink" }: MarkProps): React.ReactElement {
  const { fg, fgSoft, softOpacity } = toneColors(tone);
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M6 10 L10 6 H38 L42 10 V38 L38 42 H10 L6 38 Z"
        stroke={fg}
        strokeWidth={1.5}
        strokeLinejoin="miter"
      />
      <circle cx={24} cy={8} r={1.5} fill={fg} />
      <path d="M24 8 V32" stroke={fg} strokeWidth={1.4} />
      <path d="M20 32 H28 L24 40 Z" fill={fg} />
      <path
        d="M10 28 H17 M31 28 H38"
        stroke={fgSoft}
        strokeOpacity={softOpacity * 0.5}
        strokeWidth={1.2}
      />
      <path
        d="M8 16 H10 M8 22 H10 M8 28 H10 M8 34 H10 M38 16 H40 M38 22 H40 M38 28 H40 M38 34 H40"
        stroke={fgSoft}
        strokeOpacity={softOpacity * 0.55}
        strokeWidth={1}
      />
    </svg>
  );
}

export function MarkPrism({ size = 48, tone = "ink" }: MarkProps): React.ReactElement {
  const { fg, fgSoft, softOpacity } = toneColors(tone);
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M14 10 L38 24 L14 38 Z"
        stroke={fg}
        strokeWidth={1.6}
        strokeLinejoin="miter"
      />
      <path d="M2 24 H14" stroke={fg} strokeWidth={1.6} strokeLinecap="square" />
      <path
        d="M3 21 H13 M3 27 H13"
        stroke={fgSoft}
        strokeOpacity={softOpacity * 0.5}
        strokeWidth={1}
      />
      <path d="M38 24 L46 18" stroke={fg} strokeWidth={1.6} strokeLinecap="square" />
      <path
        d="M38 24 L46 30"
        stroke={fgSoft}
        strokeOpacity={softOpacity * 0.45}
        strokeWidth={1.2}
        strokeLinecap="square"
      />
      <circle cx={24} cy={24} r={2.2} fill={fg} />
    </svg>
  );
}
