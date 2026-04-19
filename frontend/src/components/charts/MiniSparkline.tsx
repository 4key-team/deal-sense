import React, { useId } from 'react';

export interface MiniSparklineProps {
  data: number[];
  height?: number;
}

export function MiniSparkline({ data, height = 40 }: MiniSparklineProps): React.ReactElement {
  const gradientId = useId();
  const W = 200;
  const H = height;

  if (data.length < 2) {
    return <svg width="100%" height={H} viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" />;
  }

  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max === min ? 1 : max - min;

  const points: Array<[number, number]> = data.map((v, i) => [
    (i / (data.length - 1)) * W,
    H - 8 - ((v - min) / range) * (H - 16),
  ]);

  const polylinePoints = points.map(([x, y]) => `${x},${y}`).join(' ');
  const polygonPoints = `0,${H} ${polylinePoints} ${W},${H}`;

  const [lastX, lastY] = points[points.length - 1];

  return (
    <svg
      width="100%"
      height={H}
      viewBox={`0 0 ${W} ${H}`}
      preserveAspectRatio="none"
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--brand)" stopOpacity={0.25} />
          <stop offset="100%" stopColor="var(--brand)" stopOpacity={0} />
        </linearGradient>
      </defs>

      {/* Fill area */}
      <polygon
        points={polygonPoints}
        fill={`url(#${gradientId})`}
      />

      {/* Line */}
      <polyline
        points={polylinePoints}
        fill="none"
        stroke="var(--brand)"
        strokeWidth={1.8}
        strokeLinecap="round"
        strokeLinejoin="round"
      />

      {/* Last point dot */}
      <circle
        cx={lastX}
        cy={lastY}
        r={3}
        fill="var(--brand)"
        stroke="var(--surface)"
        strokeWidth={1.5}
      />
    </svg>
  );
}
