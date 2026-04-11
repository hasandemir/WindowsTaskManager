package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

type telegramConfigDTO struct {
	Enabled          bool    `json:"enabled"`
	BotToken         string  `json:"bot_token"`
	AllowedChatIDs   []int64 `json:"allowed_chat_ids"`
	APIBaseURL       string  `json:"api_base_url"`
	PollTimeoutSec   int     `json:"poll_timeout_sec"`
	NotifyOnCritical bool    `json:"notify_on_critical"`
	RequireConfirm   bool    `json:"require_confirm"`
	ConfirmTTLSec    int     `json:"confirm_ttl_sec"`
}

func (s *Server) handleTelegramConfigGet(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, telegramDTOFromConfig(&cfg.Telegram))
}

func (s *Server) handleTelegramConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if s.cfgPath == "" {
		writeError(w, http.StatusServiceUnavailable, "no_config", "config file path not set")
		return
	}

	var body telegramConfigDTO
	if !readJSON(w, r, &body) {
		return
	}

	s.mu.RLock()
	current := *s.cfg
	s.mu.RUnlock()

	next := current
	next.Telegram = current.Telegram
	next.Telegram.Enabled = body.Enabled
	if body.BotToken != "" {
		next.Telegram.BotToken = strings.TrimSpace(body.BotToken)
	}
	next.Telegram.AllowedChatIDs = append([]int64(nil), body.AllowedChatIDs...)
	next.Telegram.APIBaseURL = strings.TrimSpace(body.APIBaseURL)
	next.Telegram.PollTimeout = time.Duration(body.PollTimeoutSec) * time.Second
	next.Telegram.NotifyOnCritical = body.NotifyOnCritical
	next.Telegram.RequireConfirm = body.RequireConfirm
	next.Telegram.ConfirmTTL = time.Duration(body.ConfirmTTLSec) * time.Second

	if err := next.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_config", err.Error())
		return
	}
	if err := config.Save(s.cfgPath, &next); err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	s.SetConfig(&next)
	if s.onCfgApply != nil {
		s.onCfgApply(&next)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": telegramDTOFromConfig(&next.Telegram),
	})
}

func telegramDTOFromConfig(tg *config.TelegramConfig) telegramConfigDTO {
	return telegramConfigDTO{
		Enabled:          tg.Enabled,
		BotToken:         maskSecret(tg.BotToken),
		AllowedChatIDs:   append([]int64(nil), tg.AllowedChatIDs...),
		APIBaseURL:       tg.APIBaseURL,
		PollTimeoutSec:   int(tg.PollTimeout / time.Second),
		NotifyOnCritical: tg.NotifyOnCritical,
		RequireConfirm:   tg.RequireConfirm,
		ConfirmTTLSec:    int(tg.ConfirmTTL / time.Second),
	}
}
