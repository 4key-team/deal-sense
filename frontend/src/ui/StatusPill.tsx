import styles from "./StatusPill.module.css";

type Status = "ok" | "err" | "warn";

interface StatusPillProps {
  provider: string;
  model: string;
  status?: Status;
  onClick?: () => void;
}

const statusColors: Record<Status, string> = {
  ok: "var(--go)",
  err: "var(--no)",
  warn: "var(--warn)",
};

export function StatusPill({ provider, model, status = "ok", onClick }: StatusPillProps) {
  const color = statusColors[status];
  return (
    <button className={styles.pill} onClick={onClick} aria-label={`${provider} ${model}`}>
      <span className={styles.dot} style={{ background: color }}>
        <span className={styles.halo} style={{ background: color }} />
      </span>
      <span className={styles.provider}>{provider}</span>
      <span className={styles.sep}>&middot;</span>
      <span className={styles.model}>{model}</span>
    </button>
  );
}
