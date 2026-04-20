import { useState } from "react";
import { useI18n } from "../../providers/useI18n";
import { Card } from "../../ui/Card";
import { SectionLabel } from "../../ui/SectionLabel";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { Spinner } from "../../ui/Spinner";
import { Dropzone } from "../../ui/Dropzone";
import { CheckIcon, DocIcon, DownloadIcon, SparkIcon } from "../../icons/Icons";
import { MiniDonut } from "../../components/charts";
import { generateProposal, downloadBlob } from "../../lib/api";
import { getItem, setItem } from "../../lib/storage";
import type { ProposalResult as ProposalResultType, ProposalSection } from "../../lib/api";
import styles from "./ProposalResult.module.css";

type Phase = "upload" | "generating" | "result" | "error";

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function ProposalResult() {
  const { lang, t } = useI18n();
  const cachedResult = getItem<ProposalResultType | null>("last-proposal-result", null);
  const cachedFiles = getItem<{ name: string; size: number }[]>("last-proposal-files", []);
  const [phase, setPhase] = useState<Phase>(cachedResult ? "result" : "upload");
  const [template, setTemplate] = useState<File[]>([]);
  const [contextFiles, setContextFiles] = useState<File[]>([]);
  const [fileInfos, setFileInfos] = useState(cachedFiles);
  const [errorMsg, setErrorMsg] = useState("");
  const [result, setResult] = useState<ProposalResultType | null>(cachedResult);

  async function handleGenerate() {
    if (template.length === 0) return;
    setPhase("generating");
    setErrorMsg("");
    try {
      const res = await generateProposal(template[0], contextFiles, lang);
      setResult(res);
      const infos = [...template, ...contextFiles].map((f) => ({ name: f.name, size: f.size }));
      setFileInfos(infos);
      setItem("last-proposal-result", res);
      setItem("last-proposal-files", infos);
      setPhase("result");
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
      setPhase("error");
    }
  }

  // --- Upload phase ---
  if (phase === "upload") {
    return (
      <div className={`screen-enter ${styles.uploadScreen}`}>
        <h2 className={`t-h2 font-serif ${styles.uploadTitle}`}>{t.kp.title}</h2>
        <p className={`t-body muted ${styles.uploadSubtitle}`}>{t.kp.subtitle}</p>
        <div className={styles.uploadZones}>
          <Dropzone
            files={template}
            onFiles={setTemplate}
            label={t.dropzone.proposal_tpl_label}
            hint={t.dropzone.proposal_tpl_hint}
            multiple={false}
          />
          <Dropzone
            files={contextFiles}
            onFiles={setContextFiles}
            label={t.dropzone.proposal_ctx_label}
            hint={t.dropzone.proposal_ctx_hint}
          />
        </div>
        <Button
          variant="brand"
          size="lg"
          onClick={handleGenerate}
          disabled={template.length === 0}
          icon={<SparkIcon />}
        >
          {t.kp.generate_btn}
        </Button>
      </div>
    );
  }

  // --- Generating phase ---
  if (phase === "generating") {
    return (
      <div className={`screen-enter ${styles.uploadScreen}`}>
        <Spinner />
        <p className="t-body muted">{t.kp.generating}</p>
      </div>
    );
  }

  // --- Error phase ---
  if (phase === "error") {
    return (
      <div className={`screen-enter ${styles.uploadScreen}`}>
        <p className={`t-body ${styles.errorText}`}>{t.kp.error}</p>
        <p className="t-small muted">{errorMsg}</p>
        <Button variant="secondary" onClick={handleGenerate}>
          {t.kp.retry}
        </Button>
      </div>
    );
  }

  // --- Result phase ---
  if (!result) return null;

  const sections = result.sections ?? [];
  const aiCount = sections.filter((s) => s.status === "ai").length;
  const filledCount = sections.filter((s) => s.status === "filled").length;
  const reviewCount = sections.filter((s) => s.status === "review").length;

  function statusChip(s: ProposalSection) {
    if (s.status === "ai") {
      return <Chip tone="brand" icon={<SparkIcon />}>{t.kp.section_ai}</Chip>;
    }
    if (s.status === "review") {
      return <Chip tone="warn">{t.kp.section_review}</Chip>;
    }
    return <Chip tone="neutral" icon={<CheckIcon />}>{t.kp.section_filled}</Chip>;
  }

  return (
    <div className={`screen-enter ${styles.layout}`}>
      {/* Left column */}
      <div className={styles.leftCol}>
        {/* Hero card */}
        <Card padding={32}>
          <div className={styles.heroTop}>
            <Chip tone="go" strong icon={<CheckIcon />}>
              {t.kp.ready_chip}
            </Chip>
            <span className={`t-small muted`}>
              {result.template}
            </span>
          </div>
          <h1 className={styles.heroTitle}>{t.kp.title}</h1>
          <p className={`t-body ${styles.heroSubtitle}`}>{t.kp.subtitle_result}</p>
          <p className="t-small muted">{result.summary}</p>
        </Card>

        {/* Sections list */}
        <div>
          <SectionLabel>
            {t.kp.sections} · {sections.length}
          </SectionLabel>
          <div className={styles.sectionsList}>
            {sections.map((section, idx) => (
              <Card key={idx} padding={16}>
                <div className={styles.sectionRow}>
                  <span className={`t-mono ${styles.sectionIdx}`}>
                    {String(idx + 1).padStart(2, "0")}
                  </span>
                  <div className={styles.sectionInfo}>
                    <span className={`t-body ${styles.sectionTitle}`}>{section.title}</span>
                    {section.tokens > 0 && (
                      <span className={`t-mono t-small muted`}>
                        {section.tokens} {lang === "ru" ? "токенов" : "tokens"}
                      </span>
                    )}
                  </div>
                  <div>{statusChip(section)}</div>
                </div>
              </Card>
            ))}
          </div>
        </div>
      </div>

      {/* Right column */}
      <div className={styles.rightCol}>
        {/* Actions */}
        <Card padding={18}>
          <Button variant="brand" size="lg" icon={<DownloadIcon />} onClick={() => {
            if (!result.docx) return;
            const bytes = Uint8Array.from(atob(result.docx), (c) => c.charCodeAt(0));
            const blob = new Blob([bytes], {
              type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
            });
            downloadBlob(blob, "proposal.docx");
          }}>
            {t.kp.download}
          </Button>
        </Card>

        {/* Uploaded files */}
        <Card padding={18}>
          <p className={`t-micro ${styles.cardLabel}`}>{t.kp.context_used}</p>
          <div className={styles.fileList}>
            {fileInfos.map((file, i) => (
              <div key={i} className={styles.fileRow}>
                <DocIcon />
                <span className={`t-small ${styles.fileName}`}>{file.name}</span>
                <span className={`t-mono t-small muted`}>{formatFileSize(file.size)}</span>
              </div>
            ))}
          </div>
        </Card>

        {/* Stats with donut */}
        <Card padding={18}>
          <p className={`t-micro ${styles.cardLabel}`}>{t.kp.sections}</p>
          <div className={styles.donutRow}>
            <MiniDonut met={aiCount} partial={reviewCount} miss={filledCount} size={92} />
            <div className={styles.statsGrid}>
              <div className={styles.statItem}>
                <span className={styles.statDot} style={{ background: "var(--brand)" }} />
                <span className={`t-small ${styles.statCount}`}>{aiCount}</span>
                <span className={`t-small ${styles.statLabel}`}>{t.kp.section_ai}</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statDot} style={{ background: "var(--go)" }} />
                <span className={`t-small ${styles.statCount}`}>{filledCount}</span>
                <span className={`t-small ${styles.statLabel}`}>{t.kp.section_filled}</span>
              </div>
              <div className={styles.statItem}>
                <span className={styles.statDot} style={{ background: "var(--warn)" }} />
                <span className={`t-small ${styles.statCount}`}>{reviewCount}</span>
                <span className={`t-small ${styles.statLabel}`}>{t.kp.section_review}</span>
              </div>
            </div>
          </div>
        </Card>

        {/* New proposal */}
        <Button variant="outline" size="lg" onClick={() => {
          setPhase("upload"); setResult(null); setTemplate([]); setContextFiles([]); setFileInfos([]);
          setItem("last-proposal-result", null); setItem("last-proposal-files", []);
        }}>
          {t.kp.new_proposal}
        </Button>
      </div>
    </div>
  );
}
