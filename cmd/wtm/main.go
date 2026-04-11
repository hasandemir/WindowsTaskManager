//go:build windows

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/ai"
	"github.com/ersinkoc/WindowsTaskManager/internal/anomaly"
	"github.com/ersinkoc/WindowsTaskManager/internal/collector"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/controller"
	"github.com/ersinkoc/WindowsTaskManager/internal/event"
	"github.com/ersinkoc/WindowsTaskManager/internal/server"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
	"github.com/ersinkoc/WindowsTaskManager/internal/tray"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
	"github.com/ersinkoc/WindowsTaskManager/web"
)

func main() {
	var (
		configPath  = flag.String("config", "", "config file path (default %APPDATA%\\WindowsTaskManager\\config.yaml)")
		showVersion = flag.Bool("version", false, "print version and exit")
		noTray      = flag.Bool("no-tray", false, "disable system tray icon")
		noBrowser   = flag.Bool("no-browser", false, "do not open dashboard in browser")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println("Windows Task Manager 1.0.0")
		return
	}

	cfgPath := resolveConfigPath(*configPath)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("loading config from %s", cfgPath)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Core wiring.
	emitter := event.NewEmitter()
	historyCap := int(cfg.Monitoring.HistoryDuration / cfg.Monitoring.Interval)
	if historyCap < 60 {
		historyCap = 60
	}
	store := storage.NewStore(historyCap, 60)

	cpuName, cpuMHz := collector.CPUInfoFromRegistry()
	mgr := collector.NewManager(cfg, store, emitter, cpuName, cpuMHz)
	ctrl := controller.NewController(cfg, store, emitter)
	alerts := anomaly.NewAlertStore(256)
	engine := anomaly.NewEngine(cfg, store, emitter, alerts)
	engine.SetActuator(ctrl) // rules engine can kill/suspend via the controller
	advisor := ai.NewAdvisor(cfg, store, alerts.Active)

	// Prime first snapshot before serving HTTP.
	mgr.CollectOnce()

	// applyConfig fans a new config out to every long-lived component.
	// It is called both by the file watcher and by the in-process AI
	// config update endpoint.
	applyConfig := func(newCfg *config.Config) {
		mgr.ApplyConfig(newCfg)
		ctrl.SetConfig(newCfg)
		engine.SetConfig(newCfg)
		advisor.SetConfig(newCfg)
	}

	srv := server.New(server.Options{
		Cfg:        cfg,
		CfgPath:    cfgPath,
		OnCfgApply: applyConfig,
		Store:      store,
		Controller: ctrl,
		Alerts:     alerts,
		Emitter:    emitter,
		Advisor:    advisor,
		StaticFS:   web.FS(),
	})

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.Start(rootCtx)
	engine.Start(rootCtx)

	// Config hot-reload watcher.
	watcher := config.NewWatcher(cfgPath, func(newCfg *config.Config) {
		log.Printf("config reloaded")
		applyConfig(newCfg)
		srv.SetConfig(newCfg)
	})
	go watcher.Start(rootCtx)

	// HTTP server in its own goroutine.
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.Start()
	}()

	dashURL := fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)

	if cfg.Server.OpenBrowser && !*noBrowser {
		// Wait briefly for the listener to come up.
		go func() {
			time.Sleep(400 * time.Millisecond)
			_ = winapi.ShellExecute("open", dashURL, "", "", winapi.SW_SHOWNORMAL)
		}()
	}

	var trayWG sync.WaitGroup
	var trayInst *tray.Tray
	if !*noTray {
		trayInst = tray.New(cfg, dashURL, cfgPath, emitter, func() {
			cancel()
			_ = srv.Shutdown(context.Background())
		})
		trayWG.Add(1)
		go func() {
			defer trayWG.Done()
			trayInst.Run()
		}()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-sig:
		log.Printf("signal: %v — shutting down", s)
	case err := <-srvErr:
		log.Printf("http server: %v", err)
	}
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	trayWG.Wait()
}

// resolveConfigPath chooses the active config file.
//
// Precedence:
//  1. --config flag
//  2. config.yaml next to the executable
//  3. %APPDATA%\WindowsTaskManager\config.yaml
func resolveConfigPath(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	exe, err := os.Executable()
	if err == nil {
		local := filepath.Join(filepath.Dir(exe), "config.yaml")
		if _, statErr := os.Stat(local); statErr == nil {
			return local
		}
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = "."
	}
	return filepath.Join(appData, "WindowsTaskManager", "config.yaml")
}
