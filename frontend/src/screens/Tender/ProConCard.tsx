import styles from "./ProConCard.module.css";

export interface ProConItem {
  t: string;
  d: string;
}

export interface ProConCardProps {
  tone: "go" | "no";
  title: string;
  items: ProConItem[];
}

export function ProConCard({ tone, title, items }: ProConCardProps) {
  const inkColor = tone === "go" ? "var(--go-ink)" : "var(--no-ink)";

  return (
    <div className={styles.container}>
      <div className={`${styles.header} ${styles[tone]}`}>
        <div
          className={`${styles.dot} ${tone === "go" ? styles.dotGo : styles.dotNo}`}
        />
        <span
          className={`t-h3 ${styles.headerTitle}`}
          style={{ color: inkColor }}
        >
          {title}
        </span>
        <span className={`t-mono ${styles.count}`}>{items.length}</span>
      </div>
      <div className={styles.body}>
        {items.map((item, i) => (
          <div key={i} className={styles.item}>
            <p className={`t-body ${styles.itemTitle}`} style={{ fontWeight: 500 }}>
              {item.t}
            </p>
            <p className={`t-small ${styles.itemDesc}`}>{item.d}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
