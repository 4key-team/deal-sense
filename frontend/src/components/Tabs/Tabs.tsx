import { useI18n } from "../../providers/I18nProvider";
import styles from "./Tabs.module.css";

export interface TabsProps {
  tab: string;
  setTab: (tab: string) => void;
}

interface TabDef {
  id: string;
  label: (t: ReturnType<typeof useI18n>["t"]) => string;
  count: number;
}

const TABS: TabDef[] = [
  { id: "kp",     label: (t) => t.tabs.kp,     count: 12 },
  { id: "tender", label: (t) => t.tabs.tender,  count: 4  },
];

export function Tabs({ tab, setTab }: TabsProps) {
  const { t } = useI18n();

  return (
    <nav className={styles.bar} aria-label="Main navigation">
      <div className={styles.inner} role="tablist">
        {TABS.map(({ id, label, count }) => {
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
              <span
                className={isActive ? `${styles.badge} ${styles.badgeActive}` : styles.badge}
                aria-label={`${count} items`}
              >
                {count}
              </span>
            </button>
          );
        })}
      </div>
    </nav>
  );
}
