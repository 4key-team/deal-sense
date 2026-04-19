import type { ReactNode } from "react";
import styles from "./Field.module.css";

interface FieldProps {
  label: string;
  hint?: string;
  children: ReactNode;
}

export function Field({ label, hint, children }: FieldProps) {
  return (
    <div className={styles.wrapper}>
      <div className={`t-micro ${styles.label}`}>{label}</div>
      {children}
      {hint && <div className={`t-small dim ${styles.hint}`}>{hint}</div>}
    </div>
  );
}
