package server

import (
	"io"
	"net/http"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/controller"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

// routes wires every endpoint onto the router.
func (s *Server) routes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(localOnlyMiddleware)

	// System endpoints.
	s.router.GET("/api/v1/system", s.handleSystem)
	s.router.GET("/api/v1/cpu", s.handleCPU)
	s.router.GET("/api/v1/memory", s.handleMemory)
	s.router.GET("/api/v1/gpu", s.handleGPU)
	s.router.GET("/api/v1/disk", s.handleDisk)
	s.router.GET("/api/v1/network", s.handleNetwork)
	s.router.GET("/api/v1/history", s.handleHistory)
	s.router.GET("/api/v1/info", s.handleInfo)
	s.router.GET("/api/v1/health", s.handleHealth)

	// Process endpoints.
	s.router.GET("/api/v1/processes", s.handleProcesses)
	s.router.GET("/api/v1/processes/tree", s.handleProcessTree)
	s.router.GET("/api/v1/processes/:pid", s.handleProcessByID)
	s.router.GET("/api/v1/processes/:pid/history", s.handleProcessHistory)
	s.router.POST("/api/v1/processes/:pid/kill", s.handleKill)
	s.router.POST("/api/v1/processes/:pid/kill-tree", s.handleKillTree)
	s.router.POST("/api/v1/processes/:pid/suspend", s.handleSuspend)
	s.router.POST("/api/v1/processes/:pid/resume", s.handleResume)
	s.router.POST("/api/v1/processes/:pid/priority", s.handlePriority)
	s.router.POST("/api/v1/processes/:pid/affinity", s.handleAffinity)
	s.router.POST("/api/v1/processes/:pid/limit", s.handleLimit)
	s.router.DELETE("/api/v1/processes/:pid/limit", s.handleClearLimit)
	s.router.GET("/api/v1/processes/limits", s.handleListLimits)

	// Port and connection endpoints.
	s.router.GET("/api/v1/ports", s.handlePorts)

	// Anomaly endpoints.
	s.router.GET("/api/v1/alerts", s.handleAlerts)
	s.router.GET("/api/v1/alerts/history", s.handleAlertHistory)
	s.router.POST("/api/v1/alerts/clear", s.handleAlertsClear)

	// Config endpoints.
	s.router.GET("/api/v1/config", s.handleConfigGet)

	// AI advisor endpoints.
	s.router.GET("/api/v1/ai/status", s.handleAIStatus)
	s.router.POST("/api/v1/ai/analyze", s.handleAIAnalyze)
	s.router.POST("/api/v1/ai/execute", s.handleAIExecute)
	s.router.GET("/api/v1/ai/config", s.handleAIConfigGet)
	s.router.POST("/api/v1/ai/config", s.handleAIConfigUpdate)
	s.router.GET("/api/v1/ai/presets", s.handleAIPresets)
	s.router.GET("/api/v1/ai/models", s.handleAIModels)

	// User-defined automation rules.
	s.router.GET("/api/v1/rules", s.handleRulesGet)
	s.router.POST("/api/v1/rules", s.handleRulesUpdate)

	// Per-process protect / ignore toggles.
	s.router.POST("/api/v1/config/protect", s.handleConfigProtectToggle)
	s.router.POST("/api/v1/config/ignore", s.handleConfigIgnoreToggle)

	// SSE stream and embedded UI.
	s.router.GET("/api/v1/stream", s.hub.Handler())
	s.router.SetNotFound(s.serveStatic)
}

// ----- system handlers -----

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	snap := s.store.Latest()
	if snap == nil {
		writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (s *Server) handleCPU(w http.ResponseWriter, r *http.Request) {
	if snap := s.store.Latest(); snap != nil {
		writeJSON(w, http.StatusOK, snap.CPU)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
}

func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	if snap := s.store.Latest(); snap != nil {
		writeJSON(w, http.StatusOK, snap.Memory)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
}

func (s *Server) handleGPU(w http.ResponseWriter, r *http.Request) {
	if snap := s.store.Latest(); snap != nil {
		writeJSON(w, http.StatusOK, snap.GPU)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
}

func (s *Server) handleDisk(w http.ResponseWriter, r *http.Request) {
	if snap := s.store.Latest(); snap != nil {
		writeJSON(w, http.StatusOK, snap.Disk)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
}

func (s *Server) handleNetwork(w http.ResponseWriter, r *http.Request) {
	if snap := s.store.Latest(); snap != nil {
		writeJSON(w, http.StatusOK, snap.Network)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	since := r.URL.Query().Get("since")
	if since == "" {
		writeJSON(w, http.StatusOK, s.store.SystemHistory())
		return
	}
	secs, err := strconv.ParseInt(since, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_since", "since must be unix seconds")
		return
	}
	writeJSON(w, http.StatusOK, s.store.SystemHistorySince(time.Unix(secs, 0)))
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":         "1.0.0",
		"go_version":      runtime.Version(),
		"num_cpu":         runtime.NumCPU(),
		"goroutines":      runtime.NumGoroutine(),
		"interval_ms":     cfg.Monitoring.Interval.Milliseconds(),
		"history_minutes": cfg.Monitoring.HistoryDuration.Minutes(),
		"sse_clients":     s.hub.ClientCount(),
		"tracked_pids":    s.store.TrackedProcessCount(),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ----- process handlers -----

func (s *Server) handleProcesses(w http.ResponseWriter, r *http.Request) {
	snap := s.store.Latest()
	if snap == nil {
		writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
		return
	}
	q := r.URL.Query()
	procs := append([]metrics.ProcessInfo(nil), snap.Processes...)

	if filter := strings.ToLower(q.Get("name")); filter != "" {
		out := procs[:0]
		for _, p := range procs {
			if strings.Contains(strings.ToLower(p.Name), filter) {
				out = append(out, p)
			}
		}
		procs = out
	}

	sortBy := q.Get("sort")
	if sortBy == "" {
		sortBy = "cpu"
	}
	desc := q.Get("order") != "asc"
	sortProcesses(procs, sortBy, desc)

	if limitStr := q.Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n < len(procs) {
			procs = procs[:n]
		}
	}
	writeJSON(w, http.StatusOK, procs)
}

func sortProcesses(procs []metrics.ProcessInfo, by string, desc bool) {
	less := func(i, j int) bool { return procs[i].PID < procs[j].PID }
	switch by {
	case "cpu":
		less = func(i, j int) bool { return procs[i].CPUPercent < procs[j].CPUPercent }
	case "memory", "ws":
		less = func(i, j int) bool { return procs[i].WorkingSet < procs[j].WorkingSet }
	case "private":
		less = func(i, j int) bool { return procs[i].PrivateBytes < procs[j].PrivateBytes }
	case "name":
		less = func(i, j int) bool { return procs[i].Name < procs[j].Name }
	case "pid":
		less = func(i, j int) bool { return procs[i].PID < procs[j].PID }
	case "threads":
		less = func(i, j int) bool { return procs[i].ThreadCount < procs[j].ThreadCount }
	}
	if desc {
		orig := less
		less = func(i, j int) bool { return orig(j, i) }
	}
	sort.Slice(procs, less)
}

func (s *Server) handleProcessTree(w http.ResponseWriter, r *http.Request) {
	snap := s.store.Latest()
	if snap == nil {
		writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
		return
	}
	writeJSON(w, http.StatusOK, snap.ProcessTree)
}

func (s *Server) handleProcessByID(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	snap := s.store.Latest()
	if snap == nil {
		writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
		return
	}
	for i := range snap.Processes {
		if snap.Processes[i].PID == pid {
			writeJSON(w, http.StatusOK, snap.Processes[i])
			return
		}
	}
	writeError(w, http.StatusNotFound, "not_found", "no such pid")
}

func (s *Server) handleProcessHistory(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ProcessHistory(pid))
}

func (s *Server) handleKill(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	confirm := r.URL.Query().Get("confirm") == "true"
	if err := s.controller.Kill(pid, confirm); err != nil {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid})
}

func (s *Server) handleKillTree(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	confirm := r.URL.Query().Get("confirm") == "true"
	killed, err := s.controller.KillTree(pid, confirm)
	if err != nil && killed == 0 {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "killed": killed})
}

func (s *Server) handleSuspend(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	if err := s.controller.Suspend(pid); err != nil {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid})
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	if err := s.controller.Resume(pid); err != nil {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid})
}

func (s *Server) handlePriority(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	var body struct {
		Class string `json:"class"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := s.controller.SetPriority(pid, body.Class); err != nil {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid, "class": body.Class})
}

func (s *Server) handleAffinity(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	var body struct {
		Mask uint64 `json:"mask"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := s.controller.SetAffinity(pid, body.Mask); err != nil {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid, "mask": body.Mask})
}

func (s *Server) handleLimit(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	var body struct {
		CPUPct   int    `json:"cpu_pct"`
		MemBytes uint64 `json:"mem_bytes"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := s.controller.Limit(pid, body.CPUPct, body.MemBytes); err != nil {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid})
}

func (s *Server) handleClearLimit(w http.ResponseWriter, r *http.Request) {
	pid, ok := parseUint32Param(w, r, "pid")
	if !ok {
		return
	}
	if err := s.controller.ClearLimit(pid); err != nil {
		s.controllerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid})
}

func (s *Server) handleListLimits(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.controller.ActiveLimits())
}

func (s *Server) controllerError(w http.ResponseWriter, err error) {
	switch err {
	case controller.ErrProtected:
		writeError(w, http.StatusForbidden, "protected", err.Error())
	case controller.ErrCritical:
		writeError(w, http.StatusForbidden, "critical", err.Error())
	case controller.ErrConfirmNeeded:
		writeError(w, http.StatusConflict, "confirm_required", err.Error())
	case controller.ErrNotFound:
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
	}
}

// ----- ports / alerts / config / ai handlers -----

func (s *Server) handlePorts(w http.ResponseWriter, r *http.Request) {
	snap := s.store.Latest()
	if snap == nil {
		writeError(w, http.StatusServiceUnavailable, "no_data", "no snapshot yet")
		return
	}
	writeJSON(w, http.StatusOK, snap.PortBindings)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.alerts.Active())
}

func (s *Server) handleAlertHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.alerts.History())
}

// handleAlertsClear wipes the active alert set without touching history.
// The UI exposes this as a single "Clear" button so the user can recover
// from a noisy burst without restarting the whole process.
func (s *Server) handleAlertsClear(w http.ResponseWriter, r *http.Request) {
	removed := s.alerts.ClearAll()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "removed": removed})
}

func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	if s.advisor == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	writeJSON(w, http.StatusOK, s.advisor.Status())
}

func (s *Server) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	if s.advisor == nil || !s.advisor.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "ai_disabled", "AI advisor not configured")
		return
	}
	var body struct {
		Prompt string `json:"prompt"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	resp, err := s.advisor.Analyze(r.Context(), body.Prompt)
	if err != nil {
		writeError(w, http.StatusBadGateway, "ai_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"answer":  resp.Answer,
		"actions": resp.Actions,
	})
}

// ----- static UI handler -----

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if s.staticFS == nil {
		http.NotFound(w, r)
		return
	}
	upath := strings.TrimPrefix(r.URL.Path, "/")
	if upath == "" {
		upath = "index.html"
	}
	f, err := s.staticFS.Open(upath)
	if err != nil {
		// SPA fallback to index.html.
		f2, err2 := s.staticFS.Open("index.html")
		if err2 != nil {
			http.NotFound(w, r)
			return
		}
		defer f2.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, f2)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentTypeFor(upath))
	_, _ = io.Copy(w, f)
}

func contentTypeFor(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	}
	return "application/octet-stream"
}
