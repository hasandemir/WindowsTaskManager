//go:build windows

package winapi

import (
	"fmt"
	"unsafe"
)

const SystemProcessorPerformanceInformation = 8

// QueryProcessorPerformance returns per-logical-processor performance info.
func QueryProcessorPerformance(numCPU int) ([]SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION, error) {
	if numCPU <= 0 {
		return nil, fmt.Errorf("invalid CPU count")
	}
	infos := make([]SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION, numCPU)
	size := uint32(unsafe.Sizeof(infos[0])) * uint32(numCPU)
	var returned uint32
	r1, _, _ := procNtQuerySystemInformation.Call(
		uintptr(SystemProcessorPerformanceInformation),
		uintptr(unsafe.Pointer(&infos[0])),
		uintptr(size),
		uintptr(unsafe.Pointer(&returned)),
	)
	if r1 != 0 {
		return nil, fmt.Errorf("NtQuerySystemInformation: status 0x%X", r1)
	}
	return infos, nil
}
