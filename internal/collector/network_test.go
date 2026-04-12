//go:build windows

package collector

import (
	"testing"

	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

func TestShouldSkipInterfaceSkipsKnownNoise(t *testing.T) {
	row := winapi.IfRow2{
		Alias:       "Npcap Loopback Adapter",
		Description: "Npcap Loopback Adapter",
		Type:        6,
		OperStatus:  1,
	}
	if !shouldSkipInterface(row, map[string]struct{}{}) {
		t.Fatal("expected known noisy adapter to be skipped")
	}
}

func TestShouldSkipInterfaceDedupesByNormalizedName(t *testing.T) {
	seen := map[string]struct{}{}
	first := winapi.IfRow2{Alias: "Ethernet 2", Description: "Ethernet 2", Type: 6, OperStatus: 1, SpeedRx: 1}
	second := winapi.IfRow2{Alias: " ethernet-2 ", Description: "Ethernet 2", Type: 6, OperStatus: 1, SpeedRx: 1}
	if shouldSkipInterface(first, seen) {
		t.Fatal("first interface should not be skipped")
	}
	if !shouldSkipInterface(second, seen) {
		t.Fatal("duplicate interface name should be skipped")
	}
}

func TestShouldSkipInterfaceSkipsFilterDriverSuffixes(t *testing.T) {
	row := winapi.IfRow2{
		Alias:       "Ethernet-WFP Native MAC Layer LightWeight Filter-0000",
		Description: "Ethernet",
		Type:        6,
		OperStatus:  1,
		SpeedRx:     1,
	}
	if !shouldSkipInterface(row, map[string]struct{}{}) {
		t.Fatal("expected filter-driver duplicate to be skipped")
	}
}

func TestDisplayInterfaceNameStripsDecorators(t *testing.T) {
	got := displayInterfaceName("Wi-Fi 2-QoS Packet Scheduler-0000", "")
	if got != "Wi-Fi 2" {
		t.Fatalf("displayInterfaceName=%q want Wi-Fi 2", got)
	}
}
