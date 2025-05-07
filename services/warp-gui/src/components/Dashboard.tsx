import React, { useEffect, useRef, useCallback } from "react";
import { FixedSizeList as List } from "react-window";
import { useWarpStore } from "../hooks/useWarpStore";
import { AppRow } from "./AppRow";
import { StatusBar } from "./StatusBar";
import { SearchBar } from "./SearchBar";
import { PresetBar } from "./PresetBar";
import type { AppEntry } from "../hooks/useWarpStore";

export function Dashboard() {
  const {
    apps,
    status,
    stats,
    loading,
    error,
    searchQuery,
    setSearchQuery,
    setAppPolicy,
    toggleWarp,
    applyPreset,
    refreshStats,
  } = useWarpStore();

  // Refresh stats every 5s
  useEffect(() => {
    const interval = setInterval(refreshStats, 5000);
    return () => clearInterval(interval);
  }, [refreshStats]);

  const filtered = apps.filter((a) =>
    a.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const renderRow = useCallback(
    ({ index, style }: { index: number; style: React.CSSProperties }) => {
      const app = filtered[index];
      return (
        <div style={style} key={app.id}>
          <AppRow app={app} onPolicyChange={(p) => setAppPolicy(app.id, p)} />
        </div>
      );
    },
    [filtered, setAppPolicy]
  );

  return (
    <div className="dashboard">
      <StatusBar status={status} stats={stats} onToggle={toggleWarp} />

      <PresetBar onApply={applyPreset} />

      <div className="app-list-header">
        <SearchBar value={searchQuery} onChange={setSearchQuery} />
        <span className="app-count">
          {filtered.length} app{filtered.length !== 1 ? "s" : ""}
        </span>
      </div>

      {loading && (
        <div className="loading-state">
          <div className="spinner" />
          <span>Scanning applications…</span>
        </div>
      )}

      {error && (
        <div className="error-banner">
          ⚠️ {error}
        </div>
      )}

      {!loading && filtered.length === 0 && (
        <div className="empty-state">
          {searchQuery ? "No apps match your search." : "No applications detected."}
        </div>
      )}

      {!loading && filtered.length > 0 && (
        <List
          height={380}
          itemCount={filtered.length}
          itemSize={56}
          width="100%"
          overscanCount={5}
        >
          {renderRow}
        </List>
      )}
    </div>
  );
}
