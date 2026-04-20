import { useI18n } from "../../providers/useI18n";
import styles from "./Tabs.module.css";

export interface TabsProps {
  tab: string;
  setTab: (tab: string) => void;
}

interface TabDef {
  id: string;
  label: (t: ReturnType<typeof useI18n>["t"]) => string;
}

const TABS: TabDef[] = [
  { id: "kp",      label: (t) => t.tabs.kp },
  { id: "tender",  label: (t) => t.tabs.tender },
  { id: "profile", label: (t) => t.tabs.profile },
];

export function Tabs({ tab, setTab }: TabsProps) {
  const { t } = useI18n();

  return (
    <nav className={styles.bar} aria-label="Main navigation">
      <div className={styles.inner} role="tablist">
        {TABS.map(({ id, label }) => {
          const isActive = tab === id;
          return (
            <button
              key={id}
              role="tab"
              aria-selected={isActive}
              className={isActive ? `${styles.tab} ${styles.tabActive}` : styles.tab}
              onClick={() => setTab(id)}
            >
              {label(t)}
            </button>
          );
        })}
      </div>
    </nav>
  );
}
