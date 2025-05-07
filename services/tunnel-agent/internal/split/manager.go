// Package split manages per-application split-tunnel configuration.
package split

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mehranredrose/warpture/tunnel-agent/internal/warp"
)

type Policy string

const (
	PolicyInclude Policy = "include"
	PolicyExclude Policy = "exclude"
	PolicyDefault Policy = "default"
)

type DefaultPolicy string

const (
	DefaultPolicyWarp   DefaultPolicy = "warp"
	DefaultPolicyBypass DefaultPolicy = "bypass"
	DefaultPolicySystem DefaultPolicy = "system"
)

type AppEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	BundleID string `json:"bundleId,omitempty"`
	Path     string `json:"path"`
	Icon     string `json:"icon,omitempty"`
	Policy   Policy `json:"policy"`
	Running  bool   `json:"running"`
}

type Config struct {
	Version       int           `json:"version"`
	DefaultPolicy DefaultPolicy `json:"defaultPolicy"`
	IncludedApps  []AppEntry    `json:"includedApps"`
	ExcludedApps  []AppEntry    `json:"excludedApps"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type Preset struct {
	Name     string
	Includes []string
	Excludes []string
}

var builtinPresets = map[string]Preset{
	"work": {
		Name:     "Work",
		Includes: []string{"slack", "zoom", "teams", "notion"},
		Excludes: []string{"steam", "discord", "spotify"},
	},
	"gaming": {
		Name:     "Gaming",
		Excludes: []string{"steam", "epic", "battle.net", "origin"},
		Includes: []string{"slack"},
	},
	"streaming": {
		Name:     "Streaming",
		Excludes: []string{"netflix", "spotify", "youtube", "twitch", "plex"},
		Includes: []string{"slack", "zoom"},
	},
}

type Manager struct {
	configPath string
	warp       *warp.Client
	mu         sync.RWMutex
	config     Config
	apps       map[string]*AppEntry
}

func NewManager(configPath string, w *warp.Client) *Manager {
	return &Manager{
		configPath: configPath,
		warp:       w,
		apps:       make(map[string]*AppEntry),
	}
}

func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if os.IsNotExist(err) {
		m.config = defaultConfig()
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := json.Unmarshal(data, &m.config); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	m.rebuildIndex()
	log.Infof("[split] loaded: %d included, %d excluded", len(m.config.IncludedApps), len(m.config.ExcludedApps))
	return nil
}

func (m *Manager) Save() error {
	m.mu.Lock()
	m.config.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(m.config, "", "  ")
	m.mu.Unlock()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(m.configPath, data, 0644)
}

func (m *Manager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Manager) GetApps() []AppEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]AppEntry, 0, len(m.apps))
	for _, a := range m.apps {
		result = append(result, *a)
	}
	return result
}

func (m *Manager) SetAppPolicy(appID string, policy Policy) error {
	m.mu.Lock()
	entry, ok := m.apps[appID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("app %q not found", appID)
	}
	old := entry.Policy
	entry.Policy = policy
	m.syncConfigFromIndex()
	m.mu.Unlock()

	if err := m.Save(); err != nil {
		m.mu.Lock()
		entry.Policy = old
		m.syncConfigFromIndex()
		m.mu.Unlock()
		return fmt.Errorf("save config: %w", err)
	}
	log.Infof("[split] %s: %s → %s", entry.Name, old, policy)
	return m.applyPolicy(entry)
}

func (m *Manager) SetDefaultPolicy(p DefaultPolicy) error {
	m.mu.Lock()
	m.config.DefaultPolicy = p
	m.mu.Unlock()
	return m.Save()
}

func (m *Manager) SetAppRunning(appID string, running bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry, ok := m.apps[appID]; ok {
		entry.Running = running
	}
}

func (m *Manager) ApplyPreset(presetName string) error {
	preset, ok := builtinPresets[presetName]
	if !ok {
		return fmt.Errorf("unknown preset %q", presetName)
	}
	log.Infof("[split] applying preset: %s", preset.Name)
	m.mu.Lock()
	for _, name := range preset.Includes {
		for _, entry := range m.apps {
			if matchesName(entry.Name, name) {
				entry.Policy = PolicyInclude
			}
		}
	}
	for _, name := range preset.Excludes {
		for _, entry := range m.apps {
			if matchesName(entry.Name, name) {
				entry.Policy = PolicyExclude
			}
		}
	}
	m.syncConfigFromIndex()
	m.mu.Unlock()
	return m.Save()
}

func (m *Manager) MergeDetectedApps(detected []AppEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range detected {
		a := &detected[i]
		if _, exists := m.apps[a.ID]; !exists {
			a.Policy = PolicyDefault
			m.apps[a.ID] = a
		}
	}
}

func (m *Manager) rebuildIndex() {
	m.apps = make(map[string]*AppEntry)
	for i := range m.config.IncludedApps {
		a := m.config.IncludedApps[i]
		a.Policy = PolicyInclude
		m.apps[a.ID] = &a
	}
	for i := range m.config.ExcludedApps {
		a := m.config.ExcludedApps[i]
		a.Policy = PolicyExclude
		m.apps[a.ID] = &a
	}
}

func (m *Manager) syncConfigFromIndex() {
	var included, excluded []AppEntry
	for _, a := range m.apps {
		switch a.Policy {
		case PolicyInclude:
			included = append(included, *a)
		case PolicyExclude:
			excluded = append(excluded, *a)
		}
	}
	m.config.IncludedApps = included
	m.config.ExcludedApps = excluded
}

func (m *Manager) applyPolicy(entry *AppEntry) error {
	switch entry.Policy {
	case PolicyInclude:
		log.Debugf("[split] enforce include: %s (proxy=%s)", entry.Name, m.warp.ProxyAddr())
	case PolicyExclude:
		log.Debugf("[split] enforce exclude: %s", entry.Name)
	default:
		log.Debugf("[split] clear rules: %s", entry.Name)
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		Version:       1,
		DefaultPolicy: DefaultPolicyWarp,
		IncludedApps:  []AppEntry{},
		ExcludedApps:  []AppEntry{},
	}
}

func matchesName(entryName, pattern string) bool {
	return strings.EqualFold(entryName, pattern) ||
		strings.Contains(strings.ToLower(entryName), strings.ToLower(pattern))
}
