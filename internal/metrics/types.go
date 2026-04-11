package metrics

import "time"

// CPUMetrics holds system CPU information.
type CPUMetrics struct {
	TotalPercent float64   `json:"total_percent"`
	PerCore      []float64 `json:"per_core"`
	NumLogical   int       `json:"num_logical"`
	Name         string    `json:"name"`
	FreqMHz      uint32    `json:"freq_mhz"`
}

// MemoryMetrics holds system memory information.
type MemoryMetrics struct {
	TotalPhys     uint64  `json:"total_phys"`
	AvailPhys     uint64  `json:"avail_phys"`
	UsedPhys      uint64  `json:"used_phys"`
	UsedPercent   float64 `json:"used_percent"`
	TotalPageFile uint64  `json:"total_page_file"`
	AvailPageFile uint64  `json:"avail_page_file"`
	CommitCharge  uint64  `json:"commit_charge"`
}

// DriveInfo holds per-drive disk information.
type DriveInfo struct {
	Letter     string  `json:"letter"`
	Label      string  `json:"label"`
	FSType     string  `json:"fs_type"`
	TotalBytes uint64  `json:"total_bytes"`
	FreeBytes  uint64  `json:"free_bytes"`
	UsedBytes  uint64  `json:"used_bytes"`
	UsedPct    float64 `json:"used_pct"`
	ReadBPS    uint64  `json:"read_bps"`
	WriteBPS   uint64  `json:"write_bps"`
	ReadIOPS   uint64  `json:"read_iops"`
	WriteIOPS  uint64  `json:"write_iops"`
}

// DiskMetrics aggregates per-drive metrics.
type DiskMetrics struct {
	Drives []DriveInfo `json:"drives"`
}

// InterfaceInfo holds per-interface network metrics.
type InterfaceInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	SpeedMbps uint64 `json:"speed_mbps"`
	InBPS     uint64 `json:"in_bps"`
	OutBPS    uint64 `json:"out_bps"`
	InPPS     uint64 `json:"in_pps"`
	OutPPS    uint64 `json:"out_pps"`
	InErrors  uint64 `json:"in_errors"`
	OutErrors uint64 `json:"out_errors"`
}

// NetworkMetrics aggregates network interfaces.
type NetworkMetrics struct {
	Interfaces   []InterfaceInfo `json:"interfaces"`
	TotalUpBPS   uint64          `json:"total_up_bps"`
	TotalDownBPS uint64          `json:"total_down_bps"`
}

// GPUMetrics holds GPU information.
type GPUMetrics struct {
	Name        string  `json:"name"`
	Utilization float64 `json:"utilization"`
	VRAMUsed    uint64  `json:"vram_used"`
	VRAMTotal   uint64  `json:"vram_total"`
	Temperature int     `json:"temperature"`
	Available   bool    `json:"available"`
}

// ProcessInfo holds per-process metrics.
type ProcessInfo struct {
	PID           uint32  `json:"pid"`
	ParentPID     uint32  `json:"parent_pid"`
	Name          string  `json:"name"`
	ExePath       string  `json:"exe_path"`
	CPUPercent    float64 `json:"cpu_percent"`
	WorkingSet    uint64  `json:"working_set"`
	PrivateBytes  uint64  `json:"private_bytes"`
	PageFaults    uint32  `json:"page_faults"`
	IOReadBytes   uint64  `json:"io_read_bytes"`
	IOWriteBytes  uint64  `json:"io_write_bytes"`
	IOReadOps     uint64  `json:"io_read_ops"`
	IOWriteOps    uint64  `json:"io_write_ops"`
	ThreadCount   uint32  `json:"thread_count"`
	CreateTime    int64   `json:"create_time"`
	IsCritical    bool    `json:"is_critical"`
	Status        string  `json:"status"`
	Connections   int     `json:"connections"`
	PriorityClass uint32  `json:"priority_class"`
}

// ProcessNode is a node in a process tree.
type ProcessNode struct {
	Process  ProcessInfo    `json:"process"`
	Children []*ProcessNode `json:"children,omitempty"`
	Depth    int            `json:"depth"`
	IsOrphan bool           `json:"is_orphan"`
}

// PortBinding describes one TCP/UDP listener or connection.
type PortBinding struct {
	Protocol   string `json:"protocol"`
	LocalAddr  string `json:"local_addr"`
	LocalPort  uint16 `json:"local_port"`
	RemoteAddr string `json:"remote_addr"`
	RemotePort uint16 `json:"remote_port"`
	State      string `json:"state"`
	StateCode  uint32 `json:"state_code"`
	PID        uint32 `json:"pid"`
	Process    string `json:"process"`
	Label      string `json:"label"`
	Since      int64  `json:"since"`
}

// SystemSnapshot is the full state of the system at a point in time.
type SystemSnapshot struct {
	Timestamp    time.Time      `json:"timestamp"`
	CPU          CPUMetrics     `json:"cpu"`
	Memory       MemoryMetrics  `json:"memory"`
	GPU          GPUMetrics     `json:"gpu"`
	Disk         DiskMetrics    `json:"disk"`
	Network      NetworkMetrics `json:"network"`
	Processes    []ProcessInfo  `json:"processes"`
	ProcessTree  []*ProcessNode `json:"process_tree,omitempty"`
	PortBindings []PortBinding  `json:"port_bindings,omitempty"`
}

// ProcessName returns the name for a PID, or "" if not found.
func (s *SystemSnapshot) ProcessName(pid uint32) string {
	for i := range s.Processes {
		if s.Processes[i].PID == pid {
			return s.Processes[i].Name
		}
	}
	return ""
}

// TimestampedSystem is a snapshot timestamped for ring buffer storage.
type TimestampedSystem struct {
	Time    time.Time      `json:"time"`
	CPU     CPUMetrics     `json:"cpu"`
	Memory  MemoryMetrics  `json:"memory"`
	GPU     GPUMetrics     `json:"gpu"`
	Network NetworkMetrics `json:"network"`
	Disk    DiskMetrics    `json:"disk"`
}
