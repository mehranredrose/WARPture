import { create } from "zustand";

export type WarpStatus = "connected" | "disconnected" | "connecting";
export type AppPolicy = "include" | "exclude" | "default";
export type DefaultPolicy = "warp" | "bypass" | "system";

export interface AppEntry {
  id: string;
  name: string;
  bundleId?: string;
  path: string;
  icon?: string;
  policy: AppPolicy;
  running: boolean;
}

interface WarpConfig {
  version: number;
  defaultPolicy: DefaultPolicy;
  includedApps: Omit<AppEntry, "policy" | "running">[];
  excludedApps: Omit<AppEntry, "policy" | "running">[];
}

interface Stats {
  bytesIn: number;
  bytesOut: number;
  latencyMs: number;
  connectedSince?: string;
  account?: string;
  mode?: string;
}

interface WarpState {
  status: WarpStatus;
  apps: AppEntry[];
  config: WarpConfig | null;
  stats: Stats | null;
  loading: boolean;
  error: string | null;
  searchQuery: string;

  // Actions
  init: () => Promise<void>;
  setStatus: (s: WarpStatus) => void;
  setSearchQuery: (q: string) => void;
  setAppPolicy: (appId: string, policy: AppPolicy) => Promise<void>;
  setDefaultPolicy: (p: DefaultPolicy) => Promise<void>;
  toggleWarp: () => Promise<void>;
  applyPreset: (preset: string) => Promise<void>;
  refreshApps: () => Promise<void>;
  refreshStats: () => Promise<void>;
}

const api = window.warpture;

export const useWarpStore = create<WarpState>((set, get) => ({
  status: "disconnected",
  apps: [],
  config: null,
  stats: null,
  loading: false,
  error: null,
  searchQuery: "",

  init: async () => {
    set({ loading: true, error: null });
    try {
      const [config, apps, stats] = await Promise.all([
        api?.getConfig(),
        api?.getApps(),
        api?.getStats(),
      ]);
      set({
        config: config ?? null,
        apps: apps ?? [],
        stats: stats ?? null,
        loading: false,
      });
    } catch (err: any) {
      set({ error: err.message, loading: false });
    }
  },

  setStatus: (status) => set({ status }),

  setSearchQuery: (searchQuery) => set({ searchQuery }),

  setAppPolicy: async (appId, policy) => {
    // Optimistic update
    set((s) => ({
      apps: s.apps.map((a) => (a.id === appId ? { ...a, policy } : a)),
    }));
    try {
      await api?.setAppPolicy(appId, policy);
    } catch (err: any) {
      // Revert on failure
      set((s) => ({
        error: err.message,
        apps: s.apps.map((a) => (a.id === appId ? { ...a, policy: "default" } : a)),
      }));
    }
  },

  setDefaultPolicy: async (policy) => {
    set((s) => ({
      config: s.config ? { ...s.config, defaultPolicy: policy } : null,
    }));
    try {
      await api?.setDefaultPolicy(policy);
    } catch (err: any) {
      set({ error: err.message });
    }
  },

  toggleWarp: async () => {
    const { status } = get();
    set({ status: "connecting" });
    try {
      await api?.toggleWarp();
    } catch (err: any) {
      set({ status, error: err.message });
    }
  },

  applyPreset: async (preset) => {
    try {
      await api?.applyPreset(preset);
      await get().refreshApps();
    } catch (err: any) {
      set({ error: err.message });
    }
  },

  refreshApps: async () => {
    try {
      const apps = await api?.getApps();
      if (apps) set({ apps });
    } catch (err: any) {
      set({ error: err.message });
    }
  },

  refreshStats: async () => {
    try {
      const stats = await api?.getStats();
      if (stats) set({ stats });
    } catch (err: any) {
      console.error("Failed to refresh stats:", err);
    }
  },
}));
