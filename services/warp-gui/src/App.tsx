import React, { useEffect, useState } from "react";
import { Dashboard } from "./components/Dashboard";
import { Settings } from "./components/Settings";
import { Header } from "./components/Header";
import { useWarpStore } from "./hooks/useWarpStore";
import "./styles/globals.css";

type Route = "dashboard" | "settings";

export default function App() {
  const [route, setRoute] = useState<Route>("dashboard");
  const { init, setStatus } = useWarpStore();

  useEffect(() => {
    // Bootstrap
    init();

    // Listen for main process navigation events
    const removeNav = window.warpture?.onNavigate((r: string) => {
      if (r === "/settings") setRoute("settings");
      else setRoute("dashboard");
    });

    // Listen for WARP status changes pushed via IPC
    const removeStatus = window.warpture?.onWarpStatus((s: string) => {
      setStatus(s as any);
    });

    return () => {
      removeNav?.();
      removeStatus?.();
    };
  }, []);

  return (
    <div className="app-shell">
      <Header onNavigate={setRoute} currentRoute={route} />
      <main className="app-body">
        {route === "dashboard" && <Dashboard />}
        {route === "settings" && <Settings onBack={() => setRoute("dashboard")} />}
      </main>
    </div>
  );
}
