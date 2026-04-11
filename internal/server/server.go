package server

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/ai"
	"github.com/ersinkoc/WindowsTaskManager/internal/anomaly"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/controller"
	"github.com/ersinkoc/WindowsTaskManager/internal/event"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
)

// AIAdvisor is the minimal interface the server needs from the AI package.
type AIAdvisor interface {
	Enabled() bool
	Status() map[string]any
	Analyze(ctx context.Context, prompt string) (*ai.AnalyzeResult, error)
}

// Server is the HTTP server hosting the REST API, SSE stream, and embedded UI.
type Server struct {
	cfg        *config.Config
	cfgPath    string
	onCfgApply func(*config.Config)
	store      *storage.Store
	controller *controller.Controller
	alerts     *anomaly.AlertStore
	emitter    *event.Emitter
	advisor    AIAdvisor
	staticFS   fs.FS

	router *Router
	hub    *SSEHub

	mu      sync.RWMutex
	httpSrv *http.Server
}

// Options bundles every dependency the Server needs.
type Options struct {
	Cfg        *config.Config
	CfgPath    string
	OnCfgApply func(*config.Config) // invoked after a successful in-process config update
	Store      *storage.Store
	Controller *controller.Controller
	Alerts     *anomaly.AlertStore
	Emitter    *event.Emitter
	Advisor    AIAdvisor
	StaticFS   fs.FS
}

func New(opts Options) *Server {
	s := &Server{
		cfg:        opts.Cfg,
		cfgPath:    opts.CfgPath,
		onCfgApply: opts.OnCfgApply,
		store:      opts.Store,
		controller: opts.Controller,
		alerts:     opts.Alerts,
		emitter:    opts.Emitter,
		advisor:    opts.Advisor,
		staticFS:   opts.StaticFS,
		router:     NewRouter(),
		hub:        NewSSEHub(opts.Emitter),
	}
	s.routes()
	return s
}

// SetConfig hot-swaps the active config (after a watcher reload).
func (s *Server) SetConfig(cfg *config.Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}

// Start begins serving HTTP. Returns when the listener exits.
func (s *Server) Start() error {
	s.mu.Lock()
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.mu.Unlock()
	log.Printf("HTTP server listening on http://%s", addr)
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully terminates the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	srv := s.httpSrv
	s.mu.RUnlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

// loggingMiddleware logs every request with method, path, status, duration.
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying writer so the SSE handler's
// `w.(http.Flusher)` type assertion still works through this wrapper.
// Without this, the stream endpoint returns "streaming unsupported"
// and the dashboard stays stuck on "offline".
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// localOnlyMiddleware rejects any non-loopback request. The HTTP server is
// scoped to local control by design.
func localOnlyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.RemoteAddr
		// Strip port (last colon).
		for i := len(host) - 1; i >= 0; i-- {
			if host[i] == ':' {
				host = host[:i]
				break
			}
		}
		if host != "127.0.0.1" && host != "::1" && host != "localhost" {
			http.Error(w, "forbidden: local-only API", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
