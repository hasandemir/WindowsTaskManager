//go:build windows

package winapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetSystemTimes retrieves system idle/kernel/user FILETIMEs.
func GetSystemTimes() (idle, kernel, user FILETIME, err error) {
	r1, _, e := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r1 == 0 {
		return idle, kernel, user, fmt.Errorf("GetSystemTimes: %w", e)
	}
	return idle, kernel, user, nil
}

// GlobalMemoryStatusEx retrieves system memory information.
func GlobalMemoryStatusEx() (*MEMORYSTATUSEX, error) {
	var ms MEMORYSTATUSEX
	ms.Length = uint32(unsafe.Sizeof(ms))
	r1, _, e := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return nil, fmt.Errorf("GlobalMemoryStatusEx: %w", e)
	}
	return &ms, nil
}

// CreateToolhelp32Snapshot creates a snapshot of system processes/threads/modules.
func CreateToolhelp32Snapshot(flags, pid uint32) (windows.Handle, error) {
	r1, _, e := procCreateToolhelp32Snapshot.Call(uintptr(flags), uintptr(pid))
	if r1 == 0 || r1 == ^uintptr(0) {
		return 0, fmt.Errorf("CreateToolhelp32Snapshot: %w", e)
	}
	return windows.Handle(r1), nil
}

// Process32First retrieves the first process entry from a snapshot.
func Process32First(snap windows.Handle, entry *PROCESSENTRY32W) error {
	entry.Size = uint32(unsafe.Sizeof(*entry))
	r1, _, e := procProcess32FirstW.Call(uintptr(snap), uintptr(unsafe.Pointer(entry)))
	if r1 == 0 {
		return e
	}
	return nil
}

// Process32Next retrieves the next process entry.
func Process32Next(snap windows.Handle, entry *PROCESSENTRY32W) error {
	entry.Size = uint32(unsafe.Sizeof(*entry))
	r1, _, e := procProcess32NextW.Call(uintptr(snap), uintptr(unsafe.Pointer(entry)))
	if r1 == 0 {
		return e
	}
	return nil
}

// Thread32First retrieves the first thread from a snapshot.
func Thread32First(snap windows.Handle, te *THREADENTRY32) error {
	te.Size = uint32(unsafe.Sizeof(*te))
	r1, _, e := procThread32First.Call(uintptr(snap), uintptr(unsafe.Pointer(te)))
	if r1 == 0 {
		return e
	}
	return nil
}

// Thread32Next retrieves the next thread.
func Thread32Next(snap windows.Handle, te *THREADENTRY32) error {
	te.Size = uint32(unsafe.Sizeof(*te))
	r1, _, e := procThread32Next.Call(uintptr(snap), uintptr(unsafe.Pointer(te)))
	if r1 == 0 {
		return e
	}
	return nil
}

// OpenProcessHandle opens a process and returns a handle.
func OpenProcessHandle(access uint32, pid uint32) (windows.Handle, error) {
	r1, _, e := procOpenProcess.Call(uintptr(access), 0, uintptr(pid))
	if r1 == 0 {
		return 0, fmt.Errorf("OpenProcess(%d): %w", pid, e)
	}
	return windows.Handle(r1), nil
}

// OpenThreadHandle opens a thread and returns a handle.
func OpenThreadHandle(access uint32, threadID uint32) (windows.Handle, error) {
	r1, _, e := procOpenThread.Call(uintptr(access), 0, uintptr(threadID))
	if r1 == 0 {
		return 0, fmt.Errorf("OpenThread(%d): %w", threadID, e)
	}
	return windows.Handle(r1), nil
}

// CloseHandleSafe closes a Windows handle ignoring errors.
func CloseHandleSafe(h windows.Handle) {
	if h != 0 {
		procCloseHandle.Call(uintptr(h))
	}
}

// TerminateProcessHandle terminates a process via handle.
func TerminateProcessHandle(h windows.Handle, exitCode uint32) error {
	r1, _, e := procTerminateProcess.Call(uintptr(h), uintptr(exitCode))
	if r1 == 0 {
		return fmt.Errorf("TerminateProcess: %w", e)
	}
	return nil
}

// GetProcessTimes retrieves a process's creation/exit/kernel/user times.
func GetProcessTimes(h windows.Handle) (creation, exit, kernel, user FILETIME, err error) {
	r1, _, e := procGetProcessTimes.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r1 == 0 {
		err = fmt.Errorf("GetProcessTimes: %w", e)
	}
	return
}

// GetProcessIoCounters retrieves a process's I/O counters.
func GetProcessIoCounters(h windows.Handle) (*IO_COUNTERS, error) {
	var ic IO_COUNTERS
	r1, _, e := procGetProcessIoCounters.Call(uintptr(h), uintptr(unsafe.Pointer(&ic)))
	if r1 == 0 {
		return nil, fmt.Errorf("GetProcessIoCounters: %w", e)
	}
	return &ic, nil
}

// QueryFullProcessImageName retrieves a process's full executable path.
func QueryFullProcessImageName(h windows.Handle) (string, error) {
	var buf [1024]uint16
	size := uint32(len(buf))
	r1, _, e := procQueryFullProcessImageNameW.Call(
		uintptr(h),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r1 == 0 {
		return "", fmt.Errorf("QueryFullProcessImageNameW: %w", e)
	}
	return windows.UTF16ToString(buf[:size]), nil
}

// IsProcessCritical reports whether a process is marked critical.
func IsProcessCritical(h windows.Handle) (bool, error) {
	var critical int32
	r1, _, e := procIsProcessCritical.Call(uintptr(h), uintptr(unsafe.Pointer(&critical)))
	if r1 == 0 {
		return false, fmt.Errorf("IsProcessCritical: %w", e)
	}
	return critical != 0, nil
}

// SetPriorityClass sets a process's priority class.
func SetPriorityClass(h windows.Handle, class uint32) error {
	r1, _, e := procSetPriorityClass.Call(uintptr(h), uintptr(class))
	if r1 == 0 {
		return fmt.Errorf("SetPriorityClass: %w", e)
	}
	return nil
}

// GetPriorityClass returns a process's priority class.
func GetPriorityClass(h windows.Handle) (uint32, error) {
	r1, _, e := procGetPriorityClass.Call(uintptr(h))
	if r1 == 0 {
		return 0, fmt.Errorf("GetPriorityClass: %w", e)
	}
	return uint32(r1), nil
}

// SetProcessAffinityMask sets a process affinity mask.
func SetProcessAffinityMask(h windows.Handle, mask uintptr) error {
	r1, _, e := procSetProcessAffinityMask.Call(uintptr(h), mask)
	if r1 == 0 {
		return fmt.Errorf("SetProcessAffinityMask: %w", e)
	}
	return nil
}

// GetProcessAffinityMask retrieves process and system affinity masks.
func GetProcessAffinityMask(h windows.Handle) (process, system uintptr, err error) {
	r1, _, e := procGetProcessAffinityMask.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&process)),
		uintptr(unsafe.Pointer(&system)),
	)
	if r1 == 0 {
		err = fmt.Errorf("GetProcessAffinityMask: %w", e)
	}
	return
}

// CreateJobObject creates an unnamed Job Object.
func CreateJobObject() (windows.Handle, error) {
	r1, _, e := procCreateJobObjectW.Call(0, 0)
	if r1 == 0 {
		return 0, fmt.Errorf("CreateJobObjectW: %w", e)
	}
	return windows.Handle(r1), nil
}

// AssignProcessToJobObject assigns a process to a job.
func AssignProcessToJobObject(job, proc windows.Handle) error {
	r1, _, e := procAssignProcessToJobObject.Call(uintptr(job), uintptr(proc))
	if r1 == 0 {
		return fmt.Errorf("AssignProcessToJobObject: %w", e)
	}
	return nil
}

// SetInformationJobObject sets information on a Job Object.
func SetInformationJobObject(job windows.Handle, infoClass uint32, info unsafe.Pointer, length uint32) error {
	r1, _, e := procSetInformationJobObject.Call(
		uintptr(job),
		uintptr(infoClass),
		uintptr(info),
		uintptr(length),
	)
	if r1 == 0 {
		return fmt.Errorf("SetInformationJobObject: %w", e)
	}
	return nil
}

// SuspendThread suspends a thread; returns previous suspend count.
func SuspendThread(h windows.Handle) (uint32, error) {
	r1, _, e := procSuspendThread.Call(uintptr(h))
	if r1 == ^uintptr(0) {
		return 0, fmt.Errorf("SuspendThread: %w", e)
	}
	return uint32(r1), nil
}

// ResumeThread resumes a thread; returns previous suspend count.
func ResumeThread(h windows.Handle) (uint32, error) {
	r1, _, e := procResumeThread.Call(uintptr(h))
	if r1 == ^uintptr(0) {
		return 0, fmt.Errorf("ResumeThread: %w", e)
	}
	return uint32(r1), nil
}

// GetDiskFreeSpaceEx retrieves available, total, and free space on a drive.
func GetDiskFreeSpaceEx(path string) (freeAvail, totalBytes, totalFree uint64, err error) {
	pathPtr, e := windows.UTF16PtrFromString(path)
	if e != nil {
		err = e
		return
	}
	r1, _, callErr := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeAvail)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r1 == 0 {
		err = fmt.Errorf("GetDiskFreeSpaceExW: %w", callErr)
	}
	return
}

// GetLogicalDriveStrings returns a list of drive root paths.
func GetLogicalDriveStrings() ([]string, error) {
	var buf [256]uint16
	r1, _, e := procGetLogicalDriveStringsW.Call(
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("GetLogicalDriveStringsW: %w", e)
	}
	var result []string
	start := 0
	for i := 0; i < int(r1); i++ {
		if buf[i] == 0 {
			if i > start {
				result = append(result, windows.UTF16ToString(buf[start:i]))
			}
			start = i + 1
		}
	}
	return result, nil
}

// GetDriveType returns the drive type (DRIVE_*) for a root path.
func GetDriveType(root string) uint32 {
	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return DRIVE_UNKNOWN
	}
	r1, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootPtr)))
	return uint32(r1)
}

// GetVolumeInformation retrieves a drive's volume name and filesystem.
func GetVolumeInformation(root string) (label, fsName string) {
	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return "", ""
	}
	var labelBuf [261]uint16
	var fsBuf [261]uint16
	var serial, maxLen, flags uint32
	r1, _, _ := procGetVolumeInformationW.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&labelBuf[0])),
		uintptr(len(labelBuf)),
		uintptr(unsafe.Pointer(&serial)),
		uintptr(unsafe.Pointer(&maxLen)),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Pointer(&fsBuf[0])),
		uintptr(len(fsBuf)),
	)
	if r1 == 0 {
		return "", ""
	}
	return windows.UTF16ToString(labelBuf[:]), windows.UTF16ToString(fsBuf[:])
}
