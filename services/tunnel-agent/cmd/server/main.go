// WARPture – tunnel-agent
// REST + WebSocket middleware between warp-gui and warp-cli.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/mehranredrose/warpture/tunnel-agent/internal/api"
	"github.com/mehranredrose/warpture/tunnel-agent/internal/split"
	"github.com/mehranredrose/warpture/tunnel-agent/internal/warp"
)

// Version is injected at build time via -ldflags.
var Version = "1.0.0"

func main() {
	// ── Logging ────────────────────────────────────────────────────────────
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		lvl = log.InfoLevel
	}
	log.SetLevel(lvl)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})

	log.Infof("WARPture tunnel-agent v%s starting", Version)

	// ── Config ─────────────────────────────────────────────────────────────
	cfg := loadConfig()

	// ── Services ───────────────────────────────────────────────────────────
	warpClient := warp.NewClient(cfg.WarpCLIPath, cfg.ProxyHost, cfg.ProxyPort)
	splitMgr := split.NewManager(cfg.ConfigPath, warpClient)

	if err := splitMgr.Load(); err != nil {
		log.Warnf("Could not load split-tunnel config: %v (will use defaults)", err)
	}

	// Start WARP health monitor (auto-reconnect)
	ctx, cancel := context.WithCancel(context.Background())
	go warpClient.HealthMonitor(ctx)

	// ── HTTP/WS Server ─────────────────────────────────────────────────────
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := api.NewRouter(warpClient, splitMgr, Version)

	httpAddr := cfg.HTTPAddr
	wsAddr := cfg.WSAddr

	httpSrv := &http.Server{
		Addr:         httpAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// WebSocket is served on the same router, different port for clarity
	wsSrv := &http.Server{
		Addr:    wsAddr,
		Handler: api.NewWSHandler(warpClient, splitMgr),
	}

	go func() {
		log.Infof("REST API listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("REST server error: %v", err)
		}
	}()

	go func() {
		log.Infof("WebSocket server listening on %s", wsAddr)
		if err := wsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("WS server error: %v", err)
		}
	}()

	// ── Graceful Shutdown ──────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down tunnel-agent...")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = httpSrv.Shutdown(shutCtx)
	_ = wsSrv.Shutdown(shutCtx)

	log.Info("tunnel-agent stopped")
}

// config holds runtime configuration loaded from env vars.
type config struct {
	WarpCLIPath string
	ProxyHost   string
	ProxyPort   string
	ConfigPath  string
	HTTPAddr    string
	WSAddr      string
}

func loadConfig() config {
	getEnv := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}

	home, _ := os.UserHomeDir()
	return config{
		WarpCLIPath: getEnv("WARP_CLI_PATH", "warp-cli"),
		ProxyHost:   getEnv("PROXY_HOST", "127.0.0.1"),
		ProxyPort:   getEnv("PROXY_PORT", "40000"),
		ConfigPath:  getEnv("CONFIG_PATH", home+"/.config/warp-gui/split-tunnel.json"),
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		WSAddr:      getEnv("WS_ADDR", ":8081"),
	}
}
