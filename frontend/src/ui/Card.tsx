import type { CSSProperties, ReactNode } from "react";
import styles from "./Card.module.css";

interface CardProps {
  children: ReactNode;
  padding?: number;
  tight?: boolean;
  style?: CSSProperties;
  className?: string;
}

export function Card({ children, padding = 20, tight = false, style, className }: CardProps) {
  return (
    <div
      className={[styles.card, className].filter(Boolean).join(" ")}
      style={{ padding: tight ? 14 : padding, ...style }}
    >
      {children}
    </div>
  );
}
