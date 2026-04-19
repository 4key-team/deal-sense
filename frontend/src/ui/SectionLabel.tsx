import type { ReactNode } from "react";
import styles from "./SectionLabel.module.css";

interface SectionLabelProps {
  children: ReactNode;
  right?: ReactNode;
}

export function SectionLabel({ children, right }: SectionLabelProps) {
  return (
    <div className={styles.wrapper}>
      <div className={`t-micro ${styles.label}`}>{children}</div>
      <div className={styles.line} />
      {right}
    </div>
  );
}
