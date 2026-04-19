import { useI18n } from "../../providers/I18nProvider";
import { Card } from "../../ui/Card";
import { SectionLabel } from "../../ui/SectionLabel";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { CheckIcon, DocIcon, DownloadIcon, SparkIcon } from "../../icons/Icons";
import { MiniDonut } from "../../components/charts";
import { getSections, getContext, getMeta, getLog } from "../../mocks/proposal";
import type { ProposalSection } from "../../mocks/proposal";
import styles from "./ProposalResult.module.css";

export function ProposalResult() {
  const { lang, t } = useI18n();

  const sections = getSections(lang);
  const context = getContext(lang, {
    context_brief: t.kp.context_brief,
    context_cases: t.kp.context_cases,
    context_prices: t.kp.context_prices,
  });
  const meta = getMeta(lang);
  const log = getLog(lang);

  return (
    <div className={`screen-enter ${styles.layout}`}>
      {/* ── Left column ── */}
      <div className={styles.leftCol}>
        {/* Hero card */}
        <Card padding={32} style={{ position: "relative", overflow: "hidden" }}>
          <div className={styles.heroBg} aria-hidden="true" />

          {/* Top row */}
          <div className={styles.heroTop}>
            <div className={styles.heroTopLeft}>
              <Chip tone="go" strong icon={<CheckIcon />}>
                {t.kp.ready_chip}
              </Chip>
              <span className={`t-small ${styles.sourceLabel}`}>
                {lang === "ru" ? "Источник" : "Source"}: proposal-tpl.docx
              </span>
            </div>
            <div className={styles.heroTopRight}>
              <Button variant="ghost" icon={<DocIcon />}>{t.kp.open}</Button>
              <Button variant="brand" icon={<DownloadIcon />}>{t.kp.download}</Button>
            </div>
          </div>

          {/* Title */}
          <h1 className={styles.heroTitle}>{t.kp.title}</h1>

          {/* Subtitle */}
          <p className={`t-body ${styles.heroSubtitle}`}>
            {meta.project} · {meta.client}
          </p>

          {/* Meta grid */}
          <div className={styles.metaGrid}>
            {(
              [
                [t.kp.meta.client, meta.client],
                [t.kp.meta.project, meta.project],
                [t.kp.meta.price, meta.price],
                [t.kp.meta.term, meta.term],
                [t.kp.meta.created, meta.created],
              ] as [string, string][]
            ).map(([label, value]) => (
              <div key={label} className={styles.metaCell}>
                <span className={`t-micro ${styles.metaCellLabel}`}>{label}</span>
                <span className={`t-small ${styles.metaCellValue}`}>{value}</span>
              </div>
            ))}
          </div>

          {/* Footer stats */}
          <div className={styles.heroStats}>
            <span className="t-mono dim">
              {lang === "ru" ? "Заняло" : "Took"}: 1m 04s
            </span>
            <span className="t-mono dim">
              {lang === "ru" ? "Токенов" : "Tokens"}: 12,840
            </span>
            <span className="t-mono dim">
              18 {lang === "ru" ? "стр." : "pages"}
            </span>
          </div>
        </Card>

        {/* Sections list */}
        <div className={styles.sectionsBlock}>
          <SectionLabel
            right={
              <Button variant="ghost" size="sm" icon={<SparkIcon />}>
                {t.kp.re_generate}
              </Button>
            }
          >
            {t.kp.sections} · {sections.length}
          </SectionLabel>

          <div className={styles.sectionsList}>
            {sections.map((section, idx) => (
              <SectionRow
                key={section.id}
                section={section}
                index={idx}
                regenerateLabel={t.kp.regenerate_section}
                sectionAiLabel={t.kp.section_ai}
                sectionReviewLabel={t.kp.section_review}
                sectionFilledLabel={t.kp.section_filled}
                lang={lang}
              />
            ))}
          </div>
        </div>
      </div>

      {/* ── Right column ── */}
      <div className={styles.rightCol}>
        {/* Context card */}
        <Card padding={18}>
          <p className={`t-micro ${styles.cardLabel}`}>{t.kp.context_used}</p>
          <div className={styles.contextList}>
            {context.map((file) => (
              <div key={file.name} className={styles.contextRow}>
                <div
                  className={`${styles.contextIcon} ${
                    file.kind === "tpl" ? styles.contextIconTpl : styles.contextIconCtx
                  }`}
                >
                  <DocIcon />
                </div>
                <div className={styles.contextInfo}>
                  <span className={`t-small ${styles.contextName}`}>{file.name}</span>
                  <span className={`t-mono ${styles.contextMeta}`}>
                    {file.role} · {file.size}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Changelog card */}
        <Card padding={18}>
          <p className={`t-micro ${styles.cardLabel}`}>{t.kp.changelog}</p>
          <div className={styles.logList}>
            {log.map((entry) => (
              <div key={entry.time} className={styles.logRow}>
                <span className={`t-mono ${styles.logTime}`}>{entry.time}</span>
                <span className={`t-small ${styles.logMsg}`}>{entry.msg}</span>
              </div>
            ))}
          </div>
        </Card>

        {/* Requirements donut card */}
        <Card padding={18}>
          <p className={`t-micro ${styles.cardLabel}`}>
            {lang === "ru" ? "Требования" : "Requirements"}
          </p>
          <div className={styles.donutRow}>
            <MiniDonut met={5} partial={1} miss={2} size={92} />
            <div className={styles.donutLegend}>
              <div className={styles.legendItem}>
                <span className={styles.legendDot} style={{ background: "var(--go)" }} />
                <span className={`t-small ${styles.legendCount}`}>5</span>
                <span className={`t-small ${styles.legendLabel}`}>
                  {lang === "ru" ? "выполнено" : "met"}
                </span>
              </div>
              <div className={styles.legendItem}>
                <span className={styles.legendDot} style={{ background: "var(--warn)" }} />
                <span className={`t-small ${styles.legendCount}`}>1</span>
                <span className={`t-small ${styles.legendLabel}`}>
                  {lang === "ru" ? "частично" : "partial"}
                </span>
              </div>
              <div className={styles.legendItem}>
                <span className={styles.legendDot} style={{ background: "var(--no)" }} />
                <span className={`t-small ${styles.legendCount}`}>2</span>
                <span className={`t-small ${styles.legendLabel}`}>
                  {lang === "ru" ? "не закрыто" : "missing"}
                </span>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
}

// ── Section row sub-component ──

interface SectionRowProps {
  section: ProposalSection;
  index: number;
  regenerateLabel: string;
  sectionAiLabel: string;
  sectionReviewLabel: string;
  sectionFilledLabel: string;
  lang: string;
}

function SectionRow({
  section,
  index,
  regenerateLabel,
  sectionAiLabel,
  sectionReviewLabel,
  sectionFilledLabel,
  lang,
}: SectionRowProps) {
  const statusChip = (() => {
    if (section.status === "ai") {
      return (
        <Chip tone="brand" icon={<SparkIcon />}>
          {sectionAiLabel}
        </Chip>
      );
    }
    if (section.status === "review") {
      return <Chip tone="warn">{sectionReviewLabel}</Chip>;
    }
    return (
      <Chip tone="neutral" icon={<CheckIcon />}>
        {sectionFilledLabel}
      </Chip>
    );
  })();

  return (
    <div className={styles.sectionRow}>
      <span className={`t-mono ${styles.sectionIdx}`}>
        {String(index + 1).padStart(2, "0")}
      </span>
      <div className={styles.sectionInfo}>
        <span className={`t-body ${styles.sectionTitle}`}>{section.title}</span>
        {section.tokens > 0 && (
          <span className={`t-mono ${styles.sectionTokens}`}>
            {section.tokens} {lang === "ru" ? "токенов" : "tokens"}
          </span>
        )}
      </div>
      <div>{statusChip}</div>
      <button
        className={styles.regenBtn}
        title={regenerateLabel}
        type="button"
      >
        <SparkIcon />
      </button>
    </div>
  );
}
