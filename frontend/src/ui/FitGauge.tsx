import styles from "./FitGauge.module.css";

type Tone = "go" | "no" | "warn";

interface FitGaugeProps {
  value: number;
  tone?: Tone;
}

const toneColors: Record<Tone, string> = {
  go: "var(--go)",
  no: "var(--no)",
  warn: "var(--warn)",
};

export function FitGauge({ value, tone = "go" }: FitGaugeProps) {
  return (
    <div className={styles.wrapper}>
      <div className={styles.track}>
        <div className={styles.fill} style={{ width: `${value}%`, background: toneColors[tone] }} />
      </div>
      <div className={styles.labels}>
        <span className="t-mono">0</span>
        <span className="t-mono">100</span>
      </div>
    </div>
  );
}
