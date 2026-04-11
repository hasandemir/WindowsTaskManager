//go:build windows

package winapi

import "golang.org/x/sys/windows"

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	ntdll    = windows.NewLazySystemDLL("ntdll.dll")
	psapi    = windows.NewLazySystemDLL("psapi.dll")
	iphlpapi = windows.NewLazySystemDLL("iphlpapi.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")
)

// kernel32 procs
var (
	procGetSystemTimes             = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx       = kernel32.NewProc("GlobalMemoryStatusEx")
	procCreateToolhelp32Snapshot   = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW            = kernel32.NewProc("Process32FirstW")
	procProcess32NextW             = kernel32.NewProc("Process32NextW")
	procThread32First              = kernel32.NewProc("Thread32First")
	procThread32Next               = kernel32.NewProc("Thread32Next")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procOpenThread                 = kernel32.NewProc("OpenThread")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procTerminateProcess           = kernel32.NewProc("TerminateProcess")
	procGetProcessTimes            = kernel32.NewProc("GetProcessTimes")
	procGetProcessIoCounters       = kernel32.NewProc("GetProcessIoCounters")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procIsProcessCritical          = kernel32.NewProc("IsProcessCritical")
	procSetPriorityClass           = kernel32.NewProc("SetPriorityClass")
	procGetPriorityClass           = kernel32.NewProc("GetPriorityClass")
	procSetProcessAffinityMask     = kernel32.NewProc("SetProcessAffinityMask")
	procGetProcessAffinityMask     = kernel32.NewProc("GetProcessAffinityMask")
	procCreateJobObjectW           = kernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject   = kernel32.NewProc("AssignProcessToJobObject")
	procSetInformationJobObject    = kernel32.NewProc("SetInformationJobObject")
	procSuspendThread              = kernel32.NewProc("SuspendThread")
	procResumeThread               = kernel32.NewProc("ResumeThread")
	procGetDiskFreeSpaceExW        = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetLogicalDriveStringsW    = kernel32.NewProc("GetLogicalDriveStringsW")
	procGetDriveTypeW              = kernel32.NewProc("GetDriveTypeW")
	procGetVolumeInformationW      = kernel32.NewProc("GetVolumeInformationW")
)

// ntdll procs
var (
	procNtQuerySystemInformation = ntdll.NewProc("NtQuerySystemInformation")
)

// psapi procs (kernel32 also exposes K32 variants but psapi is universal)
var (
	procGetProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
)

// iphlpapi procs
var (
	procGetExtendedTcpTable = iphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUdpTable = iphlpapi.NewProc("GetExtendedUdpTable")
	procGetIfTable2         = iphlpapi.NewProc("GetIfTable2")
	procFreeMibTable        = iphlpapi.NewProc("FreeMibTable")
)

// user32 procs
var (
	procIsHungAppWindow     = user32.NewProc("IsHungAppWindow")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
)

// shell32 procs
var (
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	procShellExecuteW    = shell32.NewProc("ShellExecuteW")
)

// advapi32 procs
var (
	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")
)
