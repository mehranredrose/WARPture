//go:build darwin

package split

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// applyOSRule applies pfctl-based routing rules on macOS.
// Requires either root or a privileged helper (warp-helper via SMJobBless).
func applyOSRule(ctx context.Context, appPath string, policy Policy) error {
	anchorName := "com.warpture.split"

	switch policy {
	case PolicyInclude:
		// Force all traffic from this app through the WARP utun interface
		rule := fmt.Sprintf(`pass out route-to (utun0 192.168.0.1) from any to any app "%s"`, appPath)
		return pfctlAddRule(ctx, anchorName, rule)

	case PolicyExclude:
		// Bypass WARP: skip the utun interface for this app
		rule := fmt.Sprintf(`pass out from any to any app "%s" no state`, appPath)
		return pfctlAddRule(ctx, anchorName, rule)

	case PolicyDefault:
		// Remove any existing rules for this app
		return pfctlRemoveRules(ctx, anchorName, appPath)
	}
	return nil
}

func pfctlAddRule(ctx context.Context, anchor, rule string) error {
	// Load existing rules
	existing, _ := runPfctl(ctx, "-a", anchor, "-sr")

	// Append new rule
	rules := strings.TrimSpace(existing) + "\n" + rule
	cmd := exec.CommandContext(ctx, "pfctl", "-a", anchor, "-f", "-")
	cmd.Stdin = strings.NewReader(rules)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pfctl add rule: %w (output: %s)", err, out)
	}
	log.Debugf("[pfctl] added rule to anchor %s", anchor)
	return nil
}

func pfctlRemoveRules(ctx context.Context, anchor, appPath string) error {
	existing, err := runPfctl(ctx, "-a", anchor, "-sr")
	if err != nil {
		return nil // anchor may not exist yet
	}
	var kept []string
	for _, line := range strings.Split(existing, "\n") {
		if !strings.Contains(line, appPath) {
			kept = append(kept, line)
		}
	}
	rules := strings.Join(kept, "\n")
	cmd := exec.CommandContext(ctx, "pfctl", "-a", anchor, "-f", "-")
	cmd.Stdin = strings.NewReader(rules)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pfctl remove rules: %w (output: %s)", err, out)
	}
	return nil
}

func runPfctl(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "pfctl", args...).CombinedOutput()
	return string(out), err
}
