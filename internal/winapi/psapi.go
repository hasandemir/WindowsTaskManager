//go:build windows

package winapi

// #nosec G103 -- This file intentionally performs audited Win32 syscall interop.

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetProcessMemoryInfo retrieves a process's memory counters.
func GetProcessMemoryInfo(h windows.Handle) (*PROCESS_MEMORY_COUNTERS_EX, error) {
	var counters PROCESS_MEMORY_COUNTERS_EX
	counters.CB = uint32(unsafe.Sizeof(counters))
	r1, _, e := procGetProcessMemoryInfo.Call( // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(h),
		uintptr(unsafe.Pointer(&counters)), // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(counters.CB),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("GetProcessMemoryInfo: %w", e)
	}
	return &counters, nil
}
