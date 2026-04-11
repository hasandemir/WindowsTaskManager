package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/config"
)

func TestTelegramConfigSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "wtm.yaml")
	cfg := config.DefaultConfig()
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	s, _ := newTestServer(t, cfgPath, cfg)

	body := telegramConfigDTO{
		Enabled:          true,
		BotToken:         "123456:telegram-secret",
		AllowedChatIDs:   []int64{111, 222},
		APIBaseURL:       "https://api.telegram.org",
		PollTimeoutSec:   30,
		NotifyOnCritical: true,
		RequireConfirm:   true,
		ConfirmTTLSec:    75,
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/telegram/config", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleTelegramConfigUpdate(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Telegram.Enabled || reloaded.Telegram.BotToken != "123456:telegram-secret" {
		t.Fatalf("telegram config not saved: %+v", reloaded.Telegram)
	}
	if len(reloaded.Telegram.AllowedChatIDs) != 2 || reloaded.Telegram.AllowedChatIDs[1] != 222 {
		t.Fatalf("allowed_chat_ids=%v", reloaded.Telegram.AllowedChatIDs)
	}
	if !reloaded.Telegram.RequireConfirm || reloaded.Telegram.ConfirmTTL != 75*time.Second {
		t.Fatalf("confirm settings not saved: %+v", reloaded.Telegram)
	}

	getReq := httptest.NewRequest("GET", "/api/v1/telegram/config", nil)
	getRR := httptest.NewRecorder()
	s.handleTelegramConfigGet(getRR, getReq)
	var got telegramConfigDTO
	if err := json.Unmarshal(getRR.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.BotToken == "123456:telegram-secret" {
		t.Fatal("expected masked token in GET response")
	}
	if len(got.AllowedChatIDs) != 2 {
		t.Fatalf("got allowed ids=%v", got.AllowedChatIDs)
	}
	if !got.RequireConfirm || got.ConfirmTTLSec != 75 {
		t.Fatalf("got confirm settings=%+v", got)
	}
}
