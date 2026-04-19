import React from 'react';

export interface MiniHistogramProps {
  data: number[];
  height?: number;
}

export function MiniHistogram({ data, height = 40 }: MiniHistogramProps): React.ReactElement {
  const gap = 3;
  const colCount = data.length;
  const max = Math.max(...data, 100);

  function barColor(v: number): string {
    if (v >= 70) return 'var(--go)';
    if (v >= 50) return 'var(--warn)';
    return 'var(--no)';
  }

  return (
    <svg
      width="100%"
      height={height}
      viewBox={`0 0 ${colCount * 14} ${height}`}
      preserveAspectRatio="none"
    >
      {data.map((v, i) => {
        const h = Math.max(2, (v / max) * (height - 4));
        return (
          <rect
            key={i}
            x={i * 14 + gap / 2}
            y={height - h}
            width={14 - gap}
            height={h}
            rx={1.5}
            fill={barColor(v)}
          />
        );
      })}
    </svg>
  );
}
