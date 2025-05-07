package split_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mehranredrose/warpture/tunnel-agent/internal/split"
	"github.com/mehranredrose/warpture/tunnel-agent/internal/warp"
)

func newTestManager(t *testing.T) (*split.Manager, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "split-tunnel.json")
	w := warp.NewClient("warp-cli-nonexistent", "127.0.0.1", "40000")
	m := split.NewManager(path, w)
	return m, path
}

func TestLoad_NoFile(t *testing.T) {
	m, _ := newTestManager(t)
	err := m.Load()
	require.NoError(t, err)
	cfg := m.GetConfig()
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, split.DefaultPolicyWarp, cfg.DefaultPolicy)
}

func TestSaveAndLoad(t *testing.T) {
	m, path := newTestManager(t)
	require.NoError(t, m.Load())
	m.MergeDetectedApps([]split.AppEntry{
		{ID: "firefox", Name: "Firefox", Path: "/Applications/Firefox.app"},
		{ID: "zoom", Name: "Zoom", Path: "/Applications/zoom.us.app"},
	})
	require.NoError(t, m.SetAppPolicy("firefox", split.PolicyInclude))
	require.NoError(t, m.SetAppPolicy("zoom", split.PolicyExclude))

	w2 := warp.NewClient("warp-cli-nonexistent", "127.0.0.1", "40000")
	m2 := split.NewManager(path, w2)
	require.NoError(t, m2.Load())
	cfg := m2.GetConfig()
	assert.Len(t, cfg.IncludedApps, 1)
	assert.Len(t, cfg.ExcludedApps, 1)
	assert.Equal(t, "Firefox", cfg.IncludedApps[0].Name)
	assert.Equal(t, "Zoom", cfg.ExcludedApps[0].Name)
}

func TestSetAppPolicy_UnknownApp(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	err := m.SetAppPolicy("nonexistent-app", split.PolicyInclude)
	assert.Error(t, err)
}

func TestMergeDetectedApps(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	apps := []split.AppEntry{
		{ID: "app1", Name: "App One", Path: "/Applications/App1.app"},
		{ID: "app2", Name: "App Two", Path: "/Applications/App2.app"},
	}
	m.MergeDetectedApps(apps)
	assert.Len(t, m.GetApps(), 2)
	m.MergeDetectedApps(apps)
	assert.Len(t, m.GetApps(), 2)
}

func TestMergeDetectedApps_PreservesPolicy(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	m.MergeDetectedApps([]split.AppEntry{{ID: "slack", Name: "Slack", Path: "/Applications/Slack.app"}})
	require.NoError(t, m.SetAppPolicy("slack", split.PolicyInclude))
	m.MergeDetectedApps([]split.AppEntry{{ID: "slack", Name: "Slack", Path: "/Applications/Slack.app"}})
	apps := m.GetApps()
	require.Len(t, apps, 1)
	assert.Equal(t, split.PolicyInclude, apps[0].Policy)
}

func TestSetDefaultPolicy(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	require.NoError(t, m.SetDefaultPolicy(split.DefaultPolicyBypass))
	assert.Equal(t, split.DefaultPolicyBypass, m.GetConfig().DefaultPolicy)
}

func TestApplyPreset_Work(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	m.MergeDetectedApps([]split.AppEntry{
		{ID: "slack", Name: "Slack", Path: "/Applications/Slack.app"},
		{ID: "zoom", Name: "Zoom", Path: "/Applications/Zoom.app"},
		{ID: "steam", Name: "Steam", Path: "/Applications/Steam.app"},
	})
	require.NoError(t, m.ApplyPreset("work"))
	policyMap := make(map[string]split.Policy)
	for _, a := range m.GetApps() {
		policyMap[a.ID] = a.Policy
	}
	assert.Equal(t, split.PolicyInclude, policyMap["slack"])
	assert.Equal(t, split.PolicyInclude, policyMap["zoom"])
	assert.Equal(t, split.PolicyExclude, policyMap["steam"])
}

func TestApplyPreset_Unknown(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	assert.Error(t, m.ApplyPreset("nonexistent-preset"))
}

func TestSetAppRunning(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	m.MergeDetectedApps([]split.AppEntry{{ID: "chrome", Name: "Chrome", Path: "/Applications/Chrome.app"}})
	m.SetAppRunning("chrome", true)
	apps := m.GetApps()
	require.Len(t, apps, 1)
	assert.True(t, apps[0].Running)
	m.SetAppRunning("chrome", false)
	assert.False(t, m.GetApps()[0].Running)
}

func TestConfigFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "split-tunnel.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid json}"), 0644))
	w := warp.NewClient("warp-cli-nonexistent", "127.0.0.1", "40000")
	m := split.NewManager(path, w)
	assert.Error(t, m.Load())
}

func TestPolicyCycle(t *testing.T) {
	m, _ := newTestManager(t)
	require.NoError(t, m.Load())
	m.MergeDetectedApps([]split.AppEntry{{ID: "firefox", Name: "Firefox", Path: "/Applications/Firefox.app"}})
	for _, p := range []split.Policy{split.PolicyInclude, split.PolicyExclude, split.PolicyDefault} {
		require.NoError(t, m.SetAppPolicy("firefox", p))
		apps := m.GetApps()
		require.Len(t, apps, 1)
		assert.Equal(t, p, apps[0].Policy)
	}
}
