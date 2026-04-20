import { useState, type ReactNode } from "react";
import styles from "./Field.module.css";

interface FieldProps {
  label: string;
  hint?: string;
  tooltip?: string;
  children: ReactNode;
}

export function Field({ label, hint, tooltip, children }: FieldProps) {
  const [showTip, setShowTip] = useState(false);

  return (
    <div className={styles.wrapper}>
      <div className={`t-micro ${styles.label}`}>
        {label}
        {tooltip && (
          <button
            type="button"
            className={styles.tipBtn}
            onClick={() => setShowTip((v) => !v)}
            aria-label={`Info: ${label}`}
          >
            ?
          </button>
        )}
      </div>
      {showTip && tooltip && (
        <div className={`t-small ${styles.tipBox}`}>{tooltip}</div>
      )}
      {children}
      {hint && <div className={`t-small dim ${styles.hint}`}>{hint}</div>}
    </div>
  );
}
