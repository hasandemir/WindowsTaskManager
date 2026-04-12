package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
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
	Chat(ctx context.Context, message string) (*ai.AnalyzeResult, error)
	BackgroundState() ai.BackgroundState
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
	version    string
	csrfToken  string

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
	Version    string
}

func New(opts Options) *Server {
	version := opts.Version
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
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
		version:    version,
		csrfToken:  newCSRFSafeToken(),
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

func securityHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		next(w, r)
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
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "forbidden: invalid remote address", http.StatusForbidden)
			return
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden: local-only API", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func mutationGuardMiddleware(csrfToken string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next(w, r)
				return
			}
			if !validOrigin(r) {
				http.Error(w, "forbidden: invalid origin", http.StatusForbidden)
				return
			}
			if r.Header.Get("X-WTM-CSRF") != csrfToken {
				http.Error(w, "forbidden: missing csrf token", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func validOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if !(strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())) {
		return false
	}
	return samePort(u.Port(), requestPort(r.Host))
}

func requestPort(hostport string) string {
	if host, port, err := net.SplitHostPort(hostport); err == nil {
		_ = host
		return port
	}
	return ""
}

func samePort(originPort, requestPort string) bool {
	if originPort == "" {
		originPort = "80"
	}
	if requestPort == "" {
		requestPort = "80"
	}
	return originPort == requestPort
}

func newCSRFSafeToken() string {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}
