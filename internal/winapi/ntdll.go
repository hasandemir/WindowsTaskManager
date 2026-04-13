//go:build windows

package winapi

import (
	"fmt"
	"math"
	"unsafe"
)

const SystemProcessorPerformanceInformation = 8

// QueryProcessorPerformance returns per-logical-processor performance info.
func QueryProcessorPerformance(numCPU int) ([]SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION, error) {
	if numCPU <= 0 {
		return nil, fmt.Errorf("invalid CPU count")
	}
	infos := make([]SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION, numCPU)
	entrySize := uint64(unsafe.Sizeof(infos[0]))
	totalSize := entrySize * uint64(numCPU)
	if totalSize > math.MaxUint32 {
		return nil, fmt.Errorf("processor performance table too large: %d", totalSize)
	}
	size := uint32(totalSize)
	var returned uint32
	r1, _, _ := procNtQuerySystemInformation.Call(
		uintptr(SystemProcessorPerformanceInformation),
		uintptr(unsafe.Pointer(&infos[0])), // #nosec G103 -- Audited Win32 unsafe interop.
		uintptr(size),
		uintptr(unsafe.Pointer(&returned)), // #nosec G103 -- Audited Win32 unsafe interop.
	)
	if r1 != 0 {
		return nil, fmt.Errorf("NtQuerySystemInformation: status 0x%X", r1)
	}
	return infos, nil
}
