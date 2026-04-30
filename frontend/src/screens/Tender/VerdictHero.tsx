import { Card } from "../../ui/Card";
import { FitGauge } from "../../ui/FitGauge";
import { useI18n } from "../../providers/useI18n";
import type { TenderData } from "../../mocks/tender";
import styles from "./VerdictHero.module.css";

export interface VerdictHeroProps {
  data: TenderData;
  verdict: "go" | "no";
  setVerdict: (v: "go" | "no") => void;
}

export function VerdictHero({ data, verdict, setVerdict }: VerdictHeroProps) {
  const { t } = useI18n();

  const inkColor = verdict === "go" ? "var(--go-ink)" : "var(--no-ink)";
  const borderColor = verdict === "go" ? "var(--go-line)" : "var(--no-line)";
  const innerClass =
    verdict === "go"
      ? `${styles.inner} ${styles.innerGo}`
      : `${styles.inner} ${styles.innerNo}`;

  return (
    <Card
      padding={0}
      className={styles.card}
      style={{ borderColor }}
    >
      <div className={innerClass}>
        <div className={styles.topArea}>
          <div className={styles.toggle}>
            <button
              className={`${styles.toggleBtn} ${verdict === "go" ? styles.toggleBtnGoActive : ""}`}
              onClick={() => setVerdict("go")}
            >
              {t.tender.toggle_go}
            </button>
            <button
              className={`${styles.toggleBtn} ${verdict === "no" ? styles.toggleBtnNoActive : ""}`}
              onClick={() => setVerdict("no")}
            >
              {t.tender.toggle_no}
            </button>
          </div>
        </div>

        <div className={styles.content}>
          <div className={styles.left}>
            <span
              className={`t-micro ${styles.fitLabel}`}
              style={{ color: inkColor }}
            >
              {t.tender.fit} · {data.fit}%
            </span>
            <p className={styles.bigTitle} style={{ color: inkColor }}>
              {verdict === "go" ? t.tender.verdict_go : t.tender.verdict_no}
            </p>
          </div>

          <div className={styles.right}>
            <p className={`t-h3 ${styles.subtitle}`}>
              {verdict === "go" ? t.tender.verdict_go_sub : t.tender.verdict_no_sub}
            </p>
            <FitGauge value={data.fit} tone={verdict === "go" ? "go" : "no"} />
          </div>
        </div>
      </div>
    </Card>
  );
}
