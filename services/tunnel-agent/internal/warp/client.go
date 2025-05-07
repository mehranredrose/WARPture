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

type AccountInfo struct {
	AccountType string `json:"accountType"`
	Mode        string `json:"mode"`
	DeviceName  string `json:"deviceName"`
}

type Stats struct {
	BytesIn        int64  `json:"bytesIn"`
	BytesOut       int64  `json:"bytesOut"`
	LatencyMs      int    `json:"latencyMs"`
	ConnectedSince string `json:"connectedSince,omitempty"`
	Account        string `json:"account,omitempty"`
	Mode           string `json:"mode,omitempty"`
}

type StatusChangeFunc func(newStatus Status)

type Client struct {
	cliPath   string
	proxyHost string
	proxyPort string
	mu        sync.RWMutex
	status    Status
	stats     Stats
	account   AccountInfo
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

func (c *Client) Connect() error {
	if c.mockMode {
		return c.mockConnect()
	}
	out, err := c.run("connect")
	if err != nil {
		return fmt.Errorf("warp connect: %w (output: %s)", err, out)
	}
	c.setStatus(StatusConnected)
	return nil
}

func (c *Client) Disconnect() error {
	if c.mockMode {
		return c.mockDisconnect()
	}
	out, err := c.run("disconnect")
	if err != nil {
		return fmt.Errorf("warp disconnect: %w (output: %s)", err, out)
	}
	c.setStatus(StatusDisconnected)
	return nil
}

func (c *Client) GetStatus() (Status, error) {
	if c.mockMode {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.status, nil
	}
	out, err := c.run("status")
	if err != nil {
		return StatusDisconnected, fmt.Errorf("warp status: %w", err)
	}
	return parseStatus(out), nil
}

func (c *Client) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *Client) SetMode(mode string) error {
	if c.mockMode {
		log.Infof("[warp:mock] set-mode %s", mode)
		return nil
	}
	_, err := c.run("set-mode", mode)
	return err
}

func (c *Client) SetProxy(enabled bool) error {
	mode := "proxy"
	if !enabled {
		mode = "warp"
	}
	return c.SetMode(mode)
}

func (c *Client) AddSplitTunnelExclude(cidr string) error {
	if c.mockMode {
		log.Infof("[warp:mock] add-split-tunnel-exclude %s", cidr)
		return nil
	}
	_, err := c.run("add-split-tunnel-exclude", cidr)
	return err
}

func (c *Client) RemoveSplitTunnelExclude(cidr string) error {
	if c.mockMode {
		log.Infof("[warp:mock] remove-split-tunnel-exclude %s", cidr)
		return nil
	}
	_, err := c.run("remove-split-tunnel-exclude", cidr)
	return err
}

func (c *Client) GetAccount() (AccountInfo, error) {
	if c.mockMode {
		return AccountInfo{AccountType: "Team (simulated)", Mode: "proxy", DeviceName: "WARPture-Dev"}, nil
	}
	out, err := c.run("account")
	if err != nil {
		return AccountInfo{}, err
	}
	return parseAccount(out), nil
}

func (c *Client) ProxyAddr() string {
	return fmt.Sprintf("%s:%s", c.proxyHost, c.proxyPort)
}

func (c *Client) IsMockMode() bool { return c.mockMode }

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
	out, err := c.run("stats")
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	parseStatsInto(out, &c.stats)
}

var statusRe = regexp.MustCompile(`(?i)status:\s*(\w+)`)

func parseStatus(out string) Status {
	m := statusRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return StatusDisconnected
	}
	switch strings.ToLower(m[1]) {
	case "connected":
		return StatusConnected
	case "connecting":
		return StatusConnecting
	default:
		return StatusDisconnected
	}
}

var latencyRe = regexp.MustCompile(`(?i)latency:\s*(\d+)\s*ms`)

func parseStatsInto(out string, s *Stats) {
	if m := latencyRe.FindStringSubmatch(out); len(m) >= 2 {
		v, _ := strconv.Atoi(m[1])
		s.LatencyMs = v
	}
}

func parseAccount(out string) AccountInfo {
	var info AccountInfo
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(strings.ToLower(parts[0]))
		v := strings.TrimSpace(parts[1])
		switch k {
		case "account type":
			info.AccountType = v
		case "mode":
			info.Mode = v
		case "device name":
			info.DeviceName = v
		}
	}
	return info
}

func (c *Client) mockConnect() error {
	log.Info("[warp:mock] connecting...")
	c.setStatus(StatusConnecting)
	time.Sleep(800 * time.Millisecond)
	c.setStatus(StatusConnected)
	c.mu.Lock()
	c.stats.ConnectedSince = time.Now().UTC().Format(time.RFC3339)
	c.stats.Account = "Team (simulated)"
	c.stats.Mode = "proxy"
	c.mu.Unlock()
	return nil
}

func (c *Client) mockDisconnect() error {
	log.Info("[warp:mock] disconnecting...")
	time.Sleep(300 * time.Millisecond)
	c.setStatus(StatusDisconnected)
	return nil
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}
