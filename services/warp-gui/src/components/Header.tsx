import React from "react";
import { motion } from "framer-motion";
import { useWarpStore, WarpStatus, Stats } from "../hooks/useWarpStore";

// ── Header ────────────────────────────────────────────────────────────────────
interface HeaderProps {
  onNavigate: (route: "dashboard" | "settings") => void;
  currentRoute: string;
}

export function Header({ onNavigate, currentRoute }: HeaderProps) {
  return (
    <header className="app-header">
      <div className="app-header__logo">
        <span className="app-header__shield">🛡️</span>
        <span className="app-header__title">WARPture</span>
      </div>
      <nav className="app-header__nav">
        <button
          className={`nav-btn ${currentRoute === "dashboard" ? "nav-btn--active" : ""}`}
          onClick={() => onNavigate("dashboard")}
        >
          Apps
        </button>
        <button
          className={`nav-btn ${currentRoute === "settings" ? "nav-btn--active" : ""}`}
          onClick={() => onNavigate("settings")}
        >
          ⚙️
        </button>
      </nav>
    </header>
  );
}

// ── StatusBar ─────────────────────────────────────────────────────────────────
interface StatusBarProps {
  status: WarpStatus;
  stats: Stats | null;
  onToggle: () => void;
}

const STATUS_LABELS: Record<WarpStatus, string> = {
  connected: "Connected",
  disconnected: "Disconnected",
  connecting: "Connecting…",
};

export function StatusBar({ status, stats, onToggle }: StatusBarProps) {
  return (
    <div className={`status-bar status-bar--${status}`}>
      <div className="status-bar__indicator">
        <motion.span
          className="status-dot"
          animate={
            status === "connecting"
              ? { opacity: [1, 0.3, 1] }
              : { opacity: 1 }
          }
          transition={{ duration: 1, repeat: Infinity }}
        />
        <div className="status-bar__text">
          <span className="status-bar__label">{STATUS_LABELS[status]}</span>
          {stats?.latencyMs != null && status === "connected" && (
            <span className="status-bar__latency">{stats.latencyMs}ms</span>
          )}
          {stats?.account && (
            <span className="status-bar__account">{stats.account}</span>
          )}
        </div>
      </div>

      <motion.button
        className={`warp-toggle warp-toggle--${status}`}
        onClick={onToggle}
        whileTap={{ scale: 0.97 }}
        disabled={status === "connecting"}
      >
        {status === "connected" ? "Disconnect" : "Connect"}
      </motion.button>
    </div>
  );
}

// ── SearchBar ─────────────────────────────────────────────────────────────────
interface SearchBarProps {
  value: string;
  onChange: (v: string) => void;
}

export function SearchBar({ value, onChange }: SearchBarProps) {
  return (
    <div className="search-bar">
      <span className="search-bar__icon">🔍</span>
      <input
        className="search-bar__input"
        type="text"
        placeholder="Search applications…"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        spellCheck={false}
        autoComplete="off"
      />
      {value && (
        <button className="search-bar__clear" onClick={() => onChange("")}>
          ✕
        </button>
      )}
    </div>
  );
}

// ── PresetBar ─────────────────────────────────────────────────────────────────
interface PresetBarProps {
  onApply: (preset: string) => void;
}

const PRESETS = [
  { id: "work", label: "🏢 Work" },
  { id: "gaming", label: "🎮 Gaming" },
  { id: "streaming", label: "📺 Stream" },
];

export function PresetBar({ onApply }: PresetBarProps) {
  return (
    <div className="preset-bar">
      <span className="preset-bar__label">Presets:</span>
      {PRESETS.map((p) => (
        <motion.button
          key={p.id}
          className="preset-btn"
          onClick={() => onApply(p.id)}
          whileTap={{ scale: 0.95 }}
          whileHover={{ scale: 1.03 }}
        >
          {p.label}
        </motion.button>
      ))}
    </div>
  );
}
