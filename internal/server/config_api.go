package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

type configUpdateDTO struct {
	Server        *configServerUpdateDTO        `json:"server,omitempty"`
	Monitoring    *configMonitoringUpdateDTO    `json:"monitoring,omitempty"`
	Controller    *configControllerUpdateDTO    `json:"controller,omitempty"`
	Notifications *configNotificationsUpdateDTO `json:"notifications,omitempty"`
	UI            *configUIUpdateDTO            `json:"ui,omitempty"`
}

type configServerUpdateDTO struct {
	OpenBrowser *bool `json:"open_browser,omitempty"`
}

type configMonitoringUpdateDTO struct {
	IntervalMS            *int `json:"interval_ms,omitempty"`
	ProcessTreeIntervalMS *int `json:"process_tree_interval_ms,omitempty"`
	PortScanIntervalMS    *int `json:"port_scan_interval_ms,omitempty"`
	GPUIntervalMS         *int `json:"gpu_interval_ms,omitempty"`
	HistoryDurationSec    *int `json:"history_duration_sec,omitempty"`
	MaxProcesses          *int `json:"max_processes,omitempty"`
}

type configControllerUpdateDTO struct {
	ConfirmKillSystem *bool `json:"confirm_kill_system,omitempty"`
}

type configNotificationsUpdateDTO struct {
	TrayBalloon         *bool   `json:"tray_balloon,omitempty"`
	BalloonRateLimitSec *int    `json:"balloon_rate_limit_sec,omitempty"`
	BalloonMinSeverity  *string `json:"balloon_min_severity,omitempty"`
}

type configUIUpdateDTO struct {
	Theme                *string `json:"theme,omitempty"`
	DefaultSort          *string `json:"default_sort,omitempty"`
	DefaultSortOrder     *string `json:"default_sort_order,omitempty"`
	SparklinePoints      *int    `json:"sparkline_points,omitempty"`
	ProcessTablePageSize *int    `json:"process_table_page_size,omitempty"`
	RefreshRateMS        *int    `json:"refresh_rate_ms,omitempty"`
}

func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if s.cfgPath == "" {
		writeError(w, http.StatusServiceUnavailable, "no_config", "config file path not set")
		return
	}

	var body configUpdateDTO
	if !readJSON(w, r, &body) {
		return
	}

	if err := s.mutateConfig(func(c *config.Config) error {
		applyConfigUpdate(c, &body)
		return nil
	}); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_config", err.Error())
		return
	}

	s.mu.RLock()
	current := cloneConfig(s.cfg)
	s.mu.RUnlock()
	current.AI.APIKey = maskSecret(current.AI.APIKey)
	current.Telegram.BotToken = maskSecret(current.Telegram.BotToken)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": &current,
	})
}

func applyConfigUpdate(next *config.Config, body *configUpdateDTO) {
	if body == nil || next == nil {
		return
	}
	if body.Server != nil && body.Server.OpenBrowser != nil {
		next.Server.OpenBrowser = *body.Server.OpenBrowser
	}
	if body.Monitoring != nil {
		if body.Monitoring.IntervalMS != nil {
			next.Monitoring.Interval = time.Duration(*body.Monitoring.IntervalMS) * time.Millisecond
		}
		if body.Monitoring.ProcessTreeIntervalMS != nil {
			next.Monitoring.ProcessTreeInterval = time.Duration(*body.Monitoring.ProcessTreeIntervalMS) * time.Millisecond
		}
		if body.Monitoring.PortScanIntervalMS != nil {
			next.Monitoring.PortScanInterval = time.Duration(*body.Monitoring.PortScanIntervalMS) * time.Millisecond
		}
		if body.Monitoring.GPUIntervalMS != nil {
			next.Monitoring.GPUInterval = time.Duration(*body.Monitoring.GPUIntervalMS) * time.Millisecond
		}
		if body.Monitoring.HistoryDurationSec != nil {
			next.Monitoring.HistoryDuration = time.Duration(*body.Monitoring.HistoryDurationSec) * time.Second
		}
		if body.Monitoring.MaxProcesses != nil {
			next.Monitoring.MaxProcesses = *body.Monitoring.MaxProcesses
		}
	}
	if body.Controller != nil && body.Controller.ConfirmKillSystem != nil {
		next.Controller.ConfirmKillSystem = *body.Controller.ConfirmKillSystem
	}
	if body.Notifications != nil {
		if body.Notifications.TrayBalloon != nil {
			next.Notifications.TrayBalloon = *body.Notifications.TrayBalloon
		}
		if body.Notifications.BalloonRateLimitSec != nil {
			next.Notifications.BalloonRateLimit = time.Duration(*body.Notifications.BalloonRateLimitSec) * time.Second
		}
		if body.Notifications.BalloonMinSeverity != nil {
			next.Notifications.BalloonMinSeverity = strings.TrimSpace(*body.Notifications.BalloonMinSeverity)
		}
	}
	if body.UI != nil {
		if body.UI.Theme != nil {
			next.UI.Theme = strings.TrimSpace(*body.UI.Theme)
		}
		if body.UI.DefaultSort != nil {
			next.UI.DefaultSort = strings.TrimSpace(*body.UI.DefaultSort)
		}
		if body.UI.DefaultSortOrder != nil {
			next.UI.DefaultSortOrder = strings.TrimSpace(*body.UI.DefaultSortOrder)
		}
		if body.UI.SparklinePoints != nil {
			next.UI.SparklinePoints = *body.UI.SparklinePoints
		}
		if body.UI.ProcessTablePageSize != nil {
			next.UI.ProcessTablePageSize = *body.UI.ProcessTablePageSize
		}
		if body.UI.RefreshRateMS != nil {
			next.UI.RefreshRate = time.Duration(*body.UI.RefreshRateMS) * time.Millisecond
		}
	}
}
