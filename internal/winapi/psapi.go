//go:build windows

package winapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetProcessMemoryInfo retrieves a process's memory counters.
func GetProcessMemoryInfo(h windows.Handle) (*PROCESS_MEMORY_COUNTERS_EX, error) {
	var counters PROCESS_MEMORY_COUNTERS_EX
	counters.CB = uint32(unsafe.Sizeof(counters))
	r1, _, e := procGetProcessMemoryInfo.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.CB),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("GetProcessMemoryInfo: %w", e)
	}
	return &counters, nil
}
