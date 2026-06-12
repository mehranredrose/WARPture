// Package warp wraps the Cloudflare warp-cli binary and exposes a clean Go API.
// Falls back to mock/simulation mode if warp-cli is not installed.
package warp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Status string

const (
	StatusConnected    Status = "connected"
	StatusDisconnected Status = "disconnected"
	StatusConnecting   Status = "connecting"
)

type Stats struct {
	BytesIn        int64  `json:"bytesIn"`
	BytesOut       int64  `json:"bytesOut"`
	LatencyMs      int    `json:"latencyMs"`
	ConnectedSince string `json:"connectedSince,omitempty"`
	Account        string `json:"account,omitempty"`
	Mode           string `json:"mode,omitempty"`
}

type StatusChangeFunc func(Status)

type Client struct {
	cliPath   string
	proxyHost string
	proxyPort string
	mu        sync.RWMutex
	status    Status
	stats     Stats
	listeners []StatusChangeFunc
	mockMode  bool
}

func NewClient(cliPath, proxyHost, proxyPort string) *Client {
	c := &Client{
		cliPath:   cliPath,
		proxyHost: proxyHost,
		proxyPort: proxyPort,
		status:    StatusDisconnected,
	}
	if err := c.probe(); err != nil {
		log.Warnf("[warp] warp-cli not found (%v) – simulation mode", err)
		c.mockMode = true
	}
	return c
}

func (c *Client) probe() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, c.cliPath, "--version").Run()
}

func (c *Client) OnStatusChange(fn StatusChangeFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, fn)
}

func (c *Client) notifyListeners(s Status) {
	for _, fn := range c.listeners {
		go fn(s)
	}
}

// ── Core WARP commands ────────────────────────────────────────────────────────

func (c *Client) Connect() error {
	if c.mockMode {
		return c.mockConnect()
	}
	_, err := c.run("connect")
	if err != nil {
		return fmt.Errorf("warp connect: %w", err)
	}
	c.setStatus(StatusConnected)
	return nil
}

func (c *Client) Disconnect() error {
	if c.mockMode {
		return c.mockDisconnect()
	}
	_, err := c.run("disconnect")
	if err != nil {
		return fmt.Errorf("warp disconnect: %w", err)
	}
	c.setStatus(StatusDisconnected)
	return nil
}

// GetStatus queries warp-cli status and returns the parsed result.
// Real output looks like: "Status update: Connected" or "Status update: Disconnected"
func (c *Client) GetStatus() (Status, error) {
	if c.mockMode {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.status, nil
	}
	out, err := c.run("status")
	if err != nil {
		return StatusDisconnected, err
	}
	return parseStatus(out), nil
}

func (c *Client) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *Client) IsMockMode() bool { return c.mockMode }
func (c *Client) ProxyAddr() string {
	return fmt.Sprintf("%s:%s", c.proxyHost, c.proxyPort)
}

// ── Split tunnel IP commands (modern warp-cli API) ────────────────────────────

// TunnelIPAdd adds an IP to the split tunnel config.
// In exclude mode: this IP bypasses WARP.
// In include mode: this IP goes through WARP.
func (c *Client) TunnelIPAdd(ip string) error {
	if c.mockMode {
		log.Infof("[warp:mock] tunnel ip add %s", ip)
		return nil
	}
	_, err := c.run("tunnel", "ip", "add", ip)
	return err
}

// TunnelIPRemove removes an IP from the split tunnel config.
func (c *Client) TunnelIPRemove(ip string) error {
	if c.mockMode {
		log.Infof("[warp:mock] tunnel ip remove %s", ip)
		return nil
	}
	_, err := c.run("tunnel", "ip", "remove", ip)
	return err
}

// TunnelIPShow returns the current split tunnel IP list.
func (c *Client) TunnelIPShow() ([]string, error) {
	if c.mockMode {
		return []string{}, nil
	}
	out, err := c.run("tunnel", "ip", "show")
	if err != nil {
		return nil, err
	}
	return parseIPList(out), nil
}

// TunnelHostAdd adds a domain to the split tunnel config.
func (c *Client) TunnelHostAdd(host string) error {
	if c.mockMode {
		log.Infof("[warp:mock] tunnel host add %s", host)
		return nil
	}
	_, err := c.run("tunnel", "host", "add", host)
	return err
}

// TunnelHostRemove removes a domain from the split tunnel config.
func (c *Client) TunnelHostRemove(host string) error {
	if c.mockMode {
		log.Infof("[warp:mock] tunnel host remove %s", host)
		return nil
	}
	_, err := c.run("tunnel", "host", "remove", host)
	return err
}

// TunnelHostShow returns the current split tunnel host list.
func (c *Client) TunnelHostShow() ([]string, error) {
	if c.mockMode {
		return []string{}, nil
	}
	out, err := c.run("tunnel", "host", "show")
	if err != nil {
		return nil, err
	}
	return parseHostList(out), nil
}

// GetSettings returns parsed warp-cli settings list output.
func (c *Client) GetSettings() (map[string]string, error) {
	if c.mockMode {
		return map[string]string{
			"Mode":              "WarpWithDnsOverHttps",
			"Split Tunnel Mode": "Exclude",
			"Always On":         "true",
		}, nil
	}
	out, err := c.run("settings", "list")
	if err != nil {
		return nil, err
	}
	return parseSettings(out), nil
}

// ── Health Monitor ─────────────────────────────────────────────────────────────

func (c *Client) HealthMonitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	consecutive := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st, err := c.GetStatus()
			if err != nil {
				log.Warnf("[warp] health check error: %v", err)
				continue
			}
			c.mu.RLock()
			prev := c.status
			c.mu.RUnlock()
			if st != prev {
				c.setStatus(st)
			}
			if st == StatusConnected {
				consecutive = 0
				c.refreshStats()
			} else {
				consecutive++
				if consecutive >= 3 && prev == StatusConnected {
					log.Warn("[warp] unexpected disconnect – attempting reconnect")
					_ = c.Connect()
					consecutive = 0
				}
			}
		}
	}
}

// ── Internals ─────────────────────────────────────────────────────────────────

func (c *Client) run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.cliPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w; stderr: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (c *Client) setStatus(s Status) {
	c.mu.Lock()
	old := c.status
	c.status = s
	c.mu.Unlock()
	if old != s {
		log.Infof("[warp] status: %s → %s", old, s)
		c.notifyListeners(s)
	}
}

func (c *Client) refreshStats() {
	if c.mockMode {
		c.mu.Lock()
		c.stats.LatencyMs = 12 + int(time.Now().UnixMilli()%20)
		c.stats.BytesIn += 1024 * 50
		c.stats.BytesOut += 1024 * 10
		c.stats.Account = "Team (simulated)"
		c.stats.Mode = "proxy"
		c.mu.Unlock()
		return
	}
	out, err := c.run("tunnel", "stats")
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	parseStatsInto(out, &c.stats)
}

func (c *Client) mockConnect() error {
	log.Info("[warp:mock] connecting...")
	c.setStatus(StatusConnecting)
	time.Sleep(800 * time.Millisecond)
	c.setStatus(StatusConnected)
	c.mu.Lock()
	c.stats.ConnectedSince = time.Now().UTC().Format(time.RFC3339)
	c.stats.Account = "Team (simulated)"
	c.stats.Mode = "warp"
	c.mu.Unlock()
	return nil
}

func (c *Client) mockDisconnect() error {
	time.Sleep(300 * time.Millisecond)
	c.setStatus(StatusDisconnected)
	return nil
}

// ── Parsers ───────────────────────────────────────────────────────────────────

// parseStatus handles both old and new warp-cli output formats:
// Old: "Status update: Connected"
// New: "Connected" or just the word
var statusRe = regexp.MustCompile(`(?i)(status update:|status:)?\s*(connected|disconnected|connecting|unable to connect)`)

func parseStatus(out string) Status {
	m := statusRe.FindStringSubmatch(strings.ToLower(out))
	if len(m) < 3 {
		// Fallback: check if the word appears anywhere
		lower := strings.ToLower(out)
		if strings.Contains(lower, "connected") && !strings.Contains(lower, "disconnected") {
			return StatusConnected
		}
		if strings.Contains(lower, "connecting") {
			return StatusConnecting
		}
		return StatusDisconnected
	}
	switch strings.TrimSpace(m[2]) {
	case "connected":
		return StatusConnected
	case "connecting":
		return StatusConnecting
	default:
		return StatusDisconnected
	}
}

var latencyRe = regexp.MustCompile(`(?i)latency[:\s]+(\d+)\s*ms`)

func parseStatsInto(out string, s *Stats) {
	if m := latencyRe.FindStringSubmatch(out); len(m) >= 2 {
		v, _ := strconv.Atoi(m[1])
		s.LatencyMs = v
	}
}

func parseIPList(out string) []string {
	var ips []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			ips = append(ips, line)
		}
	}
	return ips
}

func parseHostList(out string) []string {
	return parseIPList(out) // same format
}

func parseSettings(out string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			// Strip "(derived)", "(network policy)", "(user set)" suffixes
			if idx := strings.Index(v, "("); idx > 0 {
				v = strings.TrimSpace(v[:idx])
			}
			result[k] = v
		}
	}
	return result
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}