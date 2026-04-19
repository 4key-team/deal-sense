import { useState } from "react";
import { useI18n } from "../../providers/I18nProvider";
import { Card } from "../../ui/Card";
import { SectionLabel } from "../../ui/SectionLabel";
import { Button } from "../../ui/Button";
import {
  CheckIcon,
  XIcon,
  MinusIcon,
  DocIcon,
  DownloadIcon,
  SparkIcon,
  TrendIcon,
} from "../../icons/Icons";
import { MiniHistogram } from "../../components/charts/MiniHistogram";
import { MiniSparkline } from "../../components/charts/MiniSparkline";
import {
  getTenderData,
  getRequirements,
  getFiles,
  fitHistory,
  winTrend,
} from "../../mocks/tender";
import type { TenderRequirement } from "../../mocks/tender";
import { VerdictHero } from "./VerdictHero";
import { ProConCard } from "./ProConCard";
import styles from "./TenderReport.module.css";

type StatusConfig = {
  bg: string;
  border: string;
  icon: React.ReactNode;
};

function getStatusConfig(status: TenderRequirement["status"]): StatusConfig {
  if (status === "met") {
    return {
      bg: "var(--go-wash)",
      border: "var(--go-line)",
      icon: (
        <span style={{ color: "var(--go)" }}>
          <CheckIcon />
        </span>
      ),
    };
  }
  if (status === "partial") {
    return {
      bg: "var(--warn-wash)",
      border: "var(--warn-line)",
      icon: (
        <span style={{ color: "var(--warn)" }}>
          <MinusIcon />
        </span>
      ),
    };
  }
  return {
    bg: "var(--no-wash)",
    border: "var(--no-line)",
    icon: (
      <span style={{ color: "var(--no)" }}>
        <XIcon />
      </span>
    ),
  };
}

export function TenderReport() {
  const { lang, t } = useI18n();
  const [verdict, setVerdict] = useState<"go" | "no">("go");

  const data = getTenderData(verdict, lang);
  const requirements = getRequirements(lang);
  const files = getFiles(lang);

  const effortText =
    verdict === "go"
      ? lang === "ru"
        ? "~6 часов"
        : "~6 hours"
      : lang === "ru"
        ? "~14 часов"
        : "~14 hours";

  const metCount = requirements.filter((r) => r.status === "met").length;
  const partialCount = requirements.filter((r) => r.status === "partial").length;
  const missCount = requirements.filter((r) => r.status === "miss").length;

  return (
    <div className={`screen-enter ${styles.screen}`}>
      {/* Left column */}
      <div className={styles.left}>
        <VerdictHero data={data} verdict={verdict} setVerdict={setVerdict} />

        {/* Pro/Con grid */}
        <div className={styles.proConGrid}>
          <ProConCard tone="go" title={t.tender.pros} items={data.pros} />
          <ProConCard tone="no" title={t.tender.cons} items={data.cons} />
        </div>

        {/* Requirements */}
        <div>
          <div className={styles.requirementsLabel}>
            <SectionLabel>
              {t.tender.requirements} · {requirements.length}
            </SectionLabel>
          </div>
          <Card padding={0} style={{ overflow: "hidden" }}>
            <div className={styles.reqGrid}>
              {requirements.map((req, i) => {
                const cfg = getStatusConfig(req.status);
                const statusText =
                  req.status === "met"
                    ? t.tender.req_met
                    : req.status === "partial"
                      ? t.tender.req_partial
                      : t.tender.req_miss;

                return (
                  <div key={i} className={styles.reqRow}>
                    <div
                      className={styles.statusCircle}
                      style={{
                        background: cfg.bg,
                        border: `1px solid ${cfg.border}`,
                      }}
                    >
                      {cfg.icon}
                    </div>
                    <span
                      className={`t-small ${styles.reqLabel}`}
                      style={{ fontWeight: 500, color: "var(--ink-1)" }}
                    >
                      {req.label}
                    </span>
                    <span className={styles.reqStatus}>{statusText}</span>
                  </div>
                );
              })}
            </div>
          </Card>
        </div>
      </div>

      {/* Right column */}
      <div className={styles.right}>
        {/* Files card */}
        <Card padding={18}>
          <span className={`t-micro ${styles.cardHeader}`}>{t.tender.files}</span>
          <div className={styles.fileList}>
            {files.map((file, i) => (
              <div key={i} className={styles.fileRow}>
                <div className={styles.docIconBox}>
                  <DocIcon />
                </div>
                <span className={styles.fileName}>{file.name}</span>
                <span className={styles.fileMeta}>{file.size}</span>
              </div>
            ))}
          </div>
        </Card>

        {/* Effort card */}
        <Card padding={18}>
          <span className={`t-micro ${styles.cardHeader}`}>{t.tender.effort}</span>
          <span className={styles.effortValue}>{effortText}</span>
          <p className="t-small dim">
            {verdict === "go"
              ? lang === "ru"
                ? "Подготовка заявки при наличии всех материалов"
                : "Bid preparation with all materials on hand"
              : lang === "ru"
                ? "Заявка потребует значительных усилий из-за пробелов"
                : "Bid requires significant effort due to gaps"}
          </p>
        </Card>

        {/* Recent activity card */}
        <Card padding={18}>
          <span className={`t-micro ${styles.cardHeader}`}>{t.tender.recent}</span>
          <span className={styles.bigNumber}>63</span>
          <p className="t-small dim">
            {lang === "ru" ? "средний fit" : "avg fit"}
          </p>
          <MiniHistogram data={fitHistory} height={40} />
          <div className={styles.legend}>
            <div className={styles.legendItem}>
              <div
                className={styles.legendDot}
                style={{ background: "var(--go)" }}
              />
              GO {metCount}
            </div>
            <div className={styles.legendItem}>
              <div
                className={styles.legendDot}
                style={{ background: "var(--warn)" }}
              />
              {lang === "ru" ? "watch" : "watch"} {partialCount}
            </div>
            <div className={styles.legendItem}>
              <div
                className={styles.legendDot}
                style={{ background: "var(--no)" }}
              />
              NO {missCount}
            </div>
          </div>
        </Card>

        {/* Trend card */}
        <Card padding={18}>
          <span className={`t-micro ${styles.cardHeader}`}>{t.tender.trend}</span>
          <div className={styles.trendHeader}>
            <span className={styles.trendValue}>+37</span>
            <span style={{ color: "var(--go)" }}>
              <TrendIcon />
            </span>
            <span className={styles.trendText}>
              {lang === "ru" ? "+37% за 8 недель" : "+37% over 8 weeks"}
            </span>
          </div>
          <MiniSparkline data={winTrend} height={40} />
          <span className={styles.trendFooter}>
            {lang === "ru"
              ? "winrate растёт · последние 8 тендеров"
              : "win rate rising · last 8 tenders"}
          </span>
        </Card>

        {/* Action buttons */}
        <div className={styles.actions}>
          {verdict === "go" ? (
            <>
              <Button variant="brand" size="lg" icon={<SparkIcon />}>
                {t.tender.actions_prep}
              </Button>
              <Button variant="secondary" size="lg" icon={<DownloadIcon />}>
                {t.tender.actions_export}
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" size="lg">
                {t.tender.actions_draft}
              </Button>
              <Button variant="ghost" size="lg" icon={<DownloadIcon />}>
                {t.tender.actions_export}
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
