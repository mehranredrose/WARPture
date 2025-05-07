//go:build linux

package split

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

const warpTable = 100 // custom routing table for WARP traffic

// applyOSRule applies ip-rule/ip-route based routing on Linux.
// Uses network namespaces or cgroups v2 for per-process routing.
func applyOSRule(ctx context.Context, appPath string, policy Policy) error {
	// Get the cgroup path for the application
	cgroupPath := cgroupForApp(appPath)

	switch policy {
	case PolicyInclude:
		// Mark packets from this cgroup and route via WARP table
		if err := ipRuleAddCgroup(ctx, cgroupPath, warpTable); err != nil {
			return err
		}
		return ipRouteEnsureWarpTable(ctx)

	case PolicyExclude:
		// Mark packets from this cgroup and route via main table (bypass WARP)
		return ipRuleAddCgroupBypass(ctx, cgroupPath)

	case PolicyDefault:
		return ipRuleRemoveCgroup(ctx, cgroupPath)
	}
	return nil
}

// cgroupForApp returns the cgroup v2 path for the given binary.
func cgroupForApp(appPath string) string {
	// Derive a sanitised cgroup name from the binary path
	name := strings.NewReplacer("/", "-", " ", "_", ".", "-").Replace(appPath)
	if len(name) > 64 {
		name = name[len(name)-64:]
	}
	return "/sys/fs/cgroup/warpture" + name
}

func ipRuleAddCgroup(ctx context.Context, cgroupPath string, table int) error {
	// ip rule add cgroup <path> table <table>
	out, err := exec.CommandContext(ctx, "ip", "rule", "add",
		"cgroup", cgroupPath,
		"table", fmt.Sprintf("%d", table),
		"priority", "100",
	).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "File exists") {
			return nil // already present
		}
		return fmt.Errorf("ip rule add cgroup: %w (output: %s)", err, out)
	}
	log.Debugf("[iprule] added cgroup rule: %s → table %d", cgroupPath, table)
	return nil
}

func ipRuleAddCgroupBypass(ctx context.Context, cgroupPath string) error {
	// Route via main table (direct, bypassing WARP)
	out, err := exec.CommandContext(ctx, "ip", "rule", "add",
		"cgroup", cgroupPath,
		"table", "main",
		"priority", "50",
	).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "File exists") {
			return nil
		}
		return fmt.Errorf("ip rule add bypass: %w (output: %s)", err, out)
	}
	return nil
}

func ipRuleRemoveCgroup(ctx context.Context, cgroupPath string) error {
	exec.CommandContext(ctx, "ip", "rule", "del", "cgroup", cgroupPath, "table",
		fmt.Sprintf("%d", warpTable)).Run()
	exec.CommandContext(ctx, "ip", "rule", "del", "cgroup", cgroupPath, "table", "main").Run()
	return nil
}

func ipRouteEnsureWarpTable(ctx context.Context) error {
	// Ensure the WARP interface (CloudflareWARP) has a route in our custom table
	// warp-cli creates CloudflareWARP or similar – discover it dynamically
	iface := discoverWarpInterface(ctx)
	if iface == "" {
		return fmt.Errorf("WARP interface not found")
	}
	out, err := exec.CommandContext(ctx, "ip", "route", "add",
		"default",
		"dev", iface,
		"table", fmt.Sprintf("%d", warpTable),
	).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "File exists") {
		return fmt.Errorf("ip route add warp table: %w (output: %s)", err, out)
	}
	return nil
}

func discoverWarpInterface(ctx context.Context) string {
	candidates := []string{"CloudflareWARP", "warp0", "utun5", "tun0"}
	out, _ := exec.CommandContext(ctx, "ip", "link", "show").Output()
	for _, iface := range candidates {
		if strings.Contains(string(out), iface) {
			return iface
		}
	}
	return ""
}
