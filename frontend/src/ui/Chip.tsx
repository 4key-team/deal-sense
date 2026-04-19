import type { ReactNode } from "react";
import styles from "./Chip.module.css";

type Tone = "neutral" | "go" | "no" | "brand" | "warn";

interface ChipProps {
  tone?: Tone;
  strong?: boolean;
  icon?: ReactNode;
  children: ReactNode;
}

export function Chip({ tone = "neutral", strong = false, icon, children }: ChipProps) {
  const cls = [styles.chip, styles[tone], strong ? styles.strong : styles.normal].join(" ");
  return (
    <span className={cls}>
      {icon}
      {children}
    </span>
  );
}
