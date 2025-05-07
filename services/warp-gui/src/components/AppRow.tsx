import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { AppEntry, AppPolicy } from "../hooks/useWarpStore";

interface Props {
  app: AppEntry;
  onPolicyChange: (policy: AppPolicy) => void;
}

const POLICY_LABELS: Record<AppPolicy, string> = {
  include: "INCLUDE",
  exclude: "EXCLUDE",
  default: "DEFAULT",
};

const POLICY_COLORS: Record<AppPolicy, string> = {
  include: "var(--color-include)",
  exclude: "var(--color-exclude)",
  default: "var(--color-default)",
};

export function AppRow({ app, onPolicyChange }: Props) {
  const cycle = (): void => {
    const order: AppPolicy[] = ["default", "include", "exclude"];
    const idx = order.indexOf(app.policy);
    onPolicyChange(order[(idx + 1) % order.length]);
  };

  return (
    <div className={`app-row ${app.running ? "app-row--running" : ""}`}>
      {/* Icon */}
      <div className="app-row__icon">
        {app.icon ? (
          <img src={app.icon} alt={app.name} width={32} height={32} />
        ) : (
          <div className="app-row__icon-fallback">
            {app.name.charAt(0).toUpperCase()}
          </div>
        )}
        {app.running && <span className="app-row__running-dot" title="Running" />}
      </div>

      {/* Name & path */}
      <div className="app-row__info">
        <span className="app-row__name">{app.name}</span>
        {app.bundleId && (
          <span className="app-row__bundle">{app.bundleId}</span>
        )}
      </div>

      {/* Policy toggle */}
      <motion.button
        className="policy-toggle"
        style={{ "--policy-color": POLICY_COLORS[app.policy] } as React.CSSProperties}
        onClick={cycle}
        whileTap={{ scale: 0.95 }}
        title={`Policy: ${app.policy} — click to cycle`}
      >
        <AnimatePresence mode="wait">
          <motion.span
            key={app.policy}
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 4 }}
            transition={{ duration: 0.12 }}
          >
            {app.policy === "include" && "●"}
            {app.policy === "exclude" && "○"}
            {app.policy === "default" && "~"}
          </motion.span>
        </AnimatePresence>
        <span className="policy-toggle__label">{POLICY_LABELS[app.policy]}</span>
      </motion.button>
    </div>
  );
}
