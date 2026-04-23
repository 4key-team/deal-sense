import { useRef, useState, useCallback } from "react";
import { UploadIcon } from "../icons/Icons";
import { XIcon, DocIcon } from "../icons/Icons";
import styles from "./Dropzone.module.css";

const DEFAULT_ACCEPT = ".pdf,.docx";
const DEFAULT_ACCEPT_TYPES = [
  "application/pdf",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
];
const DEFAULT_EXTS = ["pdf", "docx"];

function makeIsAccepted(accept?: string) {
  if (!accept || accept === DEFAULT_ACCEPT) {
    return (file: File): boolean => {
      if (DEFAULT_ACCEPT_TYPES.includes(file.type)) return true;
      const ext = file.name.split(".").pop()?.toLowerCase();
      return ext !== undefined && DEFAULT_EXTS.includes(ext);
    };
  }
  const exts = accept.split(",").map((s) => s.trim().replace(/^\./, "").toLowerCase());
  return (file: File): boolean => {
    const ext = file.name.split(".").pop()?.toLowerCase();
    return ext !== undefined && exts.includes(ext);
  };
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export interface DropzoneProps {
  files: File[];
  onFiles: (files: File[]) => void;
  label: string;
  hint?: string;
  multiple?: boolean;
  accept?: string;
}

export function Dropzone({ files, onFiles, label, hint, multiple = true, accept }: DropzoneProps) {
  const isAccepted = makeIsAccepted(accept);
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);

  const addFiles = useCallback(
    (incoming: File[]) => {
      const valid = incoming.filter(isAccepted);
      if (valid.length === 0) return;
      if (multiple) {
        const existing = new Set(files.map((f) => f.name));
        const fresh = valid.filter((f) => !existing.has(f.name));
        onFiles([...files, ...fresh]);
      } else {
        onFiles([valid[0]]);
      }
    },
    [files, onFiles, multiple, accept],
  );

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    addFiles(Array.from(e.dataTransfer.files));
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(true);
  }

  function handleDragLeave() {
    setDragOver(false);
  }

  function handleClick() {
    inputRef.current?.click();
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    if (e.target.files) {
      addFiles(Array.from(e.target.files));
      e.target.value = "";
    }
  }

  function handleRemove(index: number) {
    onFiles(files.filter((_, i) => i !== index));
  }

  return (
    <div>
      <div
        className={`${styles.zone} ${dragOver ? styles.zoneDragOver : ""}`}
        onClick={handleClick}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") handleClick();
        }}
      >
        <span className={styles.icon}>
          <UploadIcon />
        </span>
        <span className={`t-body ${styles.label}`}>{label}</span>
        {hint && <span className={`t-small ${styles.hint}`}>{hint}</span>}
        <input
          ref={inputRef}
          type="file"
          accept={accept ?? DEFAULT_ACCEPT}
          multiple={multiple}
          onChange={handleInputChange}
          className={styles.hidden}
          data-testid="dropzone-input"
        />
      </div>

      {files.length > 0 && (
        <div className={styles.fileList}>
          {files.map((file, i) => (
            <div key={file.name} className={styles.fileRow}>
              <DocIcon />
              <span className={`t-small ${styles.fileName}`}>{file.name}</span>
              <span className={`t-mono t-small ${styles.fileSize}`}>
                {formatSize(file.size)}
              </span>
              <button
                className={styles.removeBtn}
                onClick={(e) => {
                  e.stopPropagation();
                  handleRemove(i);
                }}
                type="button"
                aria-label={`Remove ${file.name}`}
              >
                <XIcon />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
