import { useState, useEffect, useRef } from "react";
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

function GeneratingView({ label }: { label: string }) {
  const [elapsed, setElapsed] = useState(0);
  const start = useRef(Date.now());

  useEffect(() => {
    const id = setInterval(() => setElapsed(Math.floor((Date.now() - start.current) / 1000)), 1000);
    return () => clearInterval(id);
  }, []);

  const min = Math.floor(elapsed / 60);
  const sec = elapsed % 60;
  const time = min > 0 ? `${min}:${String(sec).padStart(2, "0")}` : `${sec} сек`;

  return (
    <div className={`screen-enter ${styles.uploadScreen}`}>
      <Spinner size="lg" />
      <p className="t-body muted">{label}</p>
      <p className="t-mono dim" style={{ marginTop: 8 }}>{time}</p>
    </div>
  );
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

  async function fileToBase64(file: File): Promise<string> {
    return new Promise((resolve) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as string);
      reader.readAsDataURL(file);
    });
  }

  function base64ToFile(dataUrl: string, name: string): File {
    const [header, data] = dataUrl.split(",");
    const mime = header.match(/:(.*?);/)?.[1] ?? "";
    const bytes = atob(data);
    const arr = new Uint8Array(bytes.length);
    for (let i = 0; i < bytes.length; i++) arr[i] = bytes.charCodeAt(i);
    return new File([arr], name, { type: mime });
  }

  async function handleGenerate() {
    if (template.length === 0 && contextFiles.length === 0) return;
    setPhase("generating");
    setErrorMsg("");
    try {
      const res = await generateProposal(template[0] ?? null, contextFiles, lang);
      setResult(res);
      const infos = [...template, ...contextFiles].map((f) => ({ name: f.name, size: f.size }));
      setFileInfos(infos);
      setItem("last-proposal-result", res);
      setItem("last-proposal-files", infos);
      // Save files as base64 for re-generation
      const savedFiles = await Promise.all(
        [...template, ...contextFiles].map(async (f) => ({
          name: f.name,
          data: await fileToBase64(f),
        })),
      );
      setItem("last-proposal-raw-files", savedFiles);
      setPhase("result");
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
      setPhase("error");
    }
  }

  async function handleRegenerate() {
    const savedFiles = getItem<{ name: string; data: string }[]>("last-proposal-raw-files", []);
    if (savedFiles.length === 0) {
      setPhase("upload");
      return;
    }
    const files = savedFiles.map((f) => base64ToFile(f.data, f.name));
    setTemplate([files[0]]);
    setContextFiles(files.slice(1));
    setPhase("generating");
    setErrorMsg("");
    try {
      const res = await generateProposal(files[0], files.slice(1), lang);
      setResult(res);
      setItem("last-proposal-result", res);
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
            accept=".docx,.pdf,.md"
            multiple={false}
          />
          <Dropzone
            files={contextFiles}
            onFiles={setContextFiles}
            label={t.dropzone.proposal_ctx_label}
            hint={t.dropzone.proposal_ctx_hint}
            accept=".pdf,.docx,.md"
          />
        </div>
        <Button
          variant="brand"
          size="lg"
          onClick={handleGenerate}
          disabled={template.length === 0 && contextFiles.length === 0}
          icon={<SparkIcon />}
        >
          {t.kp.generate_btn}
        </Button>
      </div>
    );
  }

  // --- Generating phase ---
  if (phase === "generating") {
    return <GeneratingView label={t.kp.generating} />;
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

  const meta = result.meta ?? {};
  const logEntries = result.log ?? [];
  const usage = result.usage;
  const totalTokens = usage?.total_tokens ?? sections.reduce((sum, s) => sum + s.tokens, 0);

  return (
    <div className={`screen-enter ${styles.layout}`}>
      {/* Left column */}
      <div className={styles.leftCol}>
        {/* Hero card */}
        <Card padding={32} style={{ position: "relative", overflow: "hidden" }}>
          <div className={styles.heroBg} aria-hidden="true" />
          <div className={styles.heroTop}>
            <div className={styles.heroTopLeft}>
              <Chip tone="go" strong icon={<CheckIcon />}>
                {t.kp.ready_chip}
              </Chip>
              <span className={`t-small muted`}>
                {lang === "ru" ? "Источник" : "Source"}: {result.template}
              </span>
            </div>
            <div className={styles.heroTopRight}>
              <Button variant="brand" icon={<DownloadIcon />} onClick={() => {
                if (!result.docx) return;
                const bytes = Uint8Array.from(atob(result.docx), (c) => c.charCodeAt(0));
                const blob = new Blob([bytes], {
                  type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
                });
                downloadBlob(blob, "proposal.docx");
              }}>
                {t.kp.download_docx}
              </Button>
              {result.pdf && (
                <Button variant="secondary" icon={<DownloadIcon />} onClick={() => {
                  const bytes = Uint8Array.from(atob(result.pdf!), (c) => c.charCodeAt(0));
                  const blob = new Blob([bytes], { type: "application/pdf" });
                  downloadBlob(blob, "proposal.pdf");
                }}>
                  {t.kp.download_pdf}
                </Button>
              )}
              {result.md && (
                <Button variant="secondary" icon={<DownloadIcon />} onClick={() => {
                  const blob = new Blob([result.md!], { type: "text/markdown" });
                  downloadBlob(blob, "proposal.md");
                }}>
                  {t.kp.download_md}
                </Button>
              )}
            </div>
          </div>
          <h1 className={styles.heroTitle}>{t.kp.title}</h1>
          <p className={`t-body ${styles.heroSubtitle}`}>
            {meta.project && meta.client ? `${meta.project} · ${meta.client}` : result.summary}
          </p>

          {/* Meta grid */}
          <div className={styles.metaGrid}>
            <div className={styles.metaCell}>
              <span className={`t-micro ${styles.metaCellLabel}`}>{t.kp.meta.client}</span>
              <span className={`t-small ${styles.metaCellValue}`}>{meta.client || "—"}</span>
            </div>
            <div className={styles.metaCell}>
              <span className={`t-micro ${styles.metaCellLabel}`}>{t.kp.meta.project}</span>
              <span className={`t-small ${styles.metaCellValue}`}>{meta.project || "—"}</span>
            </div>
            <div className={styles.metaCell}>
              <span className={`t-micro ${styles.metaCellLabel}`}>{t.kp.meta.price}</span>
              <span className={`t-small ${styles.metaCellValue}`}>{meta.price || "—"}</span>
            </div>
            <div className={styles.metaCell}>
              <span className={`t-micro ${styles.metaCellLabel}`}>{t.kp.meta.term}</span>
              <span className={`t-small ${styles.metaCellValue}`}>{meta.timeline || "—"}</span>
            </div>
            <div className={styles.metaCell}>
              <span className={`t-micro ${styles.metaCellLabel}`}>{t.kp.meta.created}</span>
              <span className={`t-small ${styles.metaCellValue}`}>
                {new Date().toLocaleDateString(lang === "ru" ? "ru-RU" : "en-US", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" })}
              </span>
            </div>
          </div>

          {/* Stats footer */}
          <div className={styles.heroStats}>
            <span className="t-mono dim">
              {lang === "ru" ? "Токенов" : "Tokens"}: {totalTokens.toLocaleString()}
            </span>
            <span className="t-mono dim">
              {sections.length} {lang === "ru" ? "секций" : "sections"}
            </span>
          </div>
        </Card>

        {/* Sections list */}
        <div className={styles.sectionsBlock}>
          <SectionLabel right={
            <Button variant="ghost" size="sm" icon={<SparkIcon />} onClick={handleRegenerate}>
              {t.kp.re_generate}
            </Button>
          }>
            {t.kp.sections} · {sections.length}
          </SectionLabel>

          <Card padding={0} style={{ overflow: "hidden" }}>
            {sections.map((section, idx) => (
              <div key={idx} className={styles.sectionRow} style={{
                borderTop: idx === 0 ? "none" : "1px solid var(--line)",
              }}>
                <span className={`t-mono ${styles.sectionIdx}`}>
                  {String(idx + 1).padStart(2, "0")}
                </span>
                <div className={styles.sectionInfo}>
                  <span className={`t-body ${styles.sectionTitle}`}>{section.title}</span>
                  {section.tokens > 0 && (
                    <span className={`t-mono ${styles.sectionTokens}`}>
                      {section.tokens.toLocaleString()} {lang === "ru" ? "токенов" : "tokens"}
                    </span>
                  )}
                </div>
                <div>{statusChip(section)}</div>
                <button
                  className={styles.regenBtn}
                  title={t.kp.regenerate_section}
                  type="button"
                >
                  <SparkIcon />
                </button>
              </div>
            ))}
          </Card>
        </div>
      </div>

      {/* Right column */}
      <div className={styles.rightCol}>
        {/* Context files */}
        <Card padding={18}>
          <p className={`t-micro ${styles.cardLabel}`}>{t.kp.context_used}</p>
          <div className={styles.contextList}>
            {fileInfos.map((file, i) => {
              const isTpl = i === 0; // first file is template
              return (
                <div key={i} className={styles.contextRow}>
                  <div className={`${styles.contextIcon} ${isTpl ? styles.contextIconTpl : styles.contextIconCtx}`}>
                    <DocIcon />
                  </div>
                  <div className={styles.contextInfo}>
                    <span className={`t-small ${styles.contextName}`}>{file.name}</span>
                    <span className={`t-mono ${styles.contextMeta}`}>
                      {isTpl
                        ? (lang === "ru" ? "шаблон" : "template")
                        : (lang === "ru" ? "контекст" : "context")
                      } · {formatFileSize(file.size)}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </Card>

        {/* Changelog */}
        {logEntries.length > 0 && (
          <Card padding={18}>
            <p className={`t-micro ${styles.cardLabel}`}>{t.kp.changelog}</p>
            <div className={styles.logList}>
              {logEntries.map((entry, i) => (
                <div key={i} className={styles.logRow}>
                  <span className={`t-mono ${styles.logTime}`}>{String(i + 1).padStart(2, "0")}</span>
                  <span className={`t-small ${styles.logMsg}`}>{entry.msg}</span>
                </div>
              ))}
            </div>
          </Card>
        )}

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
