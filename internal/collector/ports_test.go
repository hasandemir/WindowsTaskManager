//go:build windows

package collector

import (
	"testing"

	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

func TestPortFromMib(t *testing.T) {
	if got := portFromMib(0x5000); got != 80 {
		t.Fatalf("portFromMib(0x5000)=%d want 80", got)
	}
	if got := portFromMib(0x901F); got != 8080 {
		t.Fatalf("portFromMib(0x901F)=%d want 8080", got)
	}
}

func TestIPv4String(t *testing.T) {
	if got := ipv4String(0x0100007F); got != "127.0.0.1" {
		t.Fatalf("ipv4String=%q want 127.0.0.1", got)
	}
}

func TestTCPStateName(t *testing.T) {
	if got := tcpStateName(winapi.MIB_TCP_STATE_ESTAB); got != "established" {
		t.Fatalf("tcpStateName(estab)=%q", got)
	}
	if got := tcpStateName(9999); got != "unknown" {
		t.Fatalf("tcpStateName(unknown)=%q", got)
	}
}
