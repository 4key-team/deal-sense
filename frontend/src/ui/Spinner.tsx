import styles from "./Spinner.module.css";

export function Spinner({ size = "sm" }: { size?: "sm" | "lg" }) {
  return <span className={size === "lg" ? styles.spinnerLg : styles.spinner} />;
}
