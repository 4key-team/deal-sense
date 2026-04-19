import { ChevIcon, CheckIcon } from "../icons/Icons";
import styles from "./Select.module.css";

interface Option {
  id: string;
  label: string;
  mono?: boolean;
}

interface SelectProps {
  value: string;
  options: Option[];
  selected: string;
  open: boolean;
  onOpen: () => void;
  onClose: () => void;
  onPick: (id: string) => void;
  mono?: boolean;
}

export function Select({ value, options, selected, open, onOpen, onClose, onPick, mono = false }: SelectProps) {
  return (
    <div className={styles.wrapper}>
      <button
        className={`${styles.trigger} ${mono ? styles.mono : ""}`}
        onClick={onOpen}
        type="button"
      >
        <span>{value}</span>
        <span className={styles.chevron}>
          <ChevIcon dir={open ? "up" : "down"} />
        </span>
      </button>
      {open && (
        <>
          <div className={styles.backdrop} onClick={onClose} />
          <div className={styles.dropdown}>
            {options.map((o) => (
              <button
                key={o.id}
                className={`${styles.option} ${o.mono ? styles.mono : ""}`}
                onClick={() => onPick(o.id)}
                type="button"
              >
                <span>{o.label}</span>
                {o.id === selected && (
                  <span className={styles.optionCheck}>
                    <CheckIcon />
                  </span>
                )}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
