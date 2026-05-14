import { useState } from "react";
import { Routes, Route, Navigate, useNavigate, useLocation } from "react-router-dom";
import { Header } from "./components/Header";
import { Tabs } from "./components/Tabs";
import { SettingsDrawer } from "./components/Settings";
import { TenderReport } from "./screens/Tender";
import { ProposalResult } from "./screens/Proposal";
import { CompanyProfile } from "./screens/Profile";
import { MetricsDashboard } from "./screens/Admin";
import styles from "./App.module.css";

export function App() {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsVersion, setSettingsVersion] = useState(0);
  const navigate = useNavigate();
  const location = useLocation();

  const pathMap: Record<string, string> = { "/proposal": "kp", "/profile": "profile" };
  const tab = pathMap[location.pathname] ?? "tender";
  const setTab = (t: string) => {
    const routes: Record<string, string> = { kp: "/proposal", profile: "/profile" };
    navigate(routes[t] ?? "/tender");
  };

  return (
    <div className={styles.shell}>
      <Header onOpenSettings={() => setSettingsOpen(true)} settingsVersion={settingsVersion} />
      <Tabs tab={tab} setTab={setTab} />

      <main className={styles.main}>
        <Routes>
          <Route path="/tender" element={<TenderReport />} />
          <Route path="/proposal" element={<ProposalResult />} />
          <Route path="/profile" element={<CompanyProfile />} />
          <Route path="/admin/metrics" element={<MetricsDashboard />} />
          <Route path="*" element={<Navigate to="/tender" replace />} />
        </Routes>
      </main>

      <SettingsDrawer open={settingsOpen} onClose={() => setSettingsOpen(false)} onSave={() => setSettingsVersion((v) => v + 1)} />
    </div>
  );
}
