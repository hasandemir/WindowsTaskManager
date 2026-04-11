# WindowsTaskManager — IMPLEMENTATION

**Repository:** `github.com/ersinkoc/WindowsTaskManager`
**Specification:** See SPECIFICATION.md
**This Document:** Detailed implementation guide for each component

---

## 1. Windows API Layer (`internal/winapi/`)

All Windows API access is centralized in the `winapi` package. No other package imports `syscall` or `golang.org/x/sys/windows` directly. This ensures a single point of maintenance for all Win32 interop.

### 1.1 DLL Loading Pattern

```go
// Every DLL is loaded once at package init via lazy proc loading
var (
    kernel32 = windows.NewLazySystemDLL("kernel32.dll")
    ntdll    = windows.NewLazySystemDLL("ntdll.dll")
    psapi    = windows.NewLazySystemDLL("psapi.dll")
    iphlpapi = windows.NewLazySystemDLL("iphlpapi.dll")
    user32   = windows.NewLazySystemDLL("user32.dll")
    shell32  = windows.NewLazySystemDLL("shell32.dll")
    gdi32    = windows.NewLazySystemDLL("gdi32.dll")
    advapi32 = windows.NewLazySystemDLL("advapi32.dll")
)

// Procs are resolved lazily on first call
var (
    procGetSystemTimes              = kernel32.NewProc("GetSystemTimes")
    procGlobalMemoryStatusEx        = kernel32.NewProc("GlobalMemoryStatusEx")
    procCreateToolhelp32Snapshot    = kernel32.NewProc("CreateToolhelp32Snapshot")
    procProcess32FirstW             = kernel32.NewProc("Process32FirstW")
    procProcess32NextW              = kernel32.NewProc("Process32NextW")
    procOpenProcess                 = kernel32.NewProc("OpenProcess")
    procTerminateProcess            = kernel32.NewProc("TerminateProcess")
    procGetProcessTimes             = kernel32.NewProc("GetProcessTimes")
    procGetProcessMemoryInfo        = psapi.NewProc("GetProcessMemoryInfo")
    procGetProcessIoCounters        = kernel32.NewProc("GetProcessIoCounters")
    procQueryFullProcessImageNameW  = kernel32.NewProc("QueryFullProcessImageNameW")
    procIsProcessCritical           = kernel32.NewProc("IsProcessCritical")
    procSetPriorityClass            = kernel32.NewProc("SetPriorityClass")
    procSetProcessAffinityMask      = kernel32.NewProc("SetProcessAffinityMask")
    procCreateJobObjectW            = kernel32.NewProc("CreateJobObjectW")
    procAssignProcessToJobObject    = kernel32.NewProc("AssignProcessToJobObject")
    procSetInformationJobObject     = kernel32.NewProc("SetInformationJobObject")
    procSuspendThread               = kernel32.NewProc("SuspendThread")
    procResumeThread                = kernel32.NewProc("ResumeThread")
    procGetDiskFreeSpaceExW         = kernel32.NewProc("GetDiskFreeSpaceExW")
    procDeviceIoControl             = kernel32.NewProc("DeviceIoControl")
    procGetIfTable2                 = iphlpapi.NewProc("GetIfTable2")
    procFreeMibTable                = iphlpapi.NewProc("FreeMibTable")
    procGetExtendedTcpTable         = iphlpapi.NewProc("GetExtendedTcpTable")
    procGetExtendedUdpTable         = iphlpapi.NewProc("GetExtendedUdpTable")
    procShellNotifyIconW            = shell32.NewProc("Shell_NotifyIconW")
    procIsHungAppWindow             = user32.NewProc("IsHungAppWindow")
    procNtQuerySystemInformation    = ntdll.NewProc("NtQuerySystemInformation")
    procD3DKMTQueryStatistics       = gdi32.NewProc("D3DKMTQueryStatistics")
    procD3DKMTEnumAdapters2         = gdi32.NewProc("D3DKMTEnumAdapters2")
)
```

### 1.2 types.go — Windows Struct Definitions

All Windows API struct definitions live in a single `types.go` file. Key structures:

```go
// FILETIME — used by GetSystemTimes, GetProcessTimes
type FILETIME struct {
    LowDateTime  uint32
    HighDateTime uint32
}

func (ft FILETIME) Ticks() uint64 {
    return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

// MEMORYSTATUSEX — GlobalMemoryStatusEx
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

// PROCESSENTRY32W — Process32FirstW/NextW
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
    ExeFile         [260]uint16 // MAX_PATH
}

// PROCESS_MEMORY_COUNTERS_EX — GetProcessMemoryInfo
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

// IO_COUNTERS — GetProcessIoCounters
type IO_COUNTERS struct {
    ReadOperationCount  uint64
    WriteOperationCount uint64
    OtherOperationCount uint64
    ReadTransferCount   uint64
    WriteTransferCount  uint64
    OtherTransferCount  uint64
}

// MIB_TCPROW_OWNER_PID — GetExtendedTcpTable
type MIB_TCPROW_OWNER_PID struct {
    State      uint32
    LocalAddr  uint32
    LocalPort  uint32
    RemoteAddr uint32
    RemotePort uint32
    OwningPid  uint32
}

// MIB_TCP6ROW_OWNER_PID — GetExtendedTcpTable (IPv6)
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

// MIB_UDPROW_OWNER_PID — GetExtendedUdpTable
type MIB_UDPROW_OWNER_PID struct {
    LocalAddr uint32
    LocalPort uint32
    OwningPid uint32
}

// MIB_IF_ROW2 — GetIfTable2 (partial, key fields only)
type MIB_IF_ROW2 struct {
    InterfaceLuid        uint64
    InterfaceIndex       uint32
    InterfaceGuid        [16]byte
    Alias                [257]uint16 // IF_MAX_STRING_SIZE + 1
    Description          [257]uint16
    PhysicalAddressLength uint32
    PhysicalAddress      [32]byte
    PermanentPhysicalAddress [32]byte
    Mtu                  uint32
    Type                 uint32
    // ... tunnel type, media type fields
    OperStatus           uint32
    AdminStatus          uint32
    MediaConnectState    uint32
    NetworkGuid          [16]byte
    ConnectionType       uint32
    // ... padding
    TransmitLinkSpeed    uint64
    ReceiveLinkSpeed     uint64
    InOctets             uint64
    InUcastPkts          uint64
    InNUcastPkts         uint64
    InDiscards           uint64
    InErrors             uint64
    InUnknownProtos      uint64
    InUcastOctets        uint64
    InMulticastOctets    uint64
    InBroadcastOctets    uint64
    OutOctets            uint64
    OutUcastPkts         uint64
    OutNUcastPkts        uint64
    OutDiscards          uint64
    OutErrors            uint64
    OutUcastOctets       uint64
    OutMulticastOctets   uint64
    OutBroadcastOctets   uint64
}

// JOBOBJECT_CPU_RATE_CONTROL_INFORMATION
type JOBOBJECT_CPU_RATE_CONTROL_INFORMATION struct {
    ControlFlags uint32
    CpuRate      uint32 // percentage * 100 (e.g. 30% = 3000)
}

// JOBOBJECT_EXTENDED_LIMIT_INFORMATION
type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
    BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
    IoInfo                IO_COUNTERS
    ProcessMemoryLimit    uintptr
    JobMemoryLimit        uintptr
    PeakProcessMemoryUsed uintptr
    PeakJobMemoryUsed     uintptr
}

// JOBOBJECT_BASIC_LIMIT_INFORMATION
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

// DISK_PERFORMANCE — DeviceIoControl(IOCTL_DISK_PERFORMANCE)
type DISK_PERFORMANCE struct {
    BytesRead           int64
    BytesWritten        int64
    ReadTime            int64
    WriteTime           int64
    IdleTime            int64
    ReadCount           uint32
    WriteCount          uint32
    QueueDepth          uint32
    SplitCount          uint32
    QueryTime           int64
    StorageDeviceNumber uint32
    StorageManagerName  [8]uint16
}

// NOTIFYICONDATAW — Shell_NotifyIconW
type NOTIFYICONDATAW struct {
    Size            uint32
    Wnd             uintptr // HWND
    ID              uint32
    Flags           uint32
    CallbackMessage uint32
    Icon            uintptr // HICON
    Tip             [128]uint16
    State           uint32
    StateMask       uint32
    Info            [256]uint16
    Union           uint32 // timeout or version
    InfoTitle       [64]uint16
    InfoFlags       uint32
    GuidItem        [16]byte
    BalloonIcon     uintptr
}

// SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION — NtQuerySystemInformation
type SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION struct {
    IdleTime   int64
    KernelTime int64
    UserTime   int64
    Reserved1  [2]int64
    Reserved2  uint32
}

// TCP connection states
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
```

### 1.3 Error Handling Pattern

All Win32 calls follow a consistent error pattern:

```go
func GetProcessMemoryInfo(handle windows.Handle) (*PROCESS_MEMORY_COUNTERS_EX, error) {
    var counters PROCESS_MEMORY_COUNTERS_EX
    counters.CB = uint32(unsafe.Sizeof(counters))
    r1, _, err := procGetProcessMemoryInfo.Call(
        uintptr(handle),
        uintptr(unsafe.Pointer(&counters)),
        uintptr(counters.CB),
    )
    if r1 == 0 {
        return nil, fmt.Errorf("GetProcessMemoryInfo: %w", err)
    }
    return &counters, nil
}
```

Key rules:
- Always check `r1 == 0` for BOOL-returning functions
- Always close handles with `defer windows.CloseHandle(handle)`
- Use `uintptr(unsafe.Pointer(...))` only in `.Call()` arguments (never store)
- Access denied errors are not fatal — skip process gracefully

---

## 2. Collector Engine (`internal/collector/`)

### 2.1 Orchestrator (`collector.go`)

```go
type Collector struct {
    config    *config.MonitoringConfig
    store     *storage.Store        // ring buffer storage
    emitter   *event.Emitter        // SSE event emitter
    
    cpuCollector     *CPUCollector
    memCollector     *MemCollector
    diskCollector    *DiskCollector
    netCollector     *NetCollector
    gpuCollector     *GPUCollector
    processCollector *ProcessCollector
    portCollector    *PortCollector
    
    mu       sync.RWMutex
    snapshot *SystemSnapshot        // latest full snapshot
}

// Start launches all collection goroutines
func (c *Collector) Start(ctx context.Context) {
    // Main tick: CPU + Memory + Process (every interval, default 1s)
    go c.runTick(ctx, c.config.Interval, func() {
        c.collectCPU()
        c.collectMemory()
        c.collectProcesses()
        c.buildSnapshot()
        c.emitter.Emit("system_metrics", c.snapshot)
    })

    // Process tree (every 2s)
    go c.runTick(ctx, c.config.ProcessTreeInterval, func() {
        c.collectProcessTree()
        c.emitter.Emit("process_tree", c.snapshot.ProcessTree)
    })

    // Port scan (every 3s)
    go c.runTick(ctx, c.config.PortScanInterval, func() {
        c.collectPorts()
        c.emitter.Emit("port_map", c.snapshot.PortMap)
    })

    // GPU (every 2s)
    go c.runTick(ctx, c.config.GPUInterval, func() {
        c.collectGPU()
    })

    // Disk (every 5s — disk I/O counters don't change as fast)
    go c.runTick(ctx, 5*time.Second, func() {
        c.collectDisk()
    })
}

func (c *Collector) runTick(ctx context.Context, interval time.Duration, fn func()) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            fn()
        }
    }
}
```

### 2.2 CPU Collector (`cpu.go`)

```go
type CPUCollector struct {
    prevIdle   uint64
    prevKernel uint64
    prevUser   uint64
    prevPerCore []coreTimes
    numCPU     int
}

type coreTimes struct {
    idle, kernel, user uint64
}

type CPUMetrics struct {
    TotalPercent float64      `json:"total_percent"`
    PerCore      []float64    `json:"per_core"`
    NumLogical   int          `json:"num_logical"`
    Name         string       `json:"name"`       // from registry
    FreqMHz      uint32       `json:"freq_mhz"`
}

func (c *CPUCollector) Collect() (*CPUMetrics, error) {
    // 1. GetSystemTimes for overall CPU
    var idle, kernel, user winapi.FILETIME
    if err := winapi.GetSystemTimes(&idle, &kernel, &user); err != nil {
        return nil, err
    }
    
    idleTicks := idle.Ticks()
    kernelTicks := kernel.Ticks() // includes idle
    userTicks := user.Ticks()
    
    deltaIdle := idleTicks - c.prevIdle
    deltaKernel := kernelTicks - c.prevKernel
    deltaUser := userTicks - c.prevUser
    deltaTotal := deltaKernel + deltaUser // kernel includes idle
    deltaActive := deltaTotal - deltaIdle
    
    var totalPercent float64
    if deltaTotal > 0 {
        totalPercent = float64(deltaActive) / float64(deltaTotal) * 100.0
    }
    
    c.prevIdle = idleTicks
    c.prevKernel = kernelTicks
    c.prevUser = userTicks
    
    // 2. NtQuerySystemInformation for per-core
    perCore := c.collectPerCore()
    
    // 3. CPU name from registry (cached, read once)
    // 4. Current frequency from CallNtPowerInformation (or registry)
    
    return &CPUMetrics{
        TotalPercent: totalPercent,
        PerCore:      perCore,
        NumLogical:   c.numCPU,
        Name:         c.cpuName,
        FreqMHz:      c.currentFreqMHz,
    }, nil
}
```

**Per-core collection** uses `NtQuerySystemInformation` with `SystemProcessorPerformanceInformation` (class 8). Returns array of `SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION`, one per logical processor. Same delta calculation per core.

### 2.3 Memory Collector (`memory.go`)

```go
type MemoryMetrics struct {
    TotalPhys     uint64  `json:"total_phys"`
    AvailPhys     uint64  `json:"avail_phys"`
    UsedPhys      uint64  `json:"used_phys"`
    UsedPercent   float64 `json:"used_percent"`
    TotalPageFile uint64  `json:"total_page_file"`
    AvailPageFile uint64  `json:"avail_page_file"`
    CommitCharge  uint64  `json:"commit_charge"`
}

func (c *MemCollector) Collect() (*MemoryMetrics, error) {
    var ms winapi.MEMORYSTATUSEX
    ms.Length = uint32(unsafe.Sizeof(ms))
    if err := winapi.GlobalMemoryStatusEx(&ms); err != nil {
        return nil, err
    }
    
    used := ms.TotalPhys - ms.AvailPhys
    return &MemoryMetrics{
        TotalPhys:     ms.TotalPhys,
        AvailPhys:     ms.AvailPhys,
        UsedPhys:      used,
        UsedPercent:   float64(used) / float64(ms.TotalPhys) * 100.0,
        TotalPageFile: ms.TotalPageFile,
        AvailPageFile: ms.AvailPageFile,
        CommitCharge:  ms.TotalPageFile - ms.AvailPageFile,
    }, nil
}
```

### 2.4 Process Collector (`process.go`)

This is the most complex collector — enumerates all processes and gathers per-process metrics.

```go
type ProcessInfo struct {
    PID             uint32  `json:"pid"`
    ParentPID       uint32  `json:"parent_pid"`
    Name            string  `json:"name"`
    ExePath         string  `json:"exe_path"`
    CPUPercent      float64 `json:"cpu_percent"`
    WorkingSet      uint64  `json:"working_set"`       // bytes
    PrivateBytes    uint64  `json:"private_bytes"`      // bytes
    PageFaults      uint32  `json:"page_faults"`
    IOReadBytes     uint64  `json:"io_read_bytes"`      // delta bytes/sec
    IOWriteBytes    uint64  `json:"io_write_bytes"`     // delta bytes/sec
    IOReadOps       uint64  `json:"io_read_ops"`        // delta ops/sec
    IOWriteOps      uint64  `json:"io_write_ops"`       // delta ops/sec
    ThreadCount     uint32  `json:"thread_count"`
    HandleCount     uint32  `json:"handle_count"`
    CreateTime      int64   `json:"create_time"`        // unix timestamp
    IsCritical      bool    `json:"is_critical"`
    Status          string  `json:"status"`             // normal, hung, orphan, etc.
    Connections     int     `json:"connections"`         // TCP+UDP count
    PriorityClass   uint32  `json:"priority_class"`
}

type processState struct {
    prevKernelTime uint64
    prevUserTime   uint64
    prevIORead     uint64
    prevIOWrite    uint64
    prevIOReadOps  uint64
    prevIOWriteOps uint64
    firstSeen      time.Time
}

type ProcessCollector struct {
    states   map[uint32]*processState // PID → previous state
    interval time.Duration
    mu       sync.Mutex
}

func (c *ProcessCollector) Collect() ([]ProcessInfo, error) {
    // 1. Create snapshot
    snap, err := winapi.CreateToolhelp32Snapshot(winapi.TH32CS_SNAPPROCESS, 0)
    if err != nil {
        return nil, err
    }
    defer windows.CloseHandle(snap)
    
    // 2. Enumerate all processes
    var entry winapi.PROCESSENTRY32W
    entry.Size = uint32(unsafe.Sizeof(entry))
    
    currentPIDs := make(map[uint32]bool)
    var processes []ProcessInfo
    
    err = winapi.Process32FirstW(snap, &entry)
    for err == nil {
        pid := entry.ProcessID
        currentPIDs[pid] = true
        
        info := ProcessInfo{
            PID:         pid,
            ParentPID:   entry.ParentProcessID,
            Name:        windows.UTF16ToString(entry.ExeFile[:]),
            ThreadCount: entry.Threads,
        }
        
        // 3. Open process for detailed info (may fail for system processes)
        if detail, err := c.collectProcessDetail(pid); err == nil {
            info.ExePath = detail.exePath
            info.CPUPercent = detail.cpuPercent
            info.WorkingSet = detail.workingSet
            info.PrivateBytes = detail.privateBytes
            info.PageFaults = detail.pageFaults
            info.IOReadBytes = detail.ioReadBytesPerSec
            info.IOWriteBytes = detail.ioWriteBytesPerSec
            info.IOReadOps = detail.ioReadOpsPerSec
            info.IOWriteOps = detail.ioWriteOpsPerSec
            info.CreateTime = detail.createTime
            info.IsCritical = detail.isCritical
            info.PriorityClass = detail.priorityClass
        }
        
        processes = append(processes, info)
        err = winapi.Process32NextW(snap, &entry)
    }
    
    // 4. Clean up states for dead processes
    c.mu.Lock()
    for pid := range c.states {
        if !currentPIDs[pid] {
            delete(c.states, pid)
        }
    }
    c.mu.Unlock()
    
    return processes, nil
}

func (c *ProcessCollector) collectProcessDetail(pid uint32) (*processDetail, error) {
    // Open with minimum required access
    access := uint32(windows.PROCESS_QUERY_LIMITED_INFORMATION | windows.PROCESS_VM_READ)
    handle, err := windows.OpenProcess(access, false, pid)
    if err != nil {
        return nil, err // access denied is common, not fatal
    }
    defer windows.CloseHandle(handle)
    
    detail := &processDetail{}
    
    // CPU times → compute delta for CPU%
    var creation, exit, kernel, user winapi.FILETIME
    if err := winapi.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err == nil {
        kernelTicks := kernel.Ticks()
        userTicks := user.Ticks()
        
        c.mu.Lock()
        state, exists := c.states[pid]
        if !exists {
            state = &processState{firstSeen: time.Now()}
            c.states[pid] = state
        }
        
        if exists {
            deltaKernel := kernelTicks - state.prevKernelTime
            deltaUser := userTicks - state.prevUserTime
            totalDelta := deltaKernel + deltaUser
            // Convert 100ns ticks to percentage over interval
            intervalTicks := uint64(c.interval.Nanoseconds() / 100)
            numCPU := uint64(runtime.NumCPU())
            detail.cpuPercent = float64(totalDelta) / float64(intervalTicks*numCPU) * 100.0
        }
        
        state.prevKernelTime = kernelTicks
        state.prevUserTime = userTicks
        c.mu.Unlock()
        
        detail.createTime = winapi.FileTimeToUnix(creation)
    }
    
    // Memory
    if mem, err := winapi.GetProcessMemoryInfo(handle); err == nil {
        detail.workingSet = uint64(mem.WorkingSetSize)
        detail.privateBytes = uint64(mem.PrivateUsage)
        detail.pageFaults = mem.PageFaultCount
    }
    
    // I/O counters (delta)
    if io, err := winapi.GetProcessIoCounters(handle); err == nil {
        c.mu.Lock()
        state := c.states[pid]
        if state.prevIORead > 0 { // not first sample
            seconds := c.interval.Seconds()
            detail.ioReadBytesPerSec = uint64(float64(io.ReadTransferCount-state.prevIORead) / seconds)
            detail.ioWriteBytesPerSec = uint64(float64(io.WriteTransferCount-state.prevIOWrite) / seconds)
            detail.ioReadOpsPerSec = uint64(float64(io.ReadOperationCount-state.prevIOReadOps) / seconds)
            detail.ioWriteOpsPerSec = uint64(float64(io.WriteOperationCount-state.prevIOWriteOps) / seconds)
        }
        state.prevIORead = io.ReadTransferCount
        state.prevIOWrite = io.WriteTransferCount
        state.prevIOReadOps = io.ReadOperationCount
        state.prevIOWriteOps = io.WriteOperationCount
        c.mu.Unlock()
    }
    
    // Executable path
    detail.exePath, _ = winapi.QueryFullProcessImageName(handle)
    
    // Critical process check
    detail.isCritical, _ = winapi.IsProcessCritical(handle)
    
    return detail, nil
}
```

### 2.5 Process Tree Builder (`process_tree.go`)

```go
type ProcessNode struct {
    Process  ProcessInfo    `json:"process"`
    Children []*ProcessNode `json:"children,omitempty"`
    Depth    int            `json:"depth"`
    IsOrphan bool           `json:"is_orphan"`
}

func BuildProcessTree(processes []ProcessInfo) []*ProcessNode {
    // 1. Build PID → ProcessInfo map
    pidMap := make(map[uint32]*ProcessNode, len(processes))
    for i := range processes {
        pidMap[processes[i].PID] = &ProcessNode{Process: processes[i]}
    }
    
    // 2. Build parent-child relationships
    var roots []*ProcessNode
    for _, node := range pidMap {
        parentPID := node.Process.ParentPID
        if parentPID == 0 || parentPID == node.Process.PID {
            roots = append(roots, node)
            continue
        }
        parent, exists := pidMap[parentPID]
        if exists {
            parent.Children = append(parent.Children, node)
        } else {
            // Parent doesn't exist → orphan
            node.IsOrphan = true
            roots = append(roots, node)
        }
    }
    
    // 3. Compute depths
    var setDepth func(node *ProcessNode, depth int)
    setDepth = func(node *ProcessNode, depth int) {
        node.Depth = depth
        for _, child := range node.Children {
            setDepth(child, depth+1)
        }
    }
    for _, root := range roots {
        setDepth(root, 0)
    }
    
    // 4. Sort children by PID for stable output
    var sortChildren func(node *ProcessNode)
    sortChildren = func(node *ProcessNode) {
        sort.Slice(node.Children, func(i, j int) bool {
            return node.Children[i].Process.PID < node.Children[j].Process.PID
        })
        for _, child := range node.Children {
            sortChildren(child)
        }
    }
    for _, root := range roots {
        sortChildren(root)
    }
    
    return roots
}

// TreeStats computes aggregate stats for a subtree
func TreeStats(node *ProcessNode) (childCount int, totalMem uint64, totalCPU float64) {
    childCount = len(node.Children)
    totalMem = node.Process.WorkingSet
    totalCPU = node.Process.CPUPercent
    for _, child := range node.Children {
        cc, cm, ccpu := TreeStats(child)
        childCount += cc
        totalMem += cm
        totalCPU += ccpu
    }
    return
}
```

### 2.6 Port Collector (`ports.go`)

```go
type PortBinding struct {
    Protocol   string `json:"protocol"`    // "tcp", "tcp6", "udp", "udp6"
    LocalAddr  string `json:"local_addr"`
    LocalPort  uint16 `json:"local_port"`
    RemoteAddr string `json:"remote_addr"` // TCP only
    RemotePort uint16 `json:"remote_port"` // TCP only
    State      string `json:"state"`       // LISTEN, ESTABLISHED, etc.
    StateCode  uint32 `json:"state_code"`
    PID        uint32 `json:"pid"`
    Process    string `json:"process"`     // filled in post-processing
    Label      string `json:"label"`       // well-known port label
    Since      int64  `json:"since"`       // first seen timestamp
}

func (c *PortCollector) Collect(processes map[uint32]string) ([]PortBinding, error) {
    var bindings []PortBinding
    
    // TCP IPv4
    tcpTable, err := c.getTcpTable(windows.AF_INET)
    if err == nil {
        for _, row := range tcpTable {
            b := PortBinding{
                Protocol:   "tcp",
                LocalAddr:  intToIPv4(row.LocalAddr),
                LocalPort:  ntohs(row.LocalPort),
                RemoteAddr: intToIPv4(row.RemoteAddr),
                RemotePort: ntohs(row.RemotePort),
                State:      tcpStateString(row.State),
                StateCode:  row.State,
                PID:        row.OwningPid,
                Process:    processes[row.OwningPid],
            }
            bindings = append(bindings, b)
        }
    }
    
    // TCP IPv6
    tcp6Table, err := c.getTcp6Table(windows.AF_INET6)
    if err == nil {
        for _, row := range tcp6Table {
            b := PortBinding{
                Protocol:   "tcp6",
                LocalAddr:  bytesToIPv6(row.LocalAddr),
                LocalPort:  ntohs(row.LocalPort),
                RemoteAddr: bytesToIPv6(row.RemoteAddr),
                RemotePort: ntohs(row.RemotePort),
                State:      tcpStateString(row.State),
                StateCode:  row.State,
                PID:        row.OwningPid,
                Process:    processes[row.OwningPid],
            }
            bindings = append(bindings, b)
        }
    }
    
    // UDP IPv4
    udpTable, err := c.getUdpTable(windows.AF_INET)
    if err == nil {
        for _, row := range udpTable {
            b := PortBinding{
                Protocol:  "udp",
                LocalAddr: intToIPv4(row.LocalAddr),
                LocalPort: ntohs(row.LocalPort),
                State:     "LISTEN",
                PID:       row.OwningPid,
                Process:   processes[row.OwningPid],
            }
            bindings = append(bindings, b)
        }
    }
    
    // UDP IPv6 — same pattern
    
    // Post-process: add labels, track "since" times
    c.addLabels(bindings)
    c.trackFirstSeen(bindings)
    
    return bindings, nil
}

// getTcpTable calls GetExtendedTcpTable with dynamic buffer sizing
func (c *PortCollector) getTcpTable(family uint32) ([]winapi.MIB_TCPROW_OWNER_PID, error) {
    var size uint32
    // First call to get required buffer size
    winapi.GetExtendedTcpTable(nil, &size, true, family, winapi.TCP_TABLE_OWNER_PID_ALL, 0)
    
    buf := make([]byte, size)
    if err := winapi.GetExtendedTcpTable(&buf[0], &size, true, family, winapi.TCP_TABLE_OWNER_PID_ALL, 0); err != nil {
        return nil, err
    }
    
    // Parse: first 4 bytes = count, then array of MIB_TCPROW_OWNER_PID
    count := *(*uint32)(unsafe.Pointer(&buf[0]))
    rows := make([]winapi.MIB_TCPROW_OWNER_PID, count)
    rowSize := unsafe.Sizeof(winapi.MIB_TCPROW_OWNER_PID{})
    for i := uint32(0); i < count; i++ {
        offset := 4 + uintptr(i)*rowSize
        rows[i] = *(*winapi.MIB_TCPROW_OWNER_PID)(unsafe.Pointer(&buf[offset]))
    }
    return rows, nil
}

// ntohs converts network byte order port to host byte order
func ntohs(port uint32) uint16 {
    return uint16((port>>8)&0xFF) | uint16((port&0xFF)<<8)
}
```

### 2.7 GPU Collector (`gpu.go`)

```go
type GPUMetrics struct {
    Name          string  `json:"name"`
    Utilization   float64 `json:"utilization"`    // 0-100%
    VRAMUsed      uint64  `json:"vram_used"`
    VRAMTotal     uint64  `json:"vram_total"`
    Temperature   int     `json:"temperature"`    // celsius, -1 if unavailable
    Available     bool    `json:"available"`
}

type GPUCollector struct {
    d3dkmt    *D3DKMTProvider  // works for all vendors
    nvml      *NVMLProvider    // optional, NVIDIA only
    available bool
}

// D3DKMT approach — vendor-agnostic GPU monitoring
// Uses gdi32.dll!D3DKMTQueryStatistics and D3DKMTEnumAdapters2
// This works with Intel, AMD, and NVIDIA GPUs

func (c *GPUCollector) initD3DKMT() error {
    // 1. Enumerate adapters via D3DKMTEnumAdapters2
    // 2. For each adapter, get adapter LUID
    // 3. Use LUID for subsequent D3DKMTQueryStatistics calls
    // Returns adapter name from D3DKMT_ADAPTERINFO.Description
    return nil
}

func (c *GPUCollector) collectD3DKMT() (*GPUMetrics, error) {
    // D3DKMTQueryStatistics with D3DKMT_QUERYSTATISTICS_TYPE:
    //   D3DKMT_QUERYSTATISTICS_ADAPTER     → adapter-level stats
    //   D3DKMT_QUERYSTATISTICS_SEGMENT     → per-segment memory (VRAM)
    //   D3DKMT_QUERYSTATISTICS_NODE        → per-engine utilization
    //
    // VRAM: Sum all segments where Aperture == 0 (dedicated VRAM)
    // Utilization: Compute from node running time deltas
    return nil, nil
}

// NVML approach — NVIDIA-specific, more detailed
// Loads nvml.dll dynamically via LoadLibrary + GetProcAddress
// No CGo, no NVML SDK needed at compile time

type NVMLProvider struct {
    dll          *windows.DLL
    nvmlInit     *windows.Proc
    nvmlShutdown *windows.Proc
    deviceGetCount          *windows.Proc
    deviceGetHandleByIndex  *windows.Proc
    deviceGetName           *windows.Proc
    deviceGetUtilizationRates *windows.Proc
    deviceGetMemoryInfo     *windows.Proc
    deviceGetTemperature    *windows.Proc
    deviceGetFanSpeed       *windows.Proc
    deviceGetPowerUsage     *windows.Proc
}

func NewNVMLProvider() (*NVMLProvider, error) {
    // Try to load nvml.dll from:
    // 1. System PATH
    // 2. C:\Program Files\NVIDIA Corporation\NVSMI\nvml.dll
    // 3. C:\Windows\System32\nvml.dll
    // If not found, NVML is simply unavailable (not an error)
    dll, err := windows.LoadDLL("nvml.dll")
    if err != nil {
        return nil, err // NVML not available, fall back to D3DKMT
    }
    // Resolve all procs...
    return &NVMLProvider{dll: dll}, nil
}
```

### 2.8 Disk Collector (`disk.go`)

```go
type DiskMetrics struct {
    Drives []DriveInfo `json:"drives"`
}

type DriveInfo struct {
    Letter     string  `json:"letter"`       // "C:", "D:"
    Label      string  `json:"label"`        // Volume label
    FSType     string  `json:"fs_type"`      // NTFS, FAT32, etc.
    TotalBytes uint64  `json:"total_bytes"`
    FreeBytes  uint64  `json:"free_bytes"`
    UsedBytes  uint64  `json:"used_bytes"`
    UsedPct    float64 `json:"used_pct"`
    ReadBPS    uint64  `json:"read_bps"`     // bytes per second (delta)
    WriteBPS   uint64  `json:"write_bps"`    // bytes per second (delta)
    ReadIOPS   uint64  `json:"read_iops"`    // I/O ops per second (delta)
    WriteIOPS  uint64  `json:"write_iops"`   // I/O ops per second (delta)
    QueueDepth uint32  `json:"queue_depth"`
}

func (c *DiskCollector) Collect() (*DiskMetrics, error) {
    // 1. Get drive letters via GetLogicalDriveStringsW
    // 2. For each fixed drive (GetDriveTypeW == DRIVE_FIXED):
    //    a. GetDiskFreeSpaceExW → total, free
    //    b. GetVolumeInformationW → label, filesystem
    //    c. DeviceIoControl(IOCTL_DISK_PERFORMANCE) → I/O counters
    //       (requires opening \\.\PhysicalDriveN with CreateFileW)
    //    d. Delta calculation for I/O rates
    return nil, nil
}
```

### 2.9 Network Collector (`network.go`)

```go
type NetworkMetrics struct {
    Interfaces     []InterfaceInfo `json:"interfaces"`
    TotalUpBPS     uint64          `json:"total_up_bps"`
    TotalDownBPS   uint64          `json:"total_down_bps"`
}

type InterfaceInfo struct {
    Name       string `json:"name"`
    Type       string `json:"type"`        // Ethernet, WiFi, Loopback
    Status     string `json:"status"`      // Up, Down
    SpeedMbps  uint64 `json:"speed_mbps"`
    InBPS      uint64 `json:"in_bps"`      // bytes per second (delta)
    OutBPS     uint64 `json:"out_bps"`     // bytes per second (delta)
    InPPS      uint64 `json:"in_pps"`      // packets per second (delta)
    OutPPS     uint64 `json:"out_pps"`     // packets per second (delta)
    InErrors   uint64 `json:"in_errors"`
    OutErrors  uint64 `json:"out_errors"`
}

func (c *NetCollector) Collect() (*NetworkMetrics, error) {
    // GetIfTable2 → array of MIB_IF_ROW2
    // Filter out: Loopback (Type == 24), down interfaces (OperStatus != 1)
    // Delta calculation for rates
    // FreeMibTable when done
    return nil, nil
}
```

---

## 3. Controller Engine (`internal/controller/`)

### 3.1 Killer (`killer.go`)

```go
type Killer struct {
    protectedList map[string]bool // exe names that cannot be killed
}

func (k *Killer) Kill(pid uint32) error {
    // 1. Safety check
    if pid == 0 || pid == 4 {
        return ErrProtectedProcess
    }
    
    // 2. Get process name for protection check
    name, _ := getProcessName(pid)
    if k.protectedList[strings.ToLower(name)] {
        return fmt.Errorf("%w: %s", ErrProtectedProcess, name)
    }
    
    // 3. Check if critical
    handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE, false, pid)
    if err != nil {
        return fmt.Errorf("open process %d: %w", pid, err)
    }
    defer windows.CloseHandle(handle)
    
    critical, _ := winapi.IsProcessCritical(handle)
    if critical {
        return fmt.Errorf("%w: PID %d is critical", ErrProtectedProcess, pid)
    }
    
    // 4. Terminate
    if err := windows.TerminateProcess(handle, 1); err != nil {
        return fmt.Errorf("terminate %d: %w", pid, err)
    }
    return nil
}

func (k *Killer) KillTree(pid uint32, tree []*ProcessNode) error {
    // Find node in tree, collect all descendant PIDs
    node := findNode(tree, pid)
    if node == nil {
        return k.Kill(pid) // no tree info, just kill the process
    }
    
    // Kill bottom-up: children first, then parent
    var killOrder []uint32
    var collect func(n *ProcessNode)
    collect = func(n *ProcessNode) {
        for _, child := range n.Children {
            collect(child)
        }
        killOrder = append(killOrder, n.Process.PID)
    }
    collect(node)
    
    var errs []error
    for _, childPID := range killOrder {
        if err := k.Kill(childPID); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)
}
```

### 3.2 Limiter via Job Objects (`limiter.go`)

```go
type Limiter struct {
    jobs map[uint32]*jobEntry // PID → active Job Object
    mu   sync.Mutex
}

type jobEntry struct {
    handle    windows.Handle
    pid       uint32
    cpuLimit  uint32 // percentage (0 = not set)
    memLimit  uint64 // bytes (0 = not set)
    procLimit uint32 // max child processes (0 = not set)
}

func (l *Limiter) SetCPULimit(pid uint32, percent uint32) error {
    if percent == 0 || percent > 100 {
        return fmt.Errorf("cpu limit must be 1-100, got %d", percent)
    }
    
    entry, err := l.getOrCreateJob(pid)
    if err != nil {
        return err
    }
    
    info := winapi.JOBOBJECT_CPU_RATE_CONTROL_INFORMATION{
        ControlFlags: winapi.JOB_OBJECT_CPU_RATE_CONTROL_ENABLE | winapi.JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP,
        CpuRate:      percent * 100, // API wants basis points (30% = 3000)
    }
    
    if err := winapi.SetInformationJobObject(
        entry.handle,
        winapi.JobObjectCpuRateControlInformation,
        unsafe.Pointer(&info),
        uint32(unsafe.Sizeof(info)),
    ); err != nil {
        return fmt.Errorf("set cpu rate: %w", err)
    }
    
    entry.cpuLimit = percent
    return nil
}

func (l *Limiter) SetMemoryLimit(pid uint32, maxBytes uint64) error {
    entry, err := l.getOrCreateJob(pid)
    if err != nil {
        return err
    }
    
    info := winapi.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
    info.BasicLimitInformation.LimitFlags = winapi.JOB_OBJECT_LIMIT_PROCESS_MEMORY
    info.ProcessMemoryLimit = uintptr(maxBytes)
    
    if err := winapi.SetInformationJobObject(
        entry.handle,
        winapi.JobObjectExtendedLimitInformation,
        unsafe.Pointer(&info),
        uint32(unsafe.Sizeof(info)),
    ); err != nil {
        return fmt.Errorf("set memory limit: %w", err)
    }
    
    entry.memLimit = maxBytes
    return nil
}

func (l *Limiter) SetProcessLimit(pid uint32, maxChildren uint32) error {
    entry, err := l.getOrCreateJob(pid)
    if err != nil {
        return err
    }
    
    info := winapi.JOBOBJECT_BASIC_LIMIT_INFORMATION{
        LimitFlags:         winapi.JOB_OBJECT_LIMIT_ACTIVE_PROCESS,
        ActiveProcessLimit: maxChildren,
    }
    
    // Wrap in EXTENDED to set via JobObjectExtendedLimitInformation
    extInfo := winapi.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
        BasicLimitInformation: info,
    }
    
    if err := winapi.SetInformationJobObject(
        entry.handle,
        winapi.JobObjectExtendedLimitInformation,
        unsafe.Pointer(&extInfo),
        uint32(unsafe.Sizeof(extInfo)),
    ); err != nil {
        return fmt.Errorf("set process limit: %w", err)
    }
    
    entry.procLimit = maxChildren
    return nil
}

func (l *Limiter) getOrCreateJob(pid uint32) (*jobEntry, error) {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    if entry, ok := l.jobs[pid]; ok {
        return entry, nil
    }
    
    // Create Job Object
    jobHandle, err := winapi.CreateJobObjectW(nil, nil)
    if err != nil {
        return nil, fmt.Errorf("create job: %w", err)
    }
    
    // Open target process
    procHandle, err := windows.OpenProcess(
        windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
        false, pid,
    )
    if err != nil {
        windows.CloseHandle(jobHandle)
        return nil, fmt.Errorf("open process %d: %w", pid, err)
    }
    defer windows.CloseHandle(procHandle)
    
    // Assign process to job
    if err := winapi.AssignProcessToJobObject(jobHandle, procHandle); err != nil {
        windows.CloseHandle(jobHandle)
        return nil, fmt.Errorf("assign to job: %w", err)
    }
    
    entry := &jobEntry{handle: jobHandle, pid: pid}
    l.jobs[pid] = entry
    return entry, nil
}

func (l *Limiter) RemoveLimits(pid uint32) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    entry, ok := l.jobs[pid]
    if !ok {
        return nil
    }
    
    windows.CloseHandle(entry.handle) // closing job handle releases all limits
    delete(l.jobs, pid)
    return nil
}
```

### 3.3 Suspender (`suspender.go`)

```go
func Suspend(pid uint32) error {
    return forEachThread(pid, func(threadID uint32) error {
        handle, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, threadID)
        if err != nil {
            return err
        }
        defer windows.CloseHandle(handle)
        _, err = winapi.SuspendThread(handle)
        return err
    })
}

func Resume(pid uint32) error {
    return forEachThread(pid, func(threadID uint32) error {
        handle, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, threadID)
        if err != nil {
            return err
        }
        defer windows.CloseHandle(handle)
        _, err = winapi.ResumeThread(handle)
        return err
    })
}

func forEachThread(pid uint32, fn func(threadID uint32) error) error {
    snap, err := winapi.CreateToolhelp32Snapshot(winapi.TH32CS_SNAPTHREAD, 0)
    if err != nil {
        return err
    }
    defer windows.CloseHandle(snap)
    
    var te winapi.THREADENTRY32
    te.Size = uint32(unsafe.Sizeof(te))
    
    var errs []error
    for err := winapi.Thread32First(snap, &te); err == nil; err = winapi.Thread32Next(snap, &te) {
        if te.OwnerProcessID == pid {
            if err := fn(te.ThreadID); err != nil {
                errs = append(errs, err)
            }
        }
    }
    return errors.Join(errs...)
}
```

---

## 4. Anomaly Detection Engine (`internal/anomaly/`)

### 4.1 Engine Orchestrator (`engine.go`)

```go
type Engine struct {
    config    *config.AnomalyConfig
    store     *storage.Store
    emitter   *event.Emitter
    
    detectors []Detector
    alerts    *AlertManager
}

// Detector interface — all detectors implement this
type Detector interface {
    Name() string
    Detect(snap *SystemSnapshot) []Alert
    Reset()
}

func (e *Engine) Start(ctx context.Context) {
    // Initialize detectors based on config
    e.detectors = []Detector{
        NewSpawnStormDetector(e.config.SpawnStorm),
        NewMemLeakDetector(e.config.MemoryLeak),
        NewHungDetector(e.config.HungProcess),
        NewOrphanDetector(e.config.Orphan),
        NewRunawayDetector(e.config.RunawayCPU),
        NewPortConflictDetector(e.config.PortConflict),
        NewNetAnomalyDetector(e.config.NetworkAnomaly),
        NewNewProcessDetector(e.config.NewProcess),
    }
    
    ticker := time.NewTicker(e.config.AnalysisInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            snap := e.store.LatestSnapshot()
            if snap == nil {
                continue
            }
            
            for _, d := range e.detectors {
                alerts := d.Detect(snap)
                for _, alert := range alerts {
                    if e.alerts.ShouldEmit(alert) {
                        e.alerts.Add(alert)
                        e.emitter.Emit("alert_new", alert)
                    }
                }
            }
            
            // Check for resolved alerts
            resolved := e.alerts.CheckResolved(snap)
            for _, alert := range resolved {
                e.emitter.Emit("alert_resolved", alert)
            }
        }
    }
}
```

### 4.2 Alert Manager (`alert.go`)

```go
type Alert struct {
    ID          string    `json:"id"`          // unique alert ID
    Type        string    `json:"type"`        // spawn_storm, mem_leak, hung, etc.
    Severity    Severity  `json:"severity"`    // info, warning, critical, emergency
    Title       string    `json:"title"`
    Description string    `json:"description"`
    PID         uint32    `json:"pid,omitempty"`
    Process     string    `json:"process,omitempty"`
    Data        any       `json:"data,omitempty"` // detector-specific data
    CreatedAt   time.Time `json:"created_at"`
    ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
    Dismissed   bool      `json:"dismissed"`
    SnoozedUntil *time.Time `json:"snoozed_until,omitempty"`
}

type Severity int

const (
    SeverityInfo Severity = iota
    SeverityWarning
    SeverityCritical
    SeverityEmergency
)

type AlertManager struct {
    active   map[string]*Alert
    history  []*Alert
    mu       sync.RWMutex
    maxHistory int
}

// ShouldEmit prevents duplicate alerts for same issue
func (am *AlertManager) ShouldEmit(alert Alert) bool {
    am.mu.RLock()
    defer am.mu.RUnlock()
    
    // Dedup key: type + PID
    key := fmt.Sprintf("%s:%d", alert.Type, alert.PID)
    existing, ok := am.active[key]
    if !ok {
        return true
    }
    
    // Don't re-emit if same severity and within dedup window
    if existing.Severity == alert.Severity && time.Since(existing.CreatedAt) < 5*time.Minute {
        return false
    }
    
    // Re-emit if severity escalated
    return alert.Severity > existing.Severity
}
```

### 4.3 Spawn Storm Detector (`spawnstorm.go`)

```go
type SpawnStormDetector struct {
    config         config.SpawnStormConfig
    prevPIDs       map[uint32]bool
    spawnHistory   map[uint32][]time.Time // parent PID → spawn timestamps
    mu             sync.Mutex
}

func (d *SpawnStormDetector) Detect(snap *SystemSnapshot) []Alert {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    currentPIDs := make(map[uint32]bool)
    parentChildCount := make(map[uint32]int)    // parent PID → total child count
    parentChildMem := make(map[uint32]uint64)   // parent PID → total child memory
    
    for _, p := range snap.Processes {
        currentPIDs[p.PID] = true
        parentChildCount[p.ParentPID]++
        parentChildMem[p.ParentPID] += p.WorkingSet
        
        // Detect new PIDs (spawned since last check)
        if d.prevPIDs != nil && !d.prevPIDs[p.PID] {
            // New process appeared
            d.spawnHistory[p.ParentPID] = append(d.spawnHistory[p.ParentPID], time.Now())
        }
    }
    
    var alerts []Alert
    now := time.Now()
    
    for parentPID, timestamps := range d.spawnHistory {
        // Count spawns in last 60 seconds
        recentCount := 0
        for _, t := range timestamps {
            if now.Sub(t) < time.Minute {
                recentCount++
            }
        }
        
        // Clean old timestamps
        var recent []time.Time
        for _, t := range timestamps {
            if now.Sub(t) < 5*time.Minute {
                recent = append(recent, t)
            }
        }
        d.spawnHistory[parentPID] = recent
        
        totalChildren := parentChildCount[parentPID]
        totalMem := parentChildMem[parentPID]
        parentName := snap.ProcessName(parentPID)
        
        if recentCount > d.config.MaxChildrenPerMinute {
            severity := SeverityCritical
            if totalChildren > d.config.MaxTotalChildren {
                severity = SeverityEmergency
            }
            
            alerts = append(alerts, Alert{
                ID:       fmt.Sprintf("spawn_storm_%d_%d", parentPID, now.Unix()),
                Type:     "spawn_storm",
                Severity: severity,
                Title:    fmt.Sprintf("Spawn Storm: %s", parentName),
                Description: fmt.Sprintf(
                    "%s (PID %d) spawned %d children in last 60s. Total: %d children, %s memory",
                    parentName, parentPID, recentCount, totalChildren, formatBytes(totalMem),
                ),
                PID:     parentPID,
                Process: parentName,
                Data: map[string]any{
                    "recent_spawns":  recentCount,
                    "total_children": totalChildren,
                    "total_memory":   totalMem,
                    "spawn_rate":     float64(recentCount) / 60.0, // per second
                },
            })
        }
    }
    
    d.prevPIDs = currentPIDs
    return alerts
}
```

### 4.4 Memory Leak Detector (`memleak.go`)

```go
type MemLeakDetector struct {
    config   config.MemLeakConfig
    trackers map[uint32]*memTracker // PID → tracker
    mu       sync.Mutex
}

type memTracker struct {
    welford  *stats.Welford
    samples  *stats.SlidingWindow[memSample] // time + value
    firstMem uint64
}

type memSample struct {
    time  time.Time
    value uint64
}

func (d *MemLeakDetector) Detect(snap *SystemSnapshot) []Alert {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    var alerts []Alert
    
    for _, p := range snap.Processes {
        tracker, ok := d.trackers[p.PID]
        if !ok {
            tracker = &memTracker{
                welford:  stats.NewWelford(),
                samples:  stats.NewSlidingWindow[memSample](300), // ~5 min at 1s interval
                firstMem: p.WorkingSet,
            }
            d.trackers[p.PID] = tracker
        }
        
        tracker.welford.Add(float64(p.WorkingSet))
        tracker.samples.Add(memSample{time: time.Now(), value: p.WorkingSet})
        
        // Need minimum samples for regression
        if tracker.samples.Len() < 30 {
            continue
        }
        
        // Linear regression on memory samples
        samples := tracker.samples.Slice()
        xs := make([]float64, len(samples))
        ys := make([]float64, len(samples))
        t0 := samples[0].time
        for i, s := range samples {
            xs[i] = s.time.Sub(t0).Seconds()
            ys[i] = float64(s.value)
        }
        
        slope, _, rSquared := stats.LinearRegression(xs, ys)
        
        // slope is bytes/second, convert to bytes/minute
        growthPerMin := slope * 60.0
        
        if growthPerMin > float64(d.config.MinGrowthRate) && rSquared > d.config.MinRSquared {
            severity := SeverityWarning
            if p.WorkingSet > d.config.MemoryThreshold {
                severity = SeverityCritical
            }
            
            // Project when it'll hit threshold
            remaining := float64(d.config.MemoryThreshold) - float64(p.WorkingSet)
            minutesToThreshold := remaining / growthPerMin
            
            alerts = append(alerts, Alert{
                ID:       fmt.Sprintf("mem_leak_%d", p.PID),
                Type:     "memory_leak",
                Severity: severity,
                Title:    fmt.Sprintf("Memory Leak: %s", p.Name),
                Description: fmt.Sprintf(
                    "%s (PID %d) memory growing at %s/min (R²=%.2f). Current: %s, started at %s. Projected to reach %s in %.0f minutes.",
                    p.Name, p.PID,
                    formatBytes(uint64(growthPerMin)), rSquared,
                    formatBytes(p.WorkingSet), formatBytes(tracker.firstMem),
                    formatBytes(d.config.MemoryThreshold), minutesToThreshold,
                ),
                PID:     p.PID,
                Process: p.Name,
                Data: map[string]any{
                    "growth_per_min_bytes": growthPerMin,
                    "r_squared":           rSquared,
                    "current_memory":      p.WorkingSet,
                    "start_memory":        tracker.firstMem,
                    "minutes_to_limit":    minutesToThreshold,
                },
            })
        }
    }
    
    return alerts
}
```

### 4.5 Port Conflict Detector (`portconflict.go`)

```go
type PortConflictDetector struct {
    config       config.PortConflictConfig
    staleStates  map[string]time.Time // "proto:port:pid:state" → first seen
    prevPortPIDs map[string]uint32    // "port" → last known PID (for hijack detection)
}

func (d *PortConflictDetector) Detect(snap *SystemSnapshot) []Alert {
    var alerts []Alert
    
    // Group bindings by local port
    portGroups := make(map[uint16][]PortBinding)
    for _, b := range snap.PortBindings {
        portGroups[b.LocalPort] = append(portGroups[b.LocalPort], b)
    }
    
    for port, bindings := range portGroups {
        // 1. Multiple PIDs on same port (conflict)
        pids := uniquePIDs(bindings)
        if len(pids) > 1 {
            alerts = append(alerts, Alert{
                Type:     "port_conflict",
                Severity: SeverityWarning,
                Title:    fmt.Sprintf("Port Conflict: :%d", port),
                Description: fmt.Sprintf(
                    "Port :%d held by %d different processes: %s",
                    port, len(pids), describePIDs(pids, bindings),
                ),
                Data: map[string]any{"port": port, "bindings": bindings},
            })
        }
        
        // 2. Stale TIME_WAIT / CLOSE_WAIT
        for _, b := range bindings {
            if b.StateCode == winapi.MIB_TCP_STATE_TIME_WAIT || b.StateCode == winapi.MIB_TCP_STATE_CLOSE_WAIT {
                key := fmt.Sprintf("%s:%d:%d:%s", b.Protocol, b.LocalPort, b.PID, b.State)
                firstSeen, tracked := d.staleStates[key]
                if !tracked {
                    d.staleStates[key] = time.Now()
                    continue
                }
                
                threshold := d.config.TimeWaitThreshold
                if b.StateCode == winapi.MIB_TCP_STATE_CLOSE_WAIT {
                    threshold = d.config.CloseWaitThreshold
                }
                
                if time.Since(firstSeen) > threshold {
                    alerts = append(alerts, Alert{
                        Type:     "port_zombie",
                        Severity: SeverityWarning,
                        Title:    fmt.Sprintf("Zombie Port: :%d (%s)", port, b.State),
                        Description: fmt.Sprintf(
                            "Port :%d held by %s (PID %d) in %s state for %s",
                            port, b.Process, b.PID, b.State,
                            time.Since(firstSeen).Round(time.Second),
                        ),
                        PID:     b.PID,
                        Process: b.Process,
                    })
                }
            }
        }
        
        // 3. Port hijack detection
        for _, b := range bindings {
            if b.StateCode == winapi.MIB_TCP_STATE_LISTEN {
                portKey := fmt.Sprintf("%d", b.LocalPort)
                prevPID, known := d.prevPortPIDs[portKey]
                if known && prevPID != b.PID {
                    alerts = append(alerts, Alert{
                        Type:     "port_hijack",
                        Severity: SeverityWarning,
                        Title:    fmt.Sprintf("Port Ownership Changed: :%d", port),
                        Description: fmt.Sprintf(
                            "Port :%d was held by PID %d, now held by %s (PID %d)",
                            port, prevPID, b.Process, b.PID,
                        ),
                    })
                }
                d.prevPortPIDs[portKey] = b.PID
            }
        }
    }
    
    return alerts
}
```

---

## 5. Statistical Utilities (`internal/stats/`)

### 5.1 Welford's Online Algorithm (`welford.go`)

```go
// Welford computes running mean, variance, and stddev in O(1) per update
type Welford struct {
    count    uint64
    mean     float64
    m2       float64 // sum of squared differences from mean
}

func NewWelford() *Welford { return &Welford{} }

func (w *Welford) Add(value float64) {
    w.count++
    delta := value - w.mean
    w.mean += delta / float64(w.count)
    delta2 := value - w.mean
    w.m2 += delta * delta2
}

func (w *Welford) Mean() float64 { return w.mean }

func (w *Welford) Variance() float64 {
    if w.count < 2 {
        return 0
    }
    return w.m2 / float64(w.count-1)
}

func (w *Welford) StdDev() float64 {
    return math.Sqrt(w.Variance())
}

func (w *Welford) Count() uint64 { return w.count }

// IsAnomaly returns true if value is more than nSigma standard deviations from mean
func (w *Welford) IsAnomaly(value float64, nSigma float64) bool {
    if w.count < 10 { // need minimum samples
        return false
    }
    return math.Abs(value-w.mean) > nSigma*w.StdDev()
}
```

### 5.2 Linear Regression (`regression.go`)

```go
// LinearRegression computes slope, intercept, and R² using least squares
// Returns: slope (dy/dx), intercept, rSquared
func LinearRegression(xs, ys []float64) (slope, intercept, rSquared float64) {
    n := float64(len(xs))
    if n < 2 {
        return 0, 0, 0
    }
    
    var sumX, sumY, sumXY, sumX2, sumY2 float64
    for i := range xs {
        sumX += xs[i]
        sumY += ys[i]
        sumXY += xs[i] * ys[i]
        sumX2 += xs[i] * xs[i]
        sumY2 += ys[i] * ys[i]
    }
    
    denom := n*sumX2 - sumX*sumX
    if denom == 0 {
        return 0, sumY / n, 0
    }
    
    slope = (n*sumXY - sumX*sumY) / denom
    intercept = (sumY - slope*sumX) / n
    
    // R² = (n*sumXY - sumX*sumY)² / ((n*sumX2 - sumX²) * (n*sumY2 - sumY²))
    ssRes := 0.0
    ssTot := 0.0
    meanY := sumY / n
    for i := range xs {
        predicted := slope*xs[i] + intercept
        ssRes += (ys[i] - predicted) * (ys[i] - predicted)
        ssTot += (ys[i] - meanY) * (ys[i] - meanY)
    }
    
    if ssTot == 0 {
        rSquared = 1 // all values identical
    } else {
        rSquared = 1 - ssRes/ssTot
    }
    
    return slope, intercept, rSquared
}
```

### 5.3 Ring Buffer (`ringbuf.go`)

```go
// RingBuffer is a fixed-capacity circular buffer
// Single writer, multiple readers via atomic index
type RingBuffer[T any] struct {
    data     []T
    capacity int
    head     atomic.Int64 // write position
    count    atomic.Int64
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
    return &RingBuffer[T]{
        data:     make([]T, capacity),
        capacity: capacity,
    }
}

func (rb *RingBuffer[T]) Add(item T) {
    pos := rb.head.Add(1) - 1
    rb.data[pos%int64(rb.capacity)] = item
    
    count := rb.count.Load()
    if count < int64(rb.capacity) {
        rb.count.Add(1)
    }
}

func (rb *RingBuffer[T]) Slice() []T {
    count := int(rb.count.Load())
    head := int(rb.head.Load())
    
    result := make([]T, count)
    for i := 0; i < count; i++ {
        idx := (head - count + i) % rb.capacity
        if idx < 0 {
            idx += rb.capacity
        }
        result[i] = rb.data[idx]
    }
    return result
}

func (rb *RingBuffer[T]) Len() int {
    return int(rb.count.Load())
}

func (rb *RingBuffer[T]) Last() (T, bool) {
    if rb.count.Load() == 0 {
        var zero T
        return zero, false
    }
    pos := (rb.head.Load() - 1) % int64(rb.capacity)
    return rb.data[pos], true
}
```

### 5.4 Sliding Window (`ringbuf.go` — same file)

```go
// SlidingWindow wraps RingBuffer with time-based or count-based eviction
type SlidingWindow[T any] struct {
    buf *RingBuffer[T]
}

func NewSlidingWindow[T any](maxSize int) *SlidingWindow[T] {
    return &SlidingWindow[T]{buf: NewRingBuffer[T](maxSize)}
}

func (sw *SlidingWindow[T]) Add(item T) { sw.buf.Add(item) }
func (sw *SlidingWindow[T]) Slice() []T { return sw.buf.Slice() }
func (sw *SlidingWindow[T]) Len() int   { return sw.buf.Len() }
```

---

## 6. AI Advisor (`internal/ai/`)

### 6.1 Advisor Client (`advisor.go`)

```go
type Advisor struct {
    config   *config.AIConfig
    client   *http.Client
    cache    *ResponseCache
    limiter  *RateLimiter
}

type AnalysisRequest struct {
    AlertID  string // optional: analyze specific alert
    Question string // optional: user question
    Snapshot *SystemSnapshot
}

type AnalysisResponse struct {
    Text        string   `json:"text"`
    Suggestions []Action `json:"suggestions,omitempty"`
    Cached      bool     `json:"cached"`
}

type Action struct {
    Type   string `json:"type"`   // kill, suspend, limit_cpu, limit_mem, create_rule
    Label  string `json:"label"`
    PID    uint32 `json:"pid,omitempty"`
    Params any    `json:"params,omitempty"`
}

func (a *Advisor) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
    if !a.config.Enabled || a.config.APIKey == "" {
        return nil, ErrAINotConfigured
    }
    
    // Check rate limit
    if !a.limiter.Allow() {
        return nil, ErrRateLimited
    }
    
    // Build prompt
    prompt := buildPrompt(req, a.config)
    
    // Check cache
    cacheKey := cacheKeyFor(req)
    if cached, ok := a.cache.Get(cacheKey); ok {
        cached.Cached = true
        return cached, nil
    }
    
    // Call API
    body := map[string]any{
        "model":      a.config.Model,
        "max_tokens": a.config.MaxTokens,
        "system":     systemPrompt(a.config.Language),
        "messages": []map[string]any{
            {"role": "user", "content": prompt},
        },
    }
    
    jsonBody, _ := json.Marshal(body)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", a.config.Endpoint, bytes.NewReader(jsonBody))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("x-api-key", a.config.APIKey)
    httpReq.Header.Set("anthropic-version", "2023-06-01")
    httpReq.Header.Set("content-type", "application/json")
    
    resp, err := a.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("api call: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 429 {
        return nil, ErrRateLimited
    }
    if resp.StatusCode == 401 {
        return nil, ErrInvalidAPIKey
    }
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(body))
    }
    
    var apiResp struct {
        Content []struct {
            Type string `json:"type"`
            Text string `json:"text"`
        } `json:"content"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    text := ""
    for _, c := range apiResp.Content {
        if c.Type == "text" {
            text += c.Text
        }
    }
    
    result := &AnalysisResponse{
        Text:        text,
        Suggestions: parseSuggestions(text),
    }
    
    a.cache.Set(cacheKey, result, 5*time.Minute)
    return result, nil
}
```

### 6.2 Prompt Builder (`prompt.go`)

```go
func systemPrompt(language string) string {
    lang := "English"
    if language == "tr" {
        lang = "Turkish"
    }
    return fmt.Sprintf(`You are a Windows system diagnostics expert integrated into WindowsTaskManager.

Your role:
1. Analyze system state and anomalies
2. Identify root causes of performance issues
3. Provide actionable, specific recommendations
4. Suggest concrete commands or configuration changes when applicable

Rules:
- Respond in %s
- Be concise and practical — no generic advice
- When you identify a specific fix, state it clearly
- Reference specific PIDs and process names from the data
- If you recommend killing processes, mention the expected resource recovery
- For development tools (vitest, webpack, node, etc.), suggest configuration fixes too`, lang)
}

func buildPrompt(req *AnalysisRequest, cfg *config.AIConfig) string {
    var b strings.Builder
    snap := req.Snapshot
    
    // System overview
    fmt.Fprintf(&b, "=== SYSTEM STATE ===\n")
    fmt.Fprintf(&b, "CPU: %.1f%% (%d logical cores)\n", snap.CPU.TotalPercent, snap.CPU.NumLogical)
    fmt.Fprintf(&b, "Memory: %s / %s (%.1f%%)\n",
        formatBytes(snap.Memory.UsedPhys), formatBytes(snap.Memory.TotalPhys), snap.Memory.UsedPercent)
    if snap.GPU.Available {
        fmt.Fprintf(&b, "GPU: %s — %.1f%%, VRAM %s / %s\n",
            snap.GPU.Name, snap.GPU.Utilization,
            formatBytes(snap.GPU.VRAMUsed), formatBytes(snap.GPU.VRAMTotal))
    }
    
    // Active alerts
    if len(snap.ActiveAlerts) > 0 {
        fmt.Fprintf(&b, "\n=== ACTIVE ALERTS ===\n")
        for _, a := range snap.ActiveAlerts {
            fmt.Fprintf(&b, "[%s] %s: %s\n", a.Severity, a.Type, a.Description)
        }
    }
    
    // Top processes (top 20 by CPU + top 20 by memory, deduplicated)
    if cfg.IncludeProcessTree {
        fmt.Fprintf(&b, "\n=== TOP PROCESSES ===\n")
        top := topProcesses(snap.Processes, 20)
        for _, p := range top {
            fmt.Fprintf(&b, "PID %d | %s | CPU %.1f%% | MEM %s | IO R:%s/s W:%s/s | Threads:%d\n",
                p.PID, p.Name, p.CPUPercent, formatBytes(p.WorkingSet),
                formatBytes(p.IOReadBytes), formatBytes(p.IOWriteBytes), p.ThreadCount)
        }
    }
    
    // Port map (listeners + conflicts only)
    if cfg.IncludePortMap {
        listeners := filterListeners(snap.PortBindings)
        if len(listeners) > 0 {
            fmt.Fprintf(&b, "\n=== LISTENING PORTS ===\n")
            for _, p := range listeners {
                label := ""
                if p.Label != "" {
                    label = fmt.Sprintf(" (%s)", p.Label)
                }
                fmt.Fprintf(&b, ":%d → %s (PID %d) [%s]%s\n",
                    p.LocalPort, p.Process, p.PID, p.State, label)
            }
        }
    }
    
    // Specific alert context (if analyzing one alert)
    if req.AlertID != "" {
        alert := findAlert(snap.ActiveAlerts, req.AlertID)
        if alert != nil {
            fmt.Fprintf(&b, "\n=== FOCUS ALERT ===\n")
            fmt.Fprintf(&b, "%s\n", alert.Description)
            if alert.Data != nil {
                dataJSON, _ := json.MarshalIndent(alert.Data, "", "  ")
                fmt.Fprintf(&b, "Data: %s\n", dataJSON)
            }
        }
    }
    
    // User question
    if req.Question != "" {
        fmt.Fprintf(&b, "\n=== USER QUESTION ===\n")
        fmt.Fprintf(&b, "%s\n", req.Question)
    }
    
    return b.String()
}
```

---

## 7. HTTP Server (`internal/server/`)

### 7.1 Router (`router.go`)

Custom router — no third-party dependency (no gorilla/mux, no chi):

```go
type Router struct {
    routes []route
    notFound http.Handler
}

type route struct {
    method  string
    pattern string // e.g. "/api/v1/processes/{pid}/kill"
    handler http.HandlerFunc
    parts   []routePart
}

type routePart struct {
    value    string
    isParam  bool
}

func (r *Router) Handle(method, pattern string, handler http.HandlerFunc) {
    parts := parsePattern(pattern) // split by /, detect {param}
    r.routes = append(r.routes, route{method: method, pattern: pattern, handler: handler, parts: parts})
}

func (r *Router) GET(pattern string, h http.HandlerFunc)  { r.Handle("GET", pattern, h) }
func (r *Router) POST(pattern string, h http.HandlerFunc) { r.Handle("POST", pattern, h) }
func (r *Router) PUT(pattern string, h http.HandlerFunc)  { r.Handle("PUT", pattern, h) }
func (r *Router) DELETE(pattern string, h http.HandlerFunc) { r.Handle("DELETE", pattern, h) }

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    for _, route := range r.routes {
        if req.Method != route.method {
            continue
        }
        params, ok := match(route.parts, req.URL.Path)
        if ok {
            ctx := context.WithValue(req.Context(), paramsKey, params)
            route.handler(w, req.WithContext(ctx))
            return
        }
    }
    if r.notFound != nil {
        r.notFound.ServeHTTP(w, req)
    } else {
        http.NotFound(w, req)
    }
}

// Param extracts a URL parameter from the request context
func Param(r *http.Request, name string) string {
    params := r.Context().Value(paramsKey).(map[string]string)
    return params[name]
}
```

### 7.2 SSE Handler (`sse.go`)

```go
type SSEHub struct {
    clients map[*sseClient]bool
    mu      sync.RWMutex
}

type sseClient struct {
    ch     chan sseEvent
    types  map[string]bool // filter: which event types this client wants
    done   chan struct{}
}

type sseEvent struct {
    Type string `json:"type"`
    Data any    `json:"data"`
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming unsupported", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Parse type filter from query
    typeFilter := parseTypeFilter(r.URL.Query().Get("types"))
    
    client := &sseClient{
        ch:    make(chan sseEvent, 64), // buffered to avoid blocking
        types: typeFilter,
        done:  make(chan struct{}),
    }
    
    h.mu.Lock()
    h.clients[client] = true
    h.mu.Unlock()
    
    defer func() {
        h.mu.Lock()
        delete(h.clients, client)
        h.mu.Unlock()
        close(client.done)
    }()
    
    for {
        select {
        case <-r.Context().Done():
            return
        case event := <-client.ch:
            data, _ := json.Marshal(event.Data)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
            flusher.Flush()
        }
    }
}

func (h *SSEHub) Broadcast(eventType string, data any) {
    event := sseEvent{Type: eventType, Data: data}
    
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    for client := range h.clients {
        // Check type filter
        if len(client.types) > 0 && !client.types[eventType] {
            continue
        }
        // Non-blocking send (drop if client is slow)
        select {
        case client.ch <- event:
        default:
            // Client too slow, drop event
        }
    }
}
```

### 7.3 API Handlers (Example: `api_process.go`)

```go
func (s *Server) handleListProcesses(w http.ResponseWriter, r *http.Request) {
    processes := s.collector.Processes()
    
    // Query params
    sortBy := r.URL.Query().Get("sort")   // cpu, memory, name, pid, io, net
    order := r.URL.Query().Get("order")   // asc, desc
    search := r.URL.Query().Get("search") // name filter
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    
    // Filter
    if search != "" {
        processes = filterByName(processes, search)
    }
    
    // Sort
    sortProcesses(processes, sortBy, order)
    
    // Limit
    if limit > 0 && limit < len(processes) {
        processes = processes[:limit]
    }
    
    writeJSON(w, processes)
}

func (s *Server) handleKillProcess(w http.ResponseWriter, r *http.Request) {
    pid, err := strconv.ParseUint(Param(r, "pid"), 10, 32)
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid PID")
        return
    }
    
    if err := s.controller.Kill(uint32(pid)); err != nil {
        if errors.Is(err, controller.ErrProtectedProcess) {
            writeError(w, http.StatusForbidden, err.Error())
            return
        }
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    writeJSON(w, map[string]string{"status": "killed", "pid": fmt.Sprintf("%d", pid)})
}

func (s *Server) handleSetCPULimit(w http.ResponseWriter, r *http.Request) {
    pid, err := strconv.ParseUint(Param(r, "pid"), 10, 32)
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid PID")
        return
    }
    
    var body struct {
        Percent uint32 `json:"percent"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeError(w, http.StatusBadRequest, "invalid body")
        return
    }
    
    if err := s.controller.SetCPULimit(uint32(pid), body.Percent); err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    writeJSON(w, map[string]any{"status": "limited", "pid": pid, "cpu_percent": body.Percent})
}
```

---

## 8. System Tray (`internal/tray/`)

### 8.1 Implementation Strategy

```go
type Tray struct {
    hwnd       uintptr // hidden message window
    nid        winapi.NOTIFYICONDATAW
    icon       uintptr // HICON
    menu       uintptr // HMENU
    
    onOpen     func()
    onPause    func()
    onSettings func()
    onExit     func()
    
    metrics    *SystemSnapshot // for tooltip + menu display
    mu         sync.RWMutex
}

func (t *Tray) Start(ctx context.Context) error {
    // 1. Register window class
    // 2. Create hidden message window (HWND_MESSAGE)
    // 3. Create initial icon (default app icon from resource)
    // 4. Shell_NotifyIconW(NIM_ADD, &nid)
    // 5. Start message pump goroutine
    
    // Message pump runs on its own OS thread (required by Windows)
    runtime.LockOSThread()
    
    var msg winapi.MSG
    for {
        ret := winapi.GetMessage(&msg, 0, 0, 0)
        if ret == 0 || ret == -1 {
            break
        }
        winapi.TranslateMessage(&msg)
        winapi.DispatchMessage(&msg)
    }
    return nil
}

// wndProc handles tray messages
func (t *Tray) wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
    switch msg {
    case WM_TRAYICON: // custom message for tray events
        switch lParam {
        case WM_LBUTTONDBLCLK:
            t.onOpen()
        case WM_RBUTTONUP:
            t.showContextMenu()
        }
    case WM_COMMAND:
        t.handleMenuCommand(wParam)
    }
    return winapi.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (t *Tray) ShowBalloon(title, text string, icon uint32) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    copy(t.nid.InfoTitle[:], windows.StringToUTF16(title))
    copy(t.nid.Info[:], windows.StringToUTF16(text))
    t.nid.Flags |= winapi.NIF_INFO
    t.nid.InfoFlags = icon // NIIF_WARNING, NIIF_ERROR, etc.
    
    winapi.ShellNotifyIconW(winapi.NIM_MODIFY, &t.nid)
}

func (t *Tray) UpdateTooltip(cpu, mem float64) {
    tip := fmt.Sprintf("WindowsTaskManager — CPU: %.0f%% | MEM: %.0f%%", cpu, mem)
    copy(t.nid.Tip[:], windows.StringToUTF16(tip))
    t.nid.Flags = winapi.NIF_TIP
    winapi.ShellNotifyIconW(winapi.NIM_MODIFY, &t.nid)
}
```

---

## 9. Configuration (`internal/config/`)

### 9.1 Config Loader (`loader.go`)

```go
func Load(path string) (*Config, error) {
    // 1. If file doesn't exist, create with defaults
    if _, err := os.Stat(path); os.IsNotExist(err) {
        cfg := DefaultConfig()
        if err := Save(path, cfg); err != nil {
            return nil, fmt.Errorf("create default config: %w", err)
        }
        return cfg, nil
    }
    
    // 2. Read and parse YAML
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    
    cfg := DefaultConfig() // start with defaults, overlay from file
    if err := yaml.Unmarshal(data, cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    
    // 3. Validate
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("validate config: %w", err)
    }
    
    return cfg, nil
}
```

### 9.2 File Watcher (`watcher.go`)

```go
// Native Windows file watcher using ReadDirectoryChangesW
// No fsnotify dependency
type Watcher struct {
    dir      string
    filename string
    onChange func()
}

func (w *Watcher) Start(ctx context.Context) error {
    dirHandle, err := windows.CreateFile(
        windows.StringToUTF16Ptr(w.dir),
        windows.FILE_LIST_DIRECTORY,
        windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
        nil,
        windows.OPEN_EXISTING,
        windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OVERLAPPED,
        0,
    )
    if err != nil {
        return err
    }
    defer windows.CloseHandle(dirHandle)
    
    buf := make([]byte, 4096)
    
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
        }
        
        var bytesReturned uint32
        err := windows.ReadDirectoryChanges(
            dirHandle,
            &buf[0],
            uint32(len(buf)),
            false, // don't watch subtree
            windows.FILE_NOTIFY_CHANGE_LAST_WRITE|windows.FILE_NOTIFY_CHANGE_SIZE,
            &bytesReturned,
            nil, nil,
        )
        if err != nil {
            continue
        }
        
        // Parse FILE_NOTIFY_INFORMATION to check if our file changed
        if w.isTargetFile(buf[:bytesReturned]) {
            // Debounce: wait 500ms for writes to settle
            time.Sleep(500 * time.Millisecond)
            w.onChange()
        }
    }
}
```

---

## 10. Admin Elevation (`internal/platform/`)

```go
func IsAdmin() bool {
    // Check if current process token has admin privileges
    var sid *windows.SID
    err := windows.AllocateAndInitializeSid(
        &windows.SECURITY_NT_AUTHORITY,
        2,
        windows.SECURITY_BUILTIN_DOMAIN_RID,
        windows.DOMAIN_ALIAS_RID_ADMINS,
        0, 0, 0, 0, 0, 0,
        &sid,
    )
    if err != nil {
        return false
    }
    defer windows.FreeSid(sid)
    
    member, err := windows.Token(0).IsMember(sid)
    if err != nil {
        return false
    }
    return member
}

func RequestElevation() error {
    // Re-launch self with "runas" verb
    exe, _ := os.Executable()
    verb := "runas"
    args := strings.Join(os.Args[1:], " ")
    
    // ShellExecuteW with "runas" triggers UAC prompt
    ret := winapi.ShellExecuteW(0,
        windows.StringToUTF16Ptr(verb),
        windows.StringToUTF16Ptr(exe),
        windows.StringToUTF16Ptr(args),
        nil,
        winapi.SW_SHOWNORMAL,
    )
    if ret <= 32 {
        return fmt.Errorf("elevation failed: %d", ret)
    }
    
    os.Exit(0) // exit non-elevated instance
    return nil
}
```

---

## 11. Application Bootstrap (`cmd/wtm/main.go`)

```go
//go:generate goversioninfo -icon=icon.ico

func main() {
    // 1. Parse flags
    configPath := flag.String("config", defaultConfigPath(), "config file path")
    noTray := flag.Bool("no-tray", false, "disable system tray")
    noBrowser := flag.Bool("no-browser", false, "don't auto-open browser")
    verbose := flag.Bool("verbose", false, "verbose logging")
    flag.Parse()
    
    // 2. Load config
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("config: %v", err)
    }
    
    // 3. Check admin (warn if not)
    if !platform.IsAdmin() {
        log.Println("WARNING: Running without admin privileges. Some features will be limited.")
    }
    
    // 4. Create core components
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    store := storage.NewStore(cfg.Monitoring.HistoryDuration)
    emitter := event.NewEmitter()
    
    collector := collector.New(cfg.Monitoring, store, emitter)
    controller := controller.New(cfg.Controller)
    anomalyEngine := anomaly.NewEngine(cfg.Anomaly, store, emitter)
    aiAdvisor := ai.NewAdvisor(cfg.AI)
    
    sseHub := server.NewSSEHub()
    emitter.Subscribe(sseHub.Broadcast) // pipe all events to SSE
    
    httpServer := server.New(cfg.Server, collector, controller, anomalyEngine, aiAdvisor, sseHub)
    
    // 5. Start everything
    go collector.Start(ctx)
    go anomalyEngine.Start(ctx)
    go httpServer.Start(ctx)
    
    // 6. Config hot-reload
    go config.WatchFile(*configPath, func(newCfg *config.Config) {
        collector.UpdateConfig(newCfg.Monitoring)
        anomalyEngine.UpdateConfig(newCfg.Anomaly)
        aiAdvisor.UpdateConfig(newCfg.AI)
    })
    
    // 7. System tray (blocks on its own thread)
    if !*noTray {
        tray := tray.New(cfg)
        tray.OnOpen = func() { openBrowser(cfg.Server.Port) }
        tray.OnExit = func() { cancel() }
        
        // Subscribe to metrics for tooltip update
        emitter.On("system_metrics", func(data any) {
            snap := data.(*SystemSnapshot)
            tray.UpdateTooltip(snap.CPU.TotalPercent, snap.Memory.UsedPercent)
        })
        
        // Subscribe to alerts for balloon notifications
        emitter.On("alert_new", func(data any) {
            alert := data.(*Alert)
            if alert.Severity >= anomaly.SeverityCritical {
                tray.ShowBalloon(alert.Title, alert.Description, winapi.NIIF_WARNING)
            }
        })
        
        go tray.Start(ctx)
    }
    
    // 8. Auto-open browser
    if cfg.Server.OpenBrowser && !*noBrowser {
        time.AfterFunc(500*time.Millisecond, func() {
            openBrowser(cfg.Server.Port)
        })
    }
    
    // 9. Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    
    select {
    case <-sigCh:
    case <-ctx.Done():
    }
    
    log.Println("Shutting down...")
    cancel()
    httpServer.Shutdown(5 * time.Second)
}

func defaultConfigPath() string {
    appData := os.Getenv("APPDATA")
    if appData == "" {
        appData = "."
    }
    dir := filepath.Join(appData, "WindowsTaskManager")
    os.MkdirAll(dir, 0755)
    return filepath.Join(dir, "config.yaml")
}

func openBrowser(port int) {
    url := fmt.Sprintf("http://127.0.0.1:%d", port)
    exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
```

---

## 12. Dashboard Implementation Notes (`web/`)

### 12.1 SSE Client (`app.js`)

```javascript
class EventSource {
    constructor(types = []) {
        const url = types.length > 0
            ? `/api/v1/events?types=${types.join(',')}`
            : '/api/v1/events';
        
        this.es = new window.EventSource(url);
        this.handlers = {};
        
        // Auto-reconnect with backoff
        this.es.onerror = () => {
            this.es.close();
            setTimeout(() => this.reconnect(), this.backoff());
        };
    }
    
    on(type, handler) {
        this.handlers[type] = handler;
        this.es.addEventListener(type, (e) => {
            handler(JSON.parse(e.data));
        });
    }
}
```

### 12.2 Virtual Scroll (`processes.js`)

Process table uses virtual scrolling for 1000+ processes:

```javascript
class VirtualTable {
    constructor(container, rowHeight = 32) {
        this.container = container;
        this.rowHeight = rowHeight;
        this.data = [];
        this.visibleStart = 0;
        this.visibleEnd = 0;
        
        // Spacer elements for scroll position
        this.topSpacer = document.createElement('div');
        this.bottomSpacer = document.createElement('div');
        this.viewport = document.createElement('div');
        
        container.appendChild(this.topSpacer);
        container.appendChild(this.viewport);
        container.appendChild(this.bottomSpacer);
        
        container.addEventListener('scroll', () => this.onScroll());
    }
    
    setData(data) {
        this.data = data;
        this.render();
    }
    
    onScroll() {
        const scrollTop = this.container.scrollTop;
        const viewportHeight = this.container.clientHeight;
        const overscan = 5; // render extra rows above/below
        
        this.visibleStart = Math.max(0, Math.floor(scrollTop / this.rowHeight) - overscan);
        this.visibleEnd = Math.min(
            this.data.length,
            Math.ceil((scrollTop + viewportHeight) / this.rowHeight) + overscan
        );
        
        this.render();
    }
    
    render() {
        // Only render visible rows
        this.topSpacer.style.height = `${this.visibleStart * this.rowHeight}px`;
        this.bottomSpacer.style.height = `${(this.data.length - this.visibleEnd) * this.rowHeight}px`;
        
        // Diff-update: only update changed cells
        const visible = this.data.slice(this.visibleStart, this.visibleEnd);
        // ... render visible rows into viewport
    }
}
```

### 12.3 Canvas Sparklines (`charts.js`)

```javascript
class Sparkline {
    constructor(canvas, options = {}) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        this.points = new Float64Array(options.maxPoints || 60);
        this.head = 0;
        this.count = 0;
        this.color = options.color || '#4CAF50';
        this.fillColor = options.fillColor || 'rgba(76, 175, 80, 0.1)';
    }
    
    addPoint(value) {
        this.points[this.head] = value;
        this.head = (this.head + 1) % this.points.length;
        if (this.count < this.points.length) this.count++;
        this.draw();
    }
    
    draw() {
        const { ctx, canvas } = this;
        const w = canvas.width;
        const h = canvas.height;
        const dpr = window.devicePixelRatio || 1;
        
        canvas.width = w * dpr;
        canvas.height = h * dpr;
        ctx.scale(dpr, dpr);
        
        ctx.clearRect(0, 0, w, h);
        
        if (this.count < 2) return;
        
        const stepX = w / (this.points.length - 1);
        const max = Math.max(...this.getValues()) || 100;
        
        // Fill area
        ctx.beginPath();
        ctx.moveTo(0, h);
        const values = this.getValues();
        values.forEach((v, i) => {
            const x = i * stepX;
            const y = h - (v / max) * h;
            ctx.lineTo(x, y);
        });
        ctx.lineTo((values.length - 1) * stepX, h);
        ctx.closePath();
        ctx.fillStyle = this.fillColor;
        ctx.fill();
        
        // Line
        ctx.beginPath();
        values.forEach((v, i) => {
            const x = i * stepX;
            const y = h - (v / max) * h;
            if (i === 0) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
        });
        ctx.strokeStyle = this.color;
        ctx.lineWidth = 1.5;
        ctx.stroke();
    }
    
    getValues() {
        const result = [];
        for (let i = 0; i < this.count; i++) {
            const idx = (this.head - this.count + i + this.points.length) % this.points.length;
            result.push(this.points[idx]);
        }
        return result;
    }
}
```

---

## 13. Build Configuration

### 13.1 go.mod

```
module github.com/ersinkoc/WindowsTaskManager

go 1.23

require (
    golang.org/x/sys v0.28.0
    gopkg.in/yaml.v3 v3.0.1
)
```

### 13.2 embed.FS Setup

```go
// In server/server.go
//go:embed all:../../web
var webFS embed.FS

func (s *Server) staticHandler() http.Handler {
    sub, _ := fs.Sub(webFS, "web")
    return http.FileServer(http.FS(sub))
}
```

### 13.3 Build Script (`scripts/build.ps1`)

```powershell
$version = git describe --tags --always 2>$null
if (-not $version) { $version = "dev" }
$commit = git rev-parse --short HEAD 2>$null

$ldflags = "-s -w -H=windowsgui -X main.version=$version -X main.commit=$commit -X main.buildTime=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ss')"

Write-Host "Building WindowsTaskManager $version ($commit)..."
go build -ldflags $ldflags -o WindowsTaskManager.exe ./cmd/wtm/

$size = (Get-Item WindowsTaskManager.exe).Length / 1MB
Write-Host "Built: WindowsTaskManager.exe ($([math]::Round($size, 1)) MB)"
```

---

## 14. Testing Strategy

### 14.1 Unit Tests

- `internal/stats/` — full unit tests for Welford, LinearRegression, RingBuffer (platform-independent)
- `internal/anomaly/` — test each detector with mock snapshots
- `internal/server/router.go` — test pattern matching
- `internal/config/` — test YAML parsing, defaults, validation

### 14.2 Integration Tests (Windows only)

- `internal/collector/` — test actual API calls (skip on CI if not Windows)
- `internal/controller/` — test kill/suspend on dummy child processes
- Build constraint: `//go:build windows`

### 14.3 Test Helper

```go
func mockSnapshot() *SystemSnapshot {
    return &SystemSnapshot{
        CPU:    CPUMetrics{TotalPercent: 45.0, NumLogical: 8},
        Memory: MemoryMetrics{TotalPhys: 16 << 30, UsedPhys: 8 << 30, UsedPercent: 50.0},
        Processes: []ProcessInfo{
            {PID: 100, Name: "explorer.exe", CPUPercent: 2.0, WorkingSet: 100 << 20},
            {PID: 200, ParentPID: 100, Name: "node.exe", CPUPercent: 45.0, WorkingSet: 500 << 20},
            // ... more test processes
        },
    }
}
```
