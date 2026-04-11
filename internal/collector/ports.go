//go:build windows

package collector

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
	"github.com/ersinkoc/WindowsTaskManager/internal/winapi"
)

// PortCollector enumerates TCP/UDP endpoints and tracks how long each
// (proto, local, remote, state) tuple has existed for port-conflict
// anomaly detection.
type PortCollector struct {
	mu             sync.Mutex
	since          map[string]int64 // tuple key -> first-seen unix seconds
	wellKnownPorts map[uint16]string
}

func NewPortCollector(wellKnown map[uint16]string) *PortCollector {
	return &PortCollector{
		since:          make(map[string]int64),
		wellKnownPorts: wellKnown,
	}
}

// SetWellKnown replaces the well-known port label map (used after config reload).
func (pc *PortCollector) SetWellKnown(m map[uint16]string) {
	pc.mu.Lock()
	pc.wellKnownPorts = m
	pc.mu.Unlock()
}

// Collect returns the current port table. resolveName looks up a process
// name from PID — pass a closure backed by the latest process snapshot.
func (pc *PortCollector) Collect(resolveName func(pid uint32) string) []metrics.PortBinding {
	now := time.Now().Unix()
	results := make([]metrics.PortBinding, 0, 256)
	seen := make(map[string]struct{}, 256)

	if rows, err := winapi.GetTcp4Table(); err == nil {
		for _, r := range rows {
			pb := metrics.PortBinding{
				Protocol:   "tcp",
				LocalAddr:  ipv4String(r.LocalAddr),
				LocalPort:  portFromMib(r.LocalPort),
				RemoteAddr: ipv4String(r.RemoteAddr),
				RemotePort: portFromMib(r.RemotePort),
				State:      tcpStateName(r.State),
				StateCode:  r.State,
				PID:        r.OwningPid,
			}
			pc.finalize(&pb, resolveName, now, seen)
			results = append(results, pb)
		}
	}
	if rows, err := winapi.GetTcp6Table(); err == nil {
		for _, r := range rows {
			pb := metrics.PortBinding{
				Protocol:   "tcp6",
				LocalAddr:  ipv6String(r.LocalAddr),
				LocalPort:  portFromMib(r.LocalPort),
				RemoteAddr: ipv6String(r.RemoteAddr),
				RemotePort: portFromMib(r.RemotePort),
				State:      tcpStateName(r.State),
				StateCode:  r.State,
				PID:        r.OwningPid,
			}
			pc.finalize(&pb, resolveName, now, seen)
			results = append(results, pb)
		}
	}
	if rows, err := winapi.GetUdp4Table(); err == nil {
		for _, r := range rows {
			pb := metrics.PortBinding{
				Protocol:  "udp",
				LocalAddr: ipv4String(r.LocalAddr),
				LocalPort: portFromMib(r.LocalPort),
				State:     "listen",
				PID:       r.OwningPid,
			}
			pc.finalize(&pb, resolveName, now, seen)
			results = append(results, pb)
		}
	}
	if rows, err := winapi.GetUdp6Table(); err == nil {
		for _, r := range rows {
			pb := metrics.PortBinding{
				Protocol:  "udp6",
				LocalAddr: ipv6String(r.LocalAddr),
				LocalPort: portFromMib(r.LocalPort),
				State:     "listen",
				PID:       r.OwningPid,
			}
			pc.finalize(&pb, resolveName, now, seen)
			results = append(results, pb)
		}
	}

	pc.mu.Lock()
	for k := range pc.since {
		if _, ok := seen[k]; !ok {
			delete(pc.since, k)
		}
	}
	pc.mu.Unlock()

	return results
}

func (pc *PortCollector) finalize(pb *metrics.PortBinding, resolveName func(pid uint32) string, now int64, seen map[string]struct{}) {
	if resolveName != nil {
		pb.Process = resolveName(pb.PID)
	}
	pc.mu.Lock()
	if label, ok := pc.wellKnownPorts[pb.LocalPort]; ok {
		pb.Label = label
	}
	key := fmt.Sprintf("%s|%s:%d|%s:%d|%s", pb.Protocol, pb.LocalAddr, pb.LocalPort, pb.RemoteAddr, pb.RemotePort, pb.State)
	if first, ok := pc.since[key]; ok {
		pb.Since = first
	} else {
		pc.since[key] = now
		pb.Since = now
	}
	pc.mu.Unlock()
	seen[key] = struct{}{}
}

// portFromMib swaps the network-byte-order port stored in the low 16 bits
// of a MIB row's Port field.
func portFromMib(p uint32) uint16 {
	// Win32 stores ports in network byte order in the high byte of low word.
	b := []byte{byte(p), byte(p >> 8)}
	return binary.BigEndian.Uint16(b)
}

func ipv4String(addr uint32) string {
	a := byte(addr)
	b := byte(addr >> 8)
	c := byte(addr >> 16)
	d := byte(addr >> 24)
	return net.IPv4(a, b, c, d).String()
}

func ipv6String(addr [16]byte) string {
	return net.IP(addr[:]).String()
}

func tcpStateName(s uint32) string {
	switch s {
	case winapi.MIB_TCP_STATE_CLOSED:
		return "closed"
	case winapi.MIB_TCP_STATE_LISTEN:
		return "listen"
	case winapi.MIB_TCP_STATE_SYN_SENT:
		return "syn-sent"
	case winapi.MIB_TCP_STATE_SYN_RCVD:
		return "syn-rcvd"
	case winapi.MIB_TCP_STATE_ESTAB:
		return "established"
	case winapi.MIB_TCP_STATE_FIN_WAIT1:
		return "fin-wait-1"
	case winapi.MIB_TCP_STATE_FIN_WAIT2:
		return "fin-wait-2"
	case winapi.MIB_TCP_STATE_CLOSE_WAIT:
		return "close-wait"
	case winapi.MIB_TCP_STATE_CLOSING:
		return "closing"
	case winapi.MIB_TCP_STATE_LAST_ACK:
		return "last-ack"
	case winapi.MIB_TCP_STATE_TIME_WAIT:
		return "time-wait"
	case winapi.MIB_TCP_STATE_DELETE_TCB:
		return "delete-tcb"
	}
	return "unknown"
}
