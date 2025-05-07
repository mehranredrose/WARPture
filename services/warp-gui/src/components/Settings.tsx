import React, { useState, useEffect } from "react";
import { useWarpStore, DefaultPolicy } from "../hooks/useWarpStore";

interface Props {
  onBack: () => void;
}

export function Settings({ onBack }: Props) {
  const { config, stats, setDefaultPolicy } = useWarpStore();
  const [autoLaunch, setAutoLaunchState] = useState(false);

  useEffect(() => {
    // Could check current autolaunch state from OS
  }, []);

  const handleAutoLaunch = async (enabled: boolean) => {
    setAutoLaunchState(enabled);
    await window.warpture?.setAutoLaunch(enabled);
  };

  const handleDefaultPolicy = async (p: DefaultPolicy) => {
    await setDefaultPolicy(p);
  };

  return (
    <div className="settings">
      <div className="settings__header">
        <button className="back-btn" onClick={onBack}>← Back</button>
        <h2>Settings</h2>
      </div>

      <section className="settings__section">
        <h3>Default Policy</h3>
        <p className="settings__desc">
          How to handle apps not explicitly included or excluded.
        </p>
        <div className="policy-selector">
          {(["warp", "bypass", "system"] as DefaultPolicy[]).map((p) => (
            <button
              key={p}
              className={`policy-option ${config?.defaultPolicy === p ? "policy-option--active" : ""}`}
              onClick={() => handleDefaultPolicy(p)}
            >
              {p === "warp" && "🛡️ Route through WARP"}
              {p === "bypass" && "🚫 Bypass WARP"}
              {p === "system" && "⚙️ Use System Default"}
            </button>
          ))}
        </div>
      </section>

      <section className="settings__section">
        <h3>Launch</h3>
        <label className="toggle-label">
          <span>Launch at login</span>
          <input
            type="checkbox"
            checked={autoLaunch}
            onChange={(e) => handleAutoLaunch(e.target.checked)}
          />
        </label>
      </section>

      {stats && (
        <section className="settings__section">
          <h3>Connection Info</h3>
          <dl className="stats-list">
            {stats.account && <><dt>Account</dt><dd>{stats.account}</dd></>}
            {stats.mode && <><dt>Mode</dt><dd>{stats.mode}</dd></>}
            {stats.connectedSince && <><dt>Connected since</dt><dd>{new Date(stats.connectedSince).toLocaleString()}</dd></>}
            <dt>Bytes in</dt><dd>{formatBytes(stats.bytesIn)}</dd>
            <dt>Bytes out</dt><dd>{formatBytes(stats.bytesOut)}</dd>
          </dl>
        </section>
      )}

      <section className="settings__section">
        <h3>About</h3>
        <p className="settings__desc">WARPture v1.0.0</p>
        <button
          className="link-btn"
          onClick={() => window.warpture?.openExternal("https://github.com/mehranredrose/WARPture")}
        >
          GitHub ↗
        </button>
      </section>
    </div>
  );
}

function formatBytes(b: number): string {
  if (!b) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let v = b;
  let u = 0;
  while (v >= 1024 && u < units.length - 1) { v /= 1024; u++; }
  return `${v.toFixed(1)} ${units[u]}`;
}
