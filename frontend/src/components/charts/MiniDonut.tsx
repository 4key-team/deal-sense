import React from 'react';

export interface MiniDonutProps {
  met: number;
  partial: number;
  miss: number;
  size?: number;
}

export function MiniDonut({ met, partial, miss, size = 100 }: MiniDonutProps): React.ReactElement {
  const r = 34;
  const C = 2 * Math.PI * r;
  const total = met + partial + miss;

  function seg(n: number): number {
    if (total === 0) return 0;
    return (n / total) * C;
  }

  const segMet = seg(met);
  const segPartial = seg(partial);

  const circleProps = {
    cx: 50,
    cy: 50,
    r,
    fill: 'none',
    strokeWidth: 14,
  } as const;

  return (
    <svg width={size} height={size} viewBox="0 0 100 100">
      {/* Background track */}
      <circle {...circleProps} stroke="var(--surface-3)" />

      {/* Met segment */}
      <circle
        {...circleProps}
        stroke="var(--go)"
        strokeDasharray={`${segMet} ${C - segMet}`}
        strokeDashoffset={0}
        transform="rotate(-90 50 50)"
      />

      {/* Partial segment */}
      <circle
        {...circleProps}
        stroke="var(--warn)"
        strokeDasharray={`${segPartial} ${C - segPartial}`}
        strokeDashoffset={`-${segMet}`}
        transform="rotate(-90 50 50)"
      />

      {/* Miss segment */}
      <circle
        {...circleProps}
        stroke="var(--no)"
        strokeDasharray={`${seg(miss)} ${C - seg(miss)}`}
        strokeDashoffset={`-${segMet + segPartial}`}
        transform="rotate(-90 50 50)"
      />

      {/* Center text */}
      <text
        x={50}
        y={47}
        textAnchor="middle"
        dominantBaseline="middle"
        fontSize={18}
        fontFamily="var(--font-mono, monospace)"
        fill="currentColor"
      >
        {met}/{total}
      </text>
      <text
        x={50}
        y={60}
        textAnchor="middle"
        dominantBaseline="middle"
        fontSize={9}
        fontFamily="var(--font-sans, sans-serif)"
        fill="currentColor"
        opacity={0.6}
      >
        ЗАКРЫТО
      </text>
    </svg>
  );
}
