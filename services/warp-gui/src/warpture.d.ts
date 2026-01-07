/**
 * WARPture – Type declarations for the contextBridge API exposed via preload.js
 */

interface WarpStats {
  bytesIn: number;
  bytesOut: number;
  latencyMs: number;
  connectedSince?: string;
  account?: string;
  mode?: string;
}

interface WarpApp {
  id: string;
  name: string;
  bundleId?: string;
  path: string;
  icon?: string;
  policy: 'include' | 'exclude' | 'default';
  running: boolean;
}

interface WarpConfig {
  version: number;
  defaultPolicy: 'warp' | 'bypass' | 'system';
  includedApps: Omit<WarpApp, 'policy' | 'running'>[];
  excludedApps: Omit<WarpApp, 'policy' | 'running'>[];
  updatedAt?: string;
}

interface WarpAPI {
  getApps: () => Promise<WarpApp[]>;
  setAppPolicy: (appId: string, policy: WarpApp['policy']) => Promise<{ ok: boolean }>;
  getConfig: () => Promise<WarpConfig>;
  setDefaultPolicy: (policy: WarpConfig['defaultPolicy']) => Promise<{ ok: boolean }>;
  toggleWarp: () => Promise<{ ok: boolean }>;
  applyPreset: (preset: 'work' | 'gaming' | 'streaming') => Promise<{ ok: boolean }>;
  getStats: () => Promise<WarpStats>;
  setAutoLaunch: (enabled: boolean) => Promise<void>;
  openExternal: (url: string) => Promise<void>;
  onWarpStatus: (cb: (status: string) => void) => () => void;
  onAgentError: (cb: (msg: string) => void) => () => void;
  onPresetApplied: (cb: (preset: string) => void) => () => void;
  onNavigate: (cb: (route: string) => void) => () => void;
}

declare global {
  interface Window {
    warpture: WarpAPI;
  }
}

export {};
