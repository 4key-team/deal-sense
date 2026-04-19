import { useState } from "react";
import { Routes, Route, Navigate, useNavigate, useLocation } from "react-router-dom";
import { Header } from "./components/Header";
import { Tabs } from "./components/Tabs";
import { SettingsDrawer } from "./components/Settings";
import { TenderReport } from "./screens/Tender";
import { ProposalResult } from "./screens/Proposal";
import styles from "./App.module.css";

export function App() {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  const tab = location.pathname === "/proposal" ? "kp" : "tender";
  const setTab = (t: string) => {
    navigate(t === "kp" ? "/proposal" : "/tender");
  };

  return (
    <div className={styles.shell}>
      <Header onOpenSettings={() => setSettingsOpen(true)} />
      <Tabs tab={tab} setTab={setTab} />

      <main className={styles.main}>
        <Routes>
          <Route path="/tender" element={<TenderReport />} />
          <Route path="/proposal" element={<ProposalResult />} />
          <Route path="*" element={<Navigate to="/tender" replace />} />
        </Routes>
      </main>

      <SettingsDrawer open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}
