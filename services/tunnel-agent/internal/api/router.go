// Package api provides the REST API and WebSocket handler for tunnel-agent.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/mehranredrose/warpture/tunnel-agent/internal/split"
	"github.com/mehranredrose/warpture/tunnel-agent/internal/warp"
)

func NewRouter(w *warp.Client, m *split.Manager, version string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware())
	r.Use(corsMiddleware())

	h := &handler{warp: w, split: m, version: version}

	r.GET("/health", h.health)
	r.GET("/version", h.getVersion)

	v1 := r.Group("/api/v1")

	warpGroup := v1.Group("/warp")
	warpGroup.GET("/status", h.getWarpStatus)
	warpGroup.POST("/connect", h.connect)
	warpGroup.POST("/disconnect", h.disconnect)

	appsGroup := v1.Group("/apps")
	appsGroup.GET("", h.listApps)
	appsGroup.POST("/policy", h.setAppPolicy)
	appsGroup.POST("/running", h.setAppRunning)
	appsGroup.POST("/merge", h.mergeApps)

	cfgGroup := v1.Group("/config")
	cfgGroup.GET("", h.getConfig)
	cfgGroup.POST("/default-policy", h.setDefaultPolicy)

	v1.POST("/presets/apply", h.applyPreset)
	v1.GET("/stats", h.getStats)

	return r
}

type handler struct {
	warp    *warp.Client
	split   *split.Manager
	version string
}

func (h *handler) health(c *gin.Context) {
	status, _ := h.warp.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"status": "ok", "warp": status, "mock": h.warp.IsMockMode(),
		"version": h.version, "timestamp": time.Now().UTC(),
	})
}

func (h *handler) getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": h.version})
}

func (h *handler) getWarpStatus(c *gin.Context) {
	st, err := h.warp.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": st, "mock": h.warp.IsMockMode()})
}

func (h *handler) connect(c *gin.Context) {
	if err := h.warp.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handler) disconnect(c *gin.Context) {
	if err := h.warp.Disconnect(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handler) listApps(c *gin.Context) {
	c.JSON(http.StatusOK, h.split.GetApps())
}

func (h *handler) setAppPolicy(c *gin.Context) {
	var req struct {
		AppID  string       `json:"appId" binding:"required"`
		Policy split.Policy `json:"policy" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.split.SetAppPolicy(req.AppID, req.Policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handler) setAppRunning(c *gin.Context) {
	var req struct {
		AppID   string `json:"appId" binding:"required"`
		Running bool   `json:"running"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.split.SetAppRunning(req.AppID, req.Running)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handler) mergeApps(c *gin.Context) {
	var apps []split.AppEntry
	if err := c.ShouldBindJSON(&apps); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.split.MergeDetectedApps(apps)
	c.JSON(http.StatusOK, gin.H{"ok": true, "total": len(h.split.GetApps())})
}

func (h *handler) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, h.split.GetConfig())
}

func (h *handler) setDefaultPolicy(c *gin.Context) {
	var req struct {
		Policy split.DefaultPolicy `json:"policy" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.split.SetDefaultPolicy(req.Policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handler) applyPreset(c *gin.Context) {
	var req struct {
		Preset string `json:"preset" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.split.ApplyPreset(req.Preset); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *handler) getStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.warp.GetStats())
}

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		log.Debugf("[api] %s %s %d %s", c.Request.Method, path, c.Writer.Status(), time.Since(start))
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
