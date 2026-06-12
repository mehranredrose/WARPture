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

// AppEntry represents one tracked application.
type AppEntry struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	BundleID string   `json:"bundleId,omitempty"`
	Path     string   `json:"path"`
	Icon     string   `json:"icon,omitempty"`
	Policy   Policy   `json:"policy"`
	Running  bool     `json:"running"`
	// Domains associated with this app for split tunnel routing
	Domains  []string `json:"domains,omitempty"`
}

type Config struct {
	Version       int           `json:"version"`
	DefaultPolicy DefaultPolicy `json:"defaultPolicy"`
	IncludedApps  []AppEntry    `json:"includedApps"`
	ExcludedApps  []AppEntry    `json:"excludedApps"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// appDomains maps known app IDs to their associated domains.
// These are added to / removed from warp-cli tunnel host list.
var appDomains = map[string][]string{
	"chrome":      {"google.com", "googleapis.com", "gstatic.com", "googleusercontent.com"},
	"firefox":     {"mozilla.org", "mozilla.net", "firefox.com"},
	"slack":       {"slack.com", "slack-edge.com", "slack-msgs.com", "slack-redir.net"},
	"zoom":        {"zoom.us", "zoom.com", "zoomgov.com", "zmtrial.com"},
	"discord":     {"discord.com", "discord.gg", "discordapp.com", "discordapp.net"},
	"spotify":     {"spotify.com", "scdn.co", "spotifycdn.com"},
	"steam":       {"steampowered.com", "steamcontent.com", "steamgames.com", "steamstatic.com"},
	"teams":       {"teams.microsoft.com", "microsoft.com", "office.com", "office365.com"},
	"telegram":    {"telegram.org", "t.me", "telegra.ph"},
	"signal":      {"signal.org", "whispersystems.org"},
	"notion":      {"notion.so", "notion-static.com"},
	"figma":       {"figma.com", "figstatic.com"},
	"dropbox":     {"dropbox.com", "dropboxstatic.com", "dropboxapi.com"},
	"nordvpn":     {"nordvpn.com", "nordvpnaccesstoken.com"},
	"plex":        {"plex.tv", "plex.direct"},
	"twitch":      {"twitch.tv", "twitchapps.com", "jtvnw.net"},
	"netflix":     {"netflix.com", "nflxvideo.net", "nflximg.net"},
	"whatsapp":    {"whatsapp.com", "whatsapp.net"},
	"skype":       {"skype.com", "skypeassets.com"},
	"obsidian":    {"obsidian.md"},
	"linear":      {"linear.app"},
	"bitwarden":   {"bitwarden.com", "bitwarden.net"},
	"1password":   {"1password.com", "1passwordusercontent.com"},
	"vscode":      {"vscode.dev", "marketplace.visualstudio.com", "update.code.visualstudio.com"},
	"brave":       {"brave.com", "bravesoftware.com"},
	"edge":        {"edge.microsoft.com"},
	"opera":       {"opera.com"},
	"thunderbird": {"thunderbird.net", "mozilla.org"},
	"vlc":         {"videolan.org"},
	"gimp":        {"gimp.org"},
	"inkscape":    {"inkscape.org"},
	"libreoffice": {"libreoffice.org", "documentfoundation.org"},
	"cyberduck":   {"cyberduck.io"},
	"postman":     {"postman.com", "getpostman.com"},
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
	// Re-apply all rules on startup
	go m.reapplyAll()
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
		return fmt.Errorf("mkdir: %w", err)
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

// SetAppPolicy updates an app's policy and immediately applies it to warp-cli.
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

	// Apply the routing change
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

func (m *Manager) MergeDetectedApps(detected []AppEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range detected {
		a := &detected[i]
		if _, exists := m.apps[a.ID]; !exists {
			a.Policy = PolicyDefault
			// Attach known domains
			if domains, ok := appDomains[a.ID]; ok {
				a.Domains = domains
			}
			m.apps[a.ID] = a
		}
	}
}

var builtinPresets = map[string]struct {
	includes []string
	excludes []string
}{
	"work": {
		includes: []string{"slack", "zoom", "teams", "notion", "linear"},
		excludes: []string{"steam", "discord", "spotify"},
	},
	"gaming": {
		excludes: []string{"steam", "epic-games", "battle-net"},
		includes: []string{"slack"},
	},
	"streaming": {
		excludes: []string{"netflix", "spotify", "plex", "twitch"},
		includes: []string{"slack", "zoom"},
	},
}

func (m *Manager) ApplyPreset(presetName string) error {
	preset, ok := builtinPresets[presetName]
	if !ok {
		return fmt.Errorf("unknown preset %q", presetName)
	}
	m.mu.Lock()
	for _, name := range preset.includes {
		for _, entry := range m.apps {
			if matchesName(entry.Name, name) || entry.ID == name {
				entry.Policy = PolicyInclude
			}
		}
	}
	for _, name := range preset.excludes {
		for _, entry := range m.apps {
			if matchesName(entry.Name, name) || entry.ID == name {
				entry.Policy = PolicyExclude
			}
		}
	}
	m.syncConfigFromIndex()
	m.mu.Unlock()

	if err := m.Save(); err != nil {
		return err
	}
	// Apply all rules
	go m.reapplyAll()
	return nil
}

// ── Routing implementation ────────────────────────────────────────────────────

// applyPolicy adds or removes warp-cli tunnel rules for the app's domains.
// 
// How it works with WARP split tunnels:
//
// WARP default mode is "Exclude" — all traffic goes through WARP except
// the domains/IPs in the exclude list.
//
// PolicyExclude → add app domains to exclude list  (app bypasses WARP)
// PolicyInclude → remove app domains from exclude list (app uses WARP)
// PolicyDefault → remove from exclude list (follows global default)
func (m *Manager) applyPolicy(entry *AppEntry) error {
	domains := m.getDomainsForApp(entry)
	if len(domains) == 0 {
		log.Debugf("[split] no domains for %s, skipping routing", entry.Name)
		return nil
	}

	switch entry.Policy {
	case PolicyExclude:
		// Add to exclude list → app bypasses WARP
		log.Infof("[split] excluding %s: adding %d domains to tunnel host list", entry.Name, len(domains))
		for _, domain := range domains {
			if err := m.warp.TunnelHostAdd(domain); err != nil {
				log.Warnf("[split] tunnel host add %s: %v", domain, err)
			}
		}

	case PolicyInclude:
		// Remove from exclude list → app goes through WARP
		log.Infof("[split] including %s: removing %d domains from tunnel host list", entry.Name, len(domains))
		for _, domain := range domains {
			if err := m.warp.TunnelHostRemove(domain); err != nil {
				log.Debugf("[split] tunnel host remove %s: %v (may not exist)", domain, err)
			}
		}

	case PolicyDefault:
		// Remove from exclude list → follows global policy
		log.Debugf("[split] resetting %s to default policy", entry.Name)
		for _, domain := range domains {
			if err := m.warp.TunnelHostRemove(domain); err != nil {
				log.Debugf("[split] tunnel host remove %s: %v", domain, err)
			}
		}
	}

	return nil
}

// reapplyAll re-applies all routing rules from scratch.
// Called on startup and after preset changes.
func (m *Manager) reapplyAll() {
	m.mu.RLock()
	apps := make([]AppEntry, 0, len(m.apps))
	for _, a := range m.apps {
		apps = append(apps, *a)
	}
	m.mu.RUnlock()

	for i := range apps {
		if apps[i].Policy != PolicyDefault {
			if err := m.applyPolicy(&apps[i]); err != nil {
				log.Warnf("[split] reapply error for %s: %v", apps[i].Name, err)
			}
		}
	}
}

func (m *Manager) getDomainsForApp(entry *AppEntry) []string {
	// Use the entry's own domains list if set
	if len(entry.Domains) > 0 {
		return entry.Domains
	}
	// Fall back to the built-in database
	if domains, ok := appDomains[entry.ID]; ok {
		return domains
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (m *Manager) rebuildIndex() {
	m.apps = make(map[string]*AppEntry)
	for i := range m.config.IncludedApps {
		a := m.config.IncludedApps[i]
		a.Policy = PolicyInclude
		if len(a.Domains) == 0 {
			a.Domains = appDomains[a.ID]
		}
		m.apps[a.ID] = &a
	}
	for i := range m.config.ExcludedApps {
		a := m.config.ExcludedApps[i]
		a.Policy = PolicyExclude
		if len(a.Domains) == 0 {
			a.Domains = appDomains[a.ID]
		}
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