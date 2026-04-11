package config

import (
	"fmt"
	"time"
)

// CurrentSchemaVersion is bumped whenever we ship a breaking defaults change
// that should be forced onto existing user configs. Loader checks this after
// unmarshal and resets the relevant sections to the new defaults.
//
//	1 — original layout
//	2 — turn off hung/orphan/port/network detectors by default; only real
//	    dangers (runaway CPU, memory leak, giant spawn storms) fire by default
const CurrentSchemaVersion = 2

// Config is the root configuration tree.
type Config struct {
	SchemaVersion  int                 `yaml:"schema_version"`
	Server         ServerConfig        `yaml:"server"`
	Monitoring     MonitoringConfig    `yaml:"monitoring"`
	Controller     ControllerConfig    `yaml:"controller"`
	Anomaly        AnomalyConfig       `yaml:"anomaly"`
	Notifications  NotificationsConfig `yaml:"notifications"`
	WellKnownPorts map[uint16]string   `yaml:"well_known_ports"`
	AI             AIConfig            `yaml:"ai"`
	UI             UIConfig            `yaml:"ui"`
	Rules          []Rule              `yaml:"rules"`
}

// Rule is a single user-defined automation rule. It says "when a process
// matching `match` satisfies `when` for at least `for` seconds, do `action`".
// Rules are evaluated on every anomaly tick and hot-reload with the config.
type Rule struct {
	Name       string        `yaml:"name" json:"name"`
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	Match      string        `yaml:"match" json:"match"`                 // case-insensitive substring of process name
	Metric     string        `yaml:"metric" json:"metric"`               // "cpu_percent" | "memory_bytes" | "thread_count"
	Op         string        `yaml:"op" json:"op"`                       // ">" | ">=" | "<" | "<=" (default ">=")
	Threshold  float64       `yaml:"threshold" json:"threshold"`         // bytes for memory, percent for cpu, count for threads
	For        time.Duration `yaml:"for" json:"for"`                     // min duration condition must hold (0 = immediate)
	Action     string        `yaml:"action" json:"action"`               // "kill" | "suspend" | "alert"
	Cooldown   time.Duration `yaml:"cooldown" json:"cooldown"`           // min gap between repeat actions on same pid (default 1m)
}

type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	OpenBrowser bool   `yaml:"open_browser"`
}

type MonitoringConfig struct {
	Interval            time.Duration `yaml:"interval"`
	ProcessTreeInterval time.Duration `yaml:"process_tree_interval"`
	PortScanInterval    time.Duration `yaml:"port_scan_interval"`
	GPUInterval         time.Duration `yaml:"gpu_interval"`
	HistoryDuration     time.Duration `yaml:"history_duration"`
	MaxProcesses        int           `yaml:"max_processes"`
}

type ControllerConfig struct {
	ProtectedProcesses []string `yaml:"protected_processes"`
	ConfirmKillSystem  bool     `yaml:"confirm_kill_system"`
}

type AnomalyConfig struct {
	AnalysisInterval time.Duration        `yaml:"analysis_interval"`
	IgnoreProcesses  []string             `yaml:"ignore_processes"`
	MaxActiveAlerts  int                  `yaml:"max_active_alerts"`
	SpawnStorm       SpawnStormConfig     `yaml:"spawn_storm"`
	MemoryLeak       MemoryLeakConfig     `yaml:"memory_leak"`
	HungProcess      HungProcessConfig    `yaml:"hung_process"`
	Orphan           OrphanConfig         `yaml:"orphan"`
	RunawayCPU       RunawayCPUConfig     `yaml:"runaway_cpu"`
	PortConflict     PortConflictConfig   `yaml:"port_conflict"`
	NetworkAnomaly   NetworkAnomalyConfig `yaml:"network_anomaly"`
	NewProcess       NewProcessConfig     `yaml:"new_process"`
}

type SpawnStormConfig struct {
	Enabled              bool   `yaml:"enabled"`
	MaxChildrenPerMinute int    `yaml:"max_children_per_minute"`
	MaxTotalChildren     int    `yaml:"max_total_children"`
	Action               string `yaml:"action"`
}

type MemoryLeakConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Window          time.Duration `yaml:"window"`
	MinGrowthRate   string        `yaml:"min_growth_rate"`
	MinRSquared     float64       `yaml:"min_r_squared"`
	MemoryThreshold string        `yaml:"memory_threshold"`
	Action          string        `yaml:"action"`
}

type HungProcessConfig struct {
	Enabled               bool          `yaml:"enabled"`
	ZeroActivityThreshold time.Duration `yaml:"zero_activity_threshold"`
	CriticalHungThreshold time.Duration `yaml:"critical_hung_threshold"`
	IdleWhitelist         []string      `yaml:"idle_whitelist"`
	Action                string        `yaml:"action"`
}

type OrphanConfig struct {
	Enabled                 bool   `yaml:"enabled"`
	ResourceThresholdCPU    int    `yaml:"resource_threshold_cpu"`
	ResourceThresholdMemory string `yaml:"resource_threshold_memory"`
	Action                  string `yaml:"action"`
}

type RunawayCPUConfig struct {
	Enabled           bool          `yaml:"enabled"`
	CPUThreshold      int           `yaml:"cpu_threshold"`
	DurationThreshold time.Duration `yaml:"duration_threshold"`
	CriticalDuration  time.Duration `yaml:"critical_duration"`
	HighCPUWhitelist  []string      `yaml:"high_cpu_whitelist"`
	Action            string        `yaml:"action"`
}

type PortConflictConfig struct {
	Enabled            bool          `yaml:"enabled"`
	TimeWaitThreshold  time.Duration `yaml:"time_wait_threshold"`
	CloseWaitThreshold time.Duration `yaml:"close_wait_threshold"`
	Action             string        `yaml:"action"`
}

type NetworkAnomalyConfig struct {
	Enabled              bool   `yaml:"enabled"`
	ConnectionSigma      int    `yaml:"connection_sigma"`
	MaxSystemConnections int    `yaml:"max_system_connections"`
	Action               string `yaml:"action"`
}

type NewProcessConfig struct {
	Enabled         bool     `yaml:"enabled"`
	SuspiciousPaths []string `yaml:"suspicious_paths"`
	Action          string   `yaml:"action"`
}

type NotificationsConfig struct {
	TrayBalloon        bool          `yaml:"tray_balloon"`
	BalloonRateLimit   time.Duration `yaml:"balloon_rate_limit"`
	BalloonMinSeverity string        `yaml:"balloon_min_severity"`
}

type AIConfig struct {
	Enabled                bool              `yaml:"enabled"`
	Provider               string            `yaml:"provider"`
	APIKey                 string            `yaml:"api_key"`
	Model                  string            `yaml:"model"`
	Endpoint               string            `yaml:"endpoint"`
	ExtraHeaders           map[string]string `yaml:"extra_headers"`
	AutoAnalyzeOnCritical  bool              `yaml:"auto_analyze_on_critical"`
	MaxTokens              int               `yaml:"max_tokens"`
	MaxRequestsPerMinute   int               `yaml:"max_requests_per_minute"`
	Language               string            `yaml:"language"`
	IncludeProcessTree     bool              `yaml:"include_process_tree"`
	IncludePortMap         bool              `yaml:"include_port_map"`
	HistoryContextDuration time.Duration     `yaml:"history_context_duration"`
}

type UIConfig struct {
	Theme                string        `yaml:"theme"`
	DefaultSort          string        `yaml:"default_sort"`
	DefaultSortOrder     string        `yaml:"default_sort_order"`
	SparklinePoints      int           `yaml:"sparkline_points"`
	ProcessTablePageSize int           `yaml:"process_table_page_size"`
	RefreshRate          time.Duration `yaml:"refresh_rate"`
}

// DefaultConfig returns sensible defaults for all settings.
func DefaultConfig() *Config {
	return &Config{
		SchemaVersion: CurrentSchemaVersion,
		Server: ServerConfig{
			Host:        "127.0.0.1",
			Port:        19876,
			OpenBrowser: true,
		},
		Monitoring: MonitoringConfig{
			Interval:            1000 * time.Millisecond,
			ProcessTreeInterval: 2000 * time.Millisecond,
			PortScanInterval:    3000 * time.Millisecond,
			GPUInterval:         2000 * time.Millisecond,
			HistoryDuration:     10 * time.Minute,
			MaxProcesses:        2000,
		},
		Controller: ControllerConfig{
			ProtectedProcesses: []string{
				"csrss.exe", "wininit.exe", "lsass.exe", "smss.exe",
				"services.exe", "svchost.exe", "winlogon.exe",
			},
			ConfirmKillSystem: true,
		},
		Anomaly: AnomalyConfig{
			AnalysisInterval: 2 * time.Second,
			MaxActiveAlerts:  100,
			IgnoreProcesses: []string{
				"System", "Registry", "Idle", "Memory Compression",
				"smss.exe", "csrss.exe", "wininit.exe", "winlogon.exe",
				"services.exe", "lsass.exe", "svchost.exe", "fontdrvhost.exe",
				"dwm.exe", "sihost.exe", "taskhostw.exe", "RuntimeBroker.exe",
				"SearchHost.exe", "SearchIndexer.exe", "StartMenuExperienceHost.exe",
				"ShellExperienceHost.exe", "ctfmon.exe", "explorer.exe",
				"spoolsv.exe", "MsMpEng.exe", "SecurityHealthService.exe",
				"audiodg.exe", "conhost.exe", "dllhost.exe", "WmiPrvSE.exe",
			},
			// SpawnStorm: only fires on genuine fork bombs (50+ children/minute),
			// and the detector itself has a hard-coded shell/browser whitelist.
			SpawnStorm: SpawnStormConfig{
				Enabled:              true,
				MaxChildrenPerMinute: 50,
				MaxTotalChildren:     100,
				Action:               "alert",
			},
			// MemoryLeak: statistical regression — high R² threshold keeps noise low.
			MemoryLeak: MemoryLeakConfig{
				Enabled:         true,
				Window:          5 * time.Minute,
				MinGrowthRate:   "10MB/min",
				MinRSquared:     0.8,
				MemoryThreshold: "2GB",
				Action:          "alert",
			},
			// HungProcess: OFF by default — on a real workstation most daemons
			// sit idle for hours, and we have no window-message-pump data to
			// distinguish them from actually frozen UIs. Users can opt in.
			HungProcess: HungProcessConfig{
				Enabled:               false,
				ZeroActivityThreshold: 120 * time.Second,
				CriticalHungThreshold: 300 * time.Second,
				IdleWhitelist:         []string{"SearchIndexer.exe", "spoolsv.exe"},
				Action:                "alert",
			},
			// Orphan: OFF by default — most orphaned processes on Windows are
			// benign detach patterns (installers, updaters, tray helpers).
			Orphan: OrphanConfig{
				Enabled:                 false,
				ResourceThresholdCPU:    10,
				ResourceThresholdMemory: "1GB",
				Action:                  "alert",
			},
			RunawayCPU: RunawayCPUConfig{
				Enabled:           true,
				CPUThreshold:      90,
				DurationThreshold: 60 * time.Second,
				CriticalDuration:  180 * time.Second,
				Action:            "alert",
			},
			// PortConflict: OFF by default — a single TIME_WAIT/CLOSE_WAIT
			// socket held for a few minutes is normal TCP housekeeping, not
			// a conflict.
			PortConflict: PortConflictConfig{
				Enabled:            false,
				TimeWaitThreshold:  120 * time.Second,
				CloseWaitThreshold: 60 * time.Second,
				Action:             "alert",
			},
			// NetworkAnomaly: OFF by default — sigma-based burst detection is
			// too noisy without a longer training window.
			NetworkAnomaly: NetworkAnomalyConfig{
				Enabled:              false,
				ConnectionSigma:      3,
				MaxSystemConnections: 10000,
				Action:               "alert",
			},
			NewProcess: NewProcessConfig{
				Enabled:         false,
				SuspiciousPaths: []string{},
				Action:          "info",
			},
		},
		Notifications: NotificationsConfig{
			TrayBalloon:        true,
			BalloonRateLimit:   30 * time.Second,
			BalloonMinSeverity: "critical",
		},
		WellKnownPorts: map[uint16]string{
			22:    "SSH",
			80:    "HTTP Server",
			443:   "HTTPS Server",
			3000:  "Dev Server (React/Next.js)",
			3001:  "Dev Server Alt",
			4200:  "Angular Dev Server",
			5173:  "Vite Dev Server",
			5432:  "PostgreSQL",
			6379:  "Redis",
			8080:  "HTTP Alt / Spring Boot",
			8443:  "HTTPS Alt",
			9090:  "Prometheus",
			27017: "MongoDB",
		},
		AI: AIConfig{
			Enabled:                false,
			Provider:               "anthropic",
			APIKey:                 "",
			Model:                  "claude-sonnet-4-20250514",
			Endpoint:               "",
			ExtraHeaders:           map[string]string{},
			AutoAnalyzeOnCritical:  true,
			MaxTokens:              1024,
			MaxRequestsPerMinute:   5,
			Language:               "en",
			IncludeProcessTree:     true,
			IncludePortMap:         true,
			HistoryContextDuration: 5 * time.Minute,
		},
		UI: UIConfig{
			Theme:                "system",
			DefaultSort:          "cpu",
			DefaultSortOrder:     "desc",
			SparklinePoints:      60,
			ProcessTablePageSize: 100,
			RefreshRate:          1000 * time.Millisecond,
		},
		Rules: []Rule{},
	}
}

// Validate checks the configuration for obvious errors.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port out of range: %d", c.Server.Port)
	}
	if c.Monitoring.Interval < 100*time.Millisecond {
		return fmt.Errorf("monitoring.interval too small: %v", c.Monitoring.Interval)
	}
	if c.Monitoring.MaxProcesses < 100 {
		c.Monitoring.MaxProcesses = 100
	}
	if c.UI.SparklinePoints < 10 {
		c.UI.SparklinePoints = 10
	}
	return nil
}
