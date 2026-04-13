//go:build windows

package winapi

import "math"

// FILETIME is Win32 FILETIME (100ns ticks since 1601-01-01).
type FILETIME struct {
	LowDateTime  uint32
	HighDateTime uint32
}

func (ft FILETIME) Ticks() uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

// FileTimeToUnix converts a Win32 FILETIME to a Unix timestamp (seconds).
func FileTimeToUnix(ft FILETIME) int64 {
	const epochDiff = 11644473600 // seconds between 1601 and 1970
	ticks := ft.Ticks()
	if ticks == 0 {
		return 0
	}
	seconds := ticks / 10000000
	if seconds >= epochDiff {
		delta := seconds - epochDiff
		if delta > uint64(math.MaxInt64) {
			return math.MaxInt64
		}
		return int64(delta)
	}
	delta := epochDiff - seconds
	if delta > uint64(math.MaxInt64) {
		return math.MinInt64
	}
	return -int64(delta)
}

// MEMORYSTATUSEX matches Win32 MEMORYSTATUSEX.
type MEMORYSTATUSEX struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// PROCESSENTRY32W matches Win32 PROCESSENTRY32W.
type PROCESSENTRY32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

// THREADENTRY32 matches Win32 THREADENTRY32.
type THREADENTRY32 struct {
	Size           uint32
	Usage          uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePri        int32
	DeltaPri       int32
	Flags          uint32
}

// PROCESS_MEMORY_COUNTERS_EX matches Win32 PROCESS_MEMORY_COUNTERS_EX.
type PROCESS_MEMORY_COUNTERS_EX struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

// IO_COUNTERS matches Win32 IO_COUNTERS.
type IO_COUNTERS struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION (NtQuerySystemInformation class 8).
type SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION struct {
	IdleTime   int64
	KernelTime int64
	UserTime   int64
	Reserved1  [2]int64
	Reserved2  uint32
}

// MIB_TCPROW_OWNER_PID matches Win32 IPv4 TCP row.
type MIB_TCPROW_OWNER_PID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPid  uint32
}

// MIB_TCP6ROW_OWNER_PID matches Win32 IPv6 TCP row.
type MIB_TCP6ROW_OWNER_PID struct {
	LocalAddr     [16]byte
	LocalScopeId  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeId uint32
	RemotePort    uint32
	State         uint32
	OwningPid     uint32
}

// MIB_UDPROW_OWNER_PID matches Win32 IPv4 UDP row.
type MIB_UDPROW_OWNER_PID struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPid uint32
}

// MIB_UDP6ROW_OWNER_PID matches Win32 IPv6 UDP row.
type MIB_UDP6ROW_OWNER_PID struct {
	LocalAddr    [16]byte
	LocalScopeId uint32
	LocalPort    uint32
	OwningPid    uint32
}

// JOBOBJECT_BASIC_LIMIT_INFORMATION matches Win32 struct.
type JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

// JOBOBJECT_EXTENDED_LIMIT_INFORMATION matches Win32 struct.
type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                IO_COUNTERS
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// JOBOBJECT_CPU_RATE_CONTROL_INFORMATION matches Win32 struct.
type JOBOBJECT_CPU_RATE_CONTROL_INFORMATION struct {
	ControlFlags uint32
	CpuRate      uint32
}

// NOTIFYICONDATAW matches Win32 NOTIFYICONDATAW.
type NOTIFYICONDATAW struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Union           uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GuidItem        [16]byte
	BalloonIcon     uintptr
}

// POINT struct (cursor position).
type POINT struct {
	X int32
	Y int32
}

// MSG struct used by message pump.
type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// WNDCLASSEXW struct used by RegisterClassExW.
type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

// TCP states (MIB_TCP_STATE_*).
const (
	MIB_TCP_STATE_CLOSED     = 1
	MIB_TCP_STATE_LISTEN     = 2
	MIB_TCP_STATE_SYN_SENT   = 3
	MIB_TCP_STATE_SYN_RCVD   = 4
	MIB_TCP_STATE_ESTAB      = 5
	MIB_TCP_STATE_FIN_WAIT1  = 6
	MIB_TCP_STATE_FIN_WAIT2  = 7
	MIB_TCP_STATE_CLOSE_WAIT = 8
	MIB_TCP_STATE_CLOSING    = 9
	MIB_TCP_STATE_LAST_ACK   = 10
	MIB_TCP_STATE_TIME_WAIT  = 11
	MIB_TCP_STATE_DELETE_TCB = 12
)

// TCP_TABLE_CLASS values for GetExtendedTcpTable.
const (
	TCP_TABLE_OWNER_PID_ALL = 5
	UDP_TABLE_OWNER_PID     = 1
)

// CreateToolhelp32Snapshot flags.
const (
	TH32CS_SNAPPROCESS = 0x00000002
	TH32CS_SNAPTHREAD  = 0x00000004
)

// Job Object information class values.
const (
	JobObjectBasicLimitInformation     = 2
	JobObjectExtendedLimitInformation  = 9
	JobObjectCpuRateControlInformation = 15
)

// Job Object limit flags.
const (
	JOB_OBJECT_LIMIT_PROCESS_MEMORY = 0x00000100
)

// Job Object CPU rate control flags.
const (
	JOB_OBJECT_CPU_RATE_CONTROL_ENABLE   = 0x00000001
	JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP = 0x00000004
)

// Priority class values.
const (
	IDLE_PRIORITY_CLASS         = 0x00000040
	BELOW_NORMAL_PRIORITY_CLASS = 0x00004000
	NORMAL_PRIORITY_CLASS       = 0x00000020
	ABOVE_NORMAL_PRIORITY_CLASS = 0x00008000
	HIGH_PRIORITY_CLASS         = 0x00000080
	REALTIME_PRIORITY_CLASS     = 0x00000100
)

// Shell_NotifyIcon flags.
const (
	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_INFO    = 0x00000010

	NIIF_INFO    = 0x00000001
	NIIF_WARNING = 0x00000002
	NIIF_ERROR   = 0x00000003
)

// Tray window message constants.
const (
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 1
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_COMMAND       = 0x0111
	WM_DESTROY       = 0x0002
)

// ShowWindow / ShellExecute constants.
const (
	SW_SHOWNORMAL = 1
)

// HWND_MESSAGE for message-only windows.
const HWND_MESSAGE = ^uintptr(2) //nolint:gocritic // 0xFFFFFFFD

// Drive types.
const (
	DRIVE_UNKNOWN   = 0
	DRIVE_REMOVABLE = 2
	DRIVE_FIXED     = 3
	DRIVE_REMOTE    = 4
)

// Address families.
const (
	AF_INET  = 2
	AF_INET6 = 23
)
