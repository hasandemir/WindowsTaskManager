//go:build windows

package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/ai"
	"github.com/ersinkoc/WindowsTaskManager/internal/anomaly"
	"github.com/ersinkoc/WindowsTaskManager/internal/config"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/storage"
)

type fakeController struct {
	killed    []uint32
	suspended []uint32
	resumed   []uint32
}

type fakeAdvisor struct {
	result *ai.AnalyzeResult
}

func (f *fakeAdvisor) Enabled() bool { return true }

func (f *fakeAdvisor) Analyze(ctx context.Context, userQuestion string) (*ai.AnalyzeResult, error) {
	return f.result, nil
}

func (f *fakeAdvisor) Chat(ctx context.Context, userMessage string) (*ai.AnalyzeResult, error) {
	return f.result, nil
}

func (f *fakeController) Kill(pid uint32, confirm bool) error {
	f.killed = append(f.killed, pid)
	return nil
}

func (f *fakeController) Suspend(pid uint32, confirm bool) error {
	f.suspended = append(f.suspended, pid)
	return nil
}

func (f *fakeController) Resume(pid uint32) error {
	f.resumed = append(f.resumed, pid)
	return nil
}

func TestParseCommand(t *testing.T) {
	cmd, args := parseCommand("/topcpu@wtm_bot 123")
	if cmd != "topcpu" {
		t.Fatalf("cmd=%q want topcpu", cmd)
	}
	if len(args) != 1 || args[0] != "123" {
		t.Fatalf("args=%v want [123]", args)
	}
}

func TestIsAllowedChat(t *testing.T) {
	if !isAllowedChat([]int64{1, 2, 3}, 2) {
		t.Fatal("expected chat 2 to be allowed")
	}
	if isAllowedChat([]int64{1, 2, 3}, 9) {
		t.Fatal("expected chat 9 to be rejected")
	}
}

func TestPIDActionRequiresConfirm(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Telegram.RequireConfirm = true
	cfg.Telegram.ConfirmTTL = 45 * time.Second

	ctrl := &fakeController{}
	store := storage.NewStore(60, 10)
	store.SetLatest(&metrics.SystemSnapshot{
		Timestamp: time.Now(),
		Processes: []metrics.ProcessInfo{{PID: 4242, Name: "node.exe"}},
	})

	bot := New(cfg, store, anomaly.NewAlertStore(32), ctrl, nil, nil, nil)
	reply := bot.pidAction(cfg, 99, []string{"4242"}, func(pid uint32) error {
		return ctrl.Kill(pid, true)
	}, "kill", "killed", true)

	if !strings.Contains(reply, "/confirm ") {
		t.Fatalf("reply=%q missing confirm hint", reply)
	}
	if len(ctrl.killed) != 0 {
		t.Fatalf("kill ran before confirm: %v", ctrl.killed)
	}

	code := extractCode(reply)
	done := bot.confirmAction([]string{code}, 99)
	if !strings.Contains(done, "Killed node.exe (PID 4242)") {
		t.Fatalf("confirm reply=%q", done)
	}
	if len(ctrl.killed) != 1 || ctrl.killed[0] != 4242 {
		t.Fatalf("kill calls=%v want [4242]", ctrl.killed)
	}

	again := bot.confirmAction([]string{code}, 99)
	if again != "Confirmation code not found or expired." {
		t.Fatalf("unexpected second confirm reply=%q", again)
	}
}

func TestCancelAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Telegram.RequireConfirm = true
	cfg.Telegram.ConfirmTTL = 30 * time.Second

	ctrl := &fakeController{}
	store := storage.NewStore(60, 10)
	store.SetLatest(&metrics.SystemSnapshot{
		Timestamp: time.Now(),
		Processes: []metrics.ProcessInfo{{PID: 9001, Name: "chrome.exe"}},
	})

	bot := New(cfg, store, anomaly.NewAlertStore(32), ctrl, nil, nil, nil)
	reply := bot.pidAction(cfg, 77, []string{"9001"}, func(pid uint32) error {
		return ctrl.Suspend(pid, true)
	}, "suspend", "suspended", true)
	code := extractCode(reply)

	cancelled := bot.cancelAction([]string{code}, 77)
	if !strings.Contains(cancelled, "Cancelled suspend chrome.exe (PID 9001)") {
		t.Fatalf("cancel reply=%q", cancelled)
	}
	if len(ctrl.suspended) != 0 {
		t.Fatalf("suspend ran after cancel: %v", ctrl.suspended)
	}
}

func TestShouldNotifyTelegramAlertHighValueOnly(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Telegram.Enabled = true
	cfg.Telegram.BotToken = "secret"
	cfg.Telegram.NotifyOnCritical = true
	cfg.Telegram.NotificationMode = "high_value"
	cfg.Telegram.NotificationTypes = []string{"runaway_cpu", "rule:*"}

	if shouldNotifyTelegramAlert(cfg, anomaly.Alert{Type: "hung_process", Severity: anomaly.SeverityCritical}) {
		t.Fatal("hung_process should be suppressed in high_value mode")
	}
	if !shouldNotifyTelegramAlert(cfg, anomaly.Alert{Type: "runaway_cpu", Severity: anomaly.SeverityCritical}) {
		t.Fatal("runaway_cpu should pass in high_value mode")
	}
	if shouldNotifyTelegramAlert(cfg, anomaly.Alert{Type: "memory_leak", Severity: anomaly.SeverityCritical}) {
		t.Fatal("memory_leak should be suppressed when removed from allowlist")
	}
	if !shouldNotifyTelegramAlert(cfg, anomaly.Alert{Type: "rule:KillChrome", Severity: anomaly.SeverityCritical, Action: "kill"}) {
		t.Fatal("rule:* should allow critical rule actions")
	}
}

func TestAIChatQueuesConfirmableAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Telegram.RequireConfirm = true
	cfg.Telegram.ConfirmTTL = 45 * time.Second

	executed := []string{}
	bot := New(
		cfg,
		storage.NewStore(60, 10),
		anomaly.NewAlertStore(32),
		&fakeController{},
		&fakeAdvisor{
			result: &ai.AnalyzeResult{
				Answer: "Protect claude.exe and add a rule.",
				Actions: []ai.Suggestion{
					{Type: "protect", Name: "claude.exe"},
				},
			},
		},
		func(suggestion ai.Suggestion) error {
			executed = append(executed, suggestion.Type+":"+suggestion.Name)
			return nil
		},
		nil,
	)

	reply := bot.aiChatText(context.Background(), cfg, 55, "make this safer")
	if !strings.Contains(reply, "/confirm ") {
		t.Fatalf("reply=%q missing confirm hint", reply)
	}
	code := extractCode(reply)
	done := bot.confirmAction([]string{code}, 55)
	if !strings.Contains(done, "AI action executed") {
		t.Fatalf("confirm reply=%q", done)
	}
	if len(executed) != 1 || executed[0] != "protect:claude.exe" {
		t.Fatalf("executed=%v", executed)
	}
}

func TestPIDActionFailsClosedWhenConfirmCodeGenerationFails(t *testing.T) {
	old := newConfirmCodeFunc
	newConfirmCodeFunc = func() (string, error) {
		return "", errors.New("entropy unavailable")
	}
	defer func() { newConfirmCodeFunc = old }()

	cfg := config.DefaultConfig()
	cfg.Telegram.RequireConfirm = true

	ctrl := &fakeController{}
	store := storage.NewStore(60, 10)
	store.SetLatest(&metrics.SystemSnapshot{
		Timestamp: time.Now(),
		Processes: []metrics.ProcessInfo{{PID: 5150, Name: "tool.exe"}},
	})

	bot := New(cfg, store, anomaly.NewAlertStore(32), ctrl, nil, nil, nil)
	reply := bot.pidAction(cfg, 12, []string{"5150"}, func(pid uint32) error {
		return ctrl.Kill(pid, true)
	}, "kill", "killed", true)

	if reply != "Failed to create confirmation code; action was not executed." {
		t.Fatalf("reply=%q", reply)
	}
	if len(ctrl.killed) != 0 {
		t.Fatalf("kill should not run on confirmation setup failure: %v", ctrl.killed)
	}
}

func extractCode(reply string) string {
	for _, field := range strings.Fields(reply) {
		if len(field) == 8 && !strings.Contains(field, "/") && strings.ToUpper(field) == field {
			return strings.Trim(field, ".,")
		}
	}
	return ""
}
