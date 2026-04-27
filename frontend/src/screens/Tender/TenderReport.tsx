import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useI18n } from "../../providers/useI18n";
import { Card } from "../../ui/Card";
import { SectionLabel } from "../../ui/SectionLabel";
import { Button } from "../../ui/Button";
import { Dropzone } from "../../ui/Dropzone";
import { Spinner } from "../../ui/Spinner";
import {
  CheckIcon,
  XIcon,
  MinusIcon,
  DocIcon,
  DownloadIcon,
  SparkIcon,
} from "../../icons/Icons";
import { MiniHistogram } from "../../components/charts/MiniHistogram";
import { MiniSparkline } from "../../components/charts/MiniSparkline";
import { analyzeTender, downloadBlob } from "../../lib/api";
import type { TenderResult, TenderRequirement as TenderReq } from "../../lib/api";
import { getItem, setItem } from "../../lib/storage";
import { addEntry, getScores, getTrend, getStats } from "../../lib/history";
import { loadProfile } from "../Profile";
import styles from "./TenderReport.module.css";

type Phase = "upload" | "analyzing" | "result" | "error";

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function reqIcon(status: TenderReq["status"]) {
  if (status === "met") {
    return { bg: "var(--go-wash)", border: "var(--go-line)", icon: <span style={{ color: "var(--go)" }}><CheckIcon /></span> };
  }
  if (status === "partial") {
    return { bg: "var(--warn-wash)", border: "var(--warn-line)", icon: <span style={{ color: "var(--warn)" }}><MinusIcon /></span> };
  }
  return { bg: "var(--no-wash)", border: "var(--no-line)", icon: <span style={{ color: "var(--no)" }}><XIcon /></span> };
}

export function TenderReport() {
  const { lang, t } = useI18n();
  const navigate = useNavigate();
  const cachedResult = getItem<TenderResult | null>("last-tender-result", null);
  const cachedFiles = getItem<{ name: string; size: number }[]>("last-tender-files", []);
  const [phase, setPhase] = useState<Phase>(cachedResult ? "result" : "upload");
  const [files, setFiles] = useState<File[]>([]);
  const [fileInfos, setFileInfos] = useState(cachedFiles);
  const [errorMsg, setErrorMsg] = useState("");
  const [result, setResult] = useState<TenderResult | null>(cachedResult);

  async function handleAnalyze() {
    if (files.length === 0) return;
    setPhase("analyzing");
    setErrorMsg("");
    try {
      const p = loadProfile();
      const profileText = [
        p.name && `Company: ${p.name}`,
        p.teamSize && `Team: ${p.teamSize} people`,
        p.experience && `Experience: ${p.experience} years`,
        p.stack.length > 0 && `Tech stack: ${p.stack.join(", ")}`,
        p.certs.length > 0 && `Certifications: ${p.certs.join(", ")}`,
        p.specializations.length > 0 && `Specializations: ${p.specializations.join(", ")}`,
        p.clients && `Key clients/projects: ${p.clients}`,
        p.extra && `Additional: ${p.extra}`,
      ].filter(Boolean).join(". ");
      const res = await analyzeTender(files, profileText, lang);
      setResult(res);
      const infos = files.map((f) => ({ name: f.name, size: f.size }));
      setFileInfos(infos);
      setItem("last-tender-result", res);
      setItem("last-tender-files", infos);
      addEntry({
        score: res.score,
        verdict: res.verdict,
        fileName: files[0]?.name ?? "unknown",
      });
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
        <h2 className={`t-h2 font-serif ${styles.uploadTitle}`}>{t.tender.title}</h2>
        <p className={`t-body muted ${styles.uploadSubtitle}`}>{t.tender.subtitle}</p>
        <div className={styles.uploadZone}>
          <Dropzone
            files={files}
            onFiles={setFiles}
            label={t.dropzone.tender_label}
            hint={t.dropzone.tender_hint}
            accept=".pdf,.docx,.md,.zip"
          />
        </div>
        <Button
          variant="brand"
          size="lg"
          onClick={handleAnalyze}
          disabled={files.length === 0}
          icon={<SparkIcon />}
        >
          {t.tender.analyze_btn}
        </Button>
      </div>
    );
  }

  // --- Analyzing phase ---
  if (phase === "analyzing") {
    return (
      <div className={`screen-enter ${styles.uploadScreen}`}>
        <Spinner size="lg" />
        <p className="t-body muted">{t.tender.analyzing}</p>
      </div>
    );
  }

  // --- Error phase ---
  if (phase === "error") {
    return (
      <div className={`screen-enter ${styles.uploadScreen}`}>
        <p className={`t-body ${styles.errorText}`}>{t.tender.error}</p>
        <p className="t-small muted">{errorMsg}</p>
        <Button variant="secondary" onClick={handleAnalyze}>
          {t.tender.retry}
        </Button>
      </div>
    );
  }

  // --- Result phase ---
  if (!result) return null;

  const isGo = result.verdict === "go";
  const verdictLabel = isGo ? t.tender.verdict_go : t.tender.verdict_no;
  const verdictSub = isGo ? t.tender.verdict_go_sub : t.tender.verdict_no_sub;
  const reqs = result.requirements ?? [];
  const metCount = reqs.filter((r) => r.status === "met").length;
  const partialCount = reqs.filter((r) => r.status === "partial").length;
  const missCount = reqs.filter((r) => r.status === "miss").length;

  function handleExport() {
    const json = JSON.stringify(result, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    downloadBlob(blob, "tender-report.json");
  }

  function handlePrep() {
    navigate("/proposal");
  }

  return (
    <div className={`screen-enter ${styles.screen}`}>
      {/* Left column */}
      <div className={styles.left}>
        {/* Verdict hero */}
        <Card padding={0} style={{
          overflow: "hidden",
          border: `1px solid ${isGo ? "var(--go-line)" : "var(--no-line)"}`,
        }}>
          <div className={styles.verdictWrap} style={{
            background: `linear-gradient(180deg, ${isGo ? "var(--go-wash)" : "var(--no-wash)"} 0%, var(--surface) 80%)`,
          }}>
            <div className={styles.verdictLayout}>
              <div className={styles.verdictLeft}>
                <span className={`t-micro`} style={{ color: isGo ? "var(--go-ink)" : "var(--no-ink)" }}>
                  {t.tender.fit} · {result.score}%
                </span>
                <h1 className={styles.verdictTitle} style={{ color: isGo ? "var(--go-ink)" : "var(--no-ink)" }}>
                  {verdictLabel}
                </h1>
              </div>
              <div className={styles.verdictRight}>
                <p className={`t-body ${styles.verdictSub}`}>{verdictSub}</p>
                <div className={styles.scoreBar}>
                  <div className={styles.scoreFill} style={{
                    width: `${result.score}%`,
                    background: isGo ? "var(--go)" : "var(--no)",
                  }} />
                </div>
                <div className={styles.scoreLabels}>
                  <span className="t-mono t-small muted">0</span>
                  <span className="t-mono t-small muted">100</span>
                </div>
                <p className={`t-small ${styles.verdictNote}`}>
                  {100 - result.score}% — {result.risk === "low"
                    ? (lang === "ru" ? "незначительные пробелы" : "minor gaps")
                    : result.risk === "medium"
                      ? (lang === "ru" ? "есть пробелы" : "some gaps")
                      : (lang === "ru" ? "серьёзные пробелы" : "major gaps")}
                </p>
              </div>
            </div>
          </div>
        </Card>

        {/* Pro/Con grid */}
        <div className={styles.proConGrid}>
          <Card padding={24}>
            <div className={styles.proConHeader}>
              <span className={styles.proConDot} style={{ background: "var(--go)" }} />
              <span className={`t-body ${styles.proConTitle}`}>{t.tender.pros}</span>
              <span className="t-mono muted">{(result.pros ?? []).length}</span>
            </div>
            {(result.pros ?? []).map((p, i) => (
              <div key={i} className={styles.proConItem}>
                <span className={`t-body ${styles.proConItemTitle}`}>{p.title}</span>
                <p className={`t-small muted ${styles.proConItemDesc}`}>{p.desc}</p>
              </div>
            ))}
          </Card>
          <Card padding={24}>
            <div className={styles.proConHeader}>
              <span className={styles.proConDot} style={{ background: "var(--no)" }} />
              <span className={`t-body ${styles.proConTitle}`}>{t.tender.cons}</span>
              <span className="t-mono muted">{(result.cons ?? []).length}</span>
            </div>
            {(result.cons ?? []).map((c, i) => (
              <div key={i} className={styles.proConItem}>
                <span className={`t-body ${styles.proConItemTitle}`}>{c.title}</span>
                <p className={`t-small muted ${styles.proConItemDesc}`}>{c.desc}</p>
              </div>
            ))}
          </Card>
        </div>

        {/* Requirements */}
        {reqs.length > 0 && (
          <div>
            <div className={styles.requirementsLabel}>
              <SectionLabel>
                {t.tender.requirements} · {reqs.length}
              </SectionLabel>
            </div>
            <Card padding={0} style={{ overflow: "hidden" }}>
              <div className={styles.reqGrid}>
                {reqs.map((req, i) => {
                  const cfg = reqIcon(req.status);
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
                        style={{ background: cfg.bg, border: `1px solid ${cfg.border}` }}
                      >
                        {cfg.icon}
                      </div>
                      <span className={`t-small ${styles.reqLabel}`} style={{ fontWeight: 500, color: "var(--ink-1)" }}>
                        {req.label}
                      </span>
                      <span className={styles.reqStatus}>{statusText}</span>
                    </div>
                  );
                })}
              </div>
            </Card>
          </div>
        )}
      </div>

      {/* Right column */}
      <div className={styles.right}>
        {/* Uploaded files */}
        <Card padding={18}>
          <span className={`t-micro ${styles.cardHeader}`}>{t.tender.files}</span>
          <div className={styles.fileList}>
            {fileInfos.map((file, i) => (
              <div key={i} className={styles.fileRow}>
                <div className={styles.docIconBox}><DocIcon /></div>
                <div className={styles.fileInfo}>
                  <span className={`t-small ${styles.fileName}`}>{file.name}</span>
                  <span className={`t-mono ${styles.fileMeta}`}>{formatFileSize(file.size)}</span>
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Effort */}
        <Card padding={18}>
          <span className={`t-micro ${styles.cardHeader}`}>{t.tender.effort}</span>
          <span className={styles.effortValue}>{result.effort || "—"}</span>
          <p className="t-small dim">{result.summary}</p>
          {result.usage && (
            <p className="t-micro dim" style={{ marginTop: 8 }}>
              {lang === "ru" ? "Токены" : "Tokens"}: {result.usage.prompt_tokens.toLocaleString()} prompt + {result.usage.completion_tokens.toLocaleString()} completion = {result.usage.total_tokens.toLocaleString()}
            </p>
          )}
        </Card>

        {/* Stats summary */}
        <Card padding={18}>
          <span className={`t-micro ${styles.cardHeader}`}>{t.tender.requirements}</span>
          <div className={styles.legend}>
            <div className={styles.legendItem}>
              <div className={styles.legendDot} style={{ background: "var(--go)" }} />
              {t.tender.req_met} {metCount}
            </div>
            <div className={styles.legendItem}>
              <div className={styles.legendDot} style={{ background: "var(--warn)" }} />
              {t.tender.req_partial} {partialCount}
            </div>
            <div className={styles.legendItem}>
              <div className={styles.legendDot} style={{ background: "var(--no)" }} />
              {t.tender.req_miss} {missCount}
            </div>
          </div>
        </Card>

        {/* History charts */}
        {(() => {
          const scores = getScores();
          const trend = getTrend();
          const stats = getStats();
          if (scores.length === 0) return null;
          return (
            <>
              <Card padding={18}>
                <span className={`t-micro ${styles.cardHeader}`}>{t.tender.recent}</span>
                <span className={styles.bigNumber}>{stats.avgScore}</span>
                <p className="t-small dim">
                  {lang === "ru" ? "средний fit" : "avg fit"}
                </p>
                <MiniHistogram data={scores} height={40} />
                <div className={styles.legend}>
                  <div className={styles.legendItem}>
                    <div className={styles.legendDot} style={{ background: "var(--go)" }} />
                    GO {stats.go}
                  </div>
                  <div className={styles.legendItem}>
                    <div className={styles.legendDot} style={{ background: "var(--warn)" }} />
                    watch {stats.watch}
                  </div>
                  <div className={styles.legendItem}>
                    <div className={styles.legendDot} style={{ background: "var(--no)" }} />
                    NO {stats.no}
                  </div>
                </div>
              </Card>

              {trend.length >= 2 && (
                <Card padding={18}>
                  <span className={`t-micro ${styles.cardHeader}`}>{t.tender.trend}</span>
                  <MiniSparkline data={trend} height={40} />
                  <span className={styles.trendFooter}>
                    {lang === "ru"
                      ? `${trend.length} анализов`
                      : `${trend.length} analyses`}
                  </span>
                </Card>
              )}
            </>
          );
        })()}

        {/* Action buttons */}
        <div className={styles.actions}>
          {isGo ? (
            <>
              <Button variant="brand" size="lg" icon={<SparkIcon />} onClick={handlePrep}>
                {t.tender.actions_prep}
              </Button>
              <Button variant="secondary" size="lg" icon={<DownloadIcon />} onClick={handleExport}>
                {t.tender.actions_export}
              </Button>
            </>
          ) : (
            <>
              <Button variant="secondary" size="lg" icon={<DownloadIcon />} onClick={handleExport}>
                {t.tender.actions_export}
              </Button>
            </>
          )}
          <Button variant="outline" size="lg" onClick={() => { setPhase("upload"); setResult(null); setFiles([]); setFileInfos([]); setItem("last-tender-result", null); setItem("last-tender-files", []); }}>
            {t.tender.new_analysis}
          </Button>
        </div>
      </div>
    </div>
  );
}
