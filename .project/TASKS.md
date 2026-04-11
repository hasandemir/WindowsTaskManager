# WindowsTaskManager — TASKS

**Repository:** `github.com/ersinkoc/WindowsTaskManager`
**Specification:** SPECIFICATION.md
**Implementation:** IMPLEMENTATION.md
**Total Tasks:** 127
**Estimated Duration:** 8-10 weeks

---

## Phase 1: Foundation (Tasks 1-22)

### 1.1 Project Scaffolding

- [ ] **Task 1:** Initialize Go module (`go mod init github.com/ersinkoc/WindowsTaskManager`), add `golang.org/x/sys` and `gopkg.in/yaml.v3` dependencies, create directory structure as defined in SPECIFICATION.md §11
- [ ] **Task 2:** Create `internal/winapi/types.go` — define all Windows struct types: `FILETIME`, `MEMORYSTATUSEX`, `PROCESSENTRY32W`, `PROCESS_MEMORY_COUNTERS_EX`, `IO_COUNTERS`, `SYSTEM_PROCESSOR_PERFORMANCE_INFORMATION`, `MIB_TCPROW_OWNER_PID`, `MIB_TCP6ROW_OWNER_PID`, `MIB_UDPROW_OWNER_PID`, `MIB_IF_ROW2`, `DISK_PERFORMANCE`, `NOTIFYICONDATAW`, all Job Object structs, TCP state constants, helper methods (`FILETIME.Ticks()`, `FileTimeToUnix()`)
- [ ] **Task 3:** Create `internal/winapi/kernel32.go` — lazy load `kernel32.dll`, define all proc references: `GetSystemTimes`, `GlobalMemoryStatusEx`, `CreateToolhelp32Snapshot`, `Process32FirstW`, `Process32NextW`, `OpenProcess`, `TerminateProcess`, `GetProcessTimes`, `GetProcessIoCounters`, `QueryFullProcessImageNameW`, `IsProcessCritical`, `SetPriorityClass`, `SetProcessAffinityMask`, `CreateJobObjectW`, `AssignProcessToJobObject`, `SetInformationJobObject`, `SuspendThread`, `ResumeThread`, `GetDiskFreeSpaceExW`, `DeviceIoControl`, `GetLogicalDriveStringsW`, `GetDriveTypeW`, `GetVolumeInformationW`, `CreateFileW`. Implement Go wrapper functions with proper error handling for each
- [ ] **Task 4:** Create `internal/winapi/ntdll.go` — lazy load `ntdll.dll`, define `NtQuerySystemInformation` proc. Implement wrapper for `SystemProcessorPerformanceInformation` (class 8) returning per-core times array
- [ ] **Task 5:** Create `internal/winapi/psapi.go` — lazy load `psapi.dll`, define `GetProcessMemoryInfo` proc (K32GetProcessMemoryInfo fallback). Implement wrapper returning `PROCESS_MEMORY_COUNTERS_EX`
- [ ] **Task 6:** Create `internal/winapi/iphlpapi.go` — lazy load `iphlpapi.dll`, define `GetExtendedTcpTable`, `GetExtendedUdpTable`, `GetIfTable2`, `FreeMibTable` procs. Implement wrappers with dynamic buffer sizing pattern (call once for size, allocate, call again)
- [ ] **Task 7:** Create `internal/winapi/user32.go` — lazy load `user32.dll`, define `IsHungAppWindow`, `CreateWindowExW`, `DefWindowProcW`, `RegisterClassExW`, `GetMessageW`, `TranslateMessage`, `DispatchMessageW`, `PostQuitMessage`, `TrackPopupMenu`, `CreatePopupMenu`, `AppendMenuW`, `DestroyMenu`, `GetCursorPos`, `SetForegroundWindow` procs with wrappers
- [ ] **Task 8:** Create `internal/winapi/shell32.go` — lazy load `shell32.dll`, define `Shell_NotifyIconW`, `ShellExecuteW` procs with wrappers. Define `NIM_ADD`, `NIM_MODIFY`, `NIM_DELETE`, `NIF_*`, `NIIF_*` constants
- [ ] **Task 9:** Create `internal/winapi/gdi32.go` — lazy load `gdi32.dll`, define `D3DKMTQueryStatistics`, `D3DKMTEnumAdapters2` procs. Define D3DKMT struct types: `D3DKMT_QUERYSTATISTICS`, `D3DKMT_ENUMADAPTERS2`, `D3DKMT_ADAPTERINFO`. Also define `CreateCompatibleDC`, `CreateCompatibleBitmap`, `SelectObject`, `DeleteDC`, `DeleteObject`, `CreateIconIndirect` for dynamic tray icon generation
- [ ] **Task 10:** Create `internal/winapi/advapi32.go` — lazy load `advapi32.dll`, define `OpenProcessToken`, `GetTokenInformation`, `AllocateAndInitializeSid`, `FreeSid`, `CheckTokenMembership`, `RegOpenKeyExW`, `RegQueryValueExW`, `RegCloseKey` procs with wrappers

### 1.2 Statistical Utilities

- [ ] **Task 11:** Create `internal/stats/welford.go` — implement Welford's online algorithm with `Add(float64)`, `Mean()`, `Variance()`, `StdDev()`, `Count()`, `IsAnomaly(value, nSigma)`, `Reset()` methods. Write comprehensive unit tests including edge cases (single value, identical values, negative values, very large values)
- [ ] **Task 12:** Create `internal/stats/regression.go` — implement `LinearRegression(xs, ys []float64) (slope, intercept, rSquared float64)` using least squares method. Write unit tests with known datasets (perfect line, noisy data, flat line, vertical data)
- [ ] **Task 13:** Create `internal/stats/ema.go` — implement `EMA` struct with `NewEMA(alpha float64)`, `Add(value float64)`, `Value() float64`, `Reset()`. Alpha range 0-1. Write unit tests
- [ ] **Task 14:** Create `internal/stats/ringbuf.go` — implement generic `RingBuffer[T]` with atomic head index, `NewRingBuffer[T](capacity)`, `Add(T)`, `Slice() []T`, `Len()`, `Last() (T, bool)`. Also implement `SlidingWindow[T]` wrapper. Write unit tests including concurrent read/write tests

### 1.3 Configuration

- [ ] **Task 15:** Create `internal/config/config.go` — define full `Config` struct tree matching SPECIFICATION.md §10.2 schema: `ServerConfig`, `MonitoringConfig`, `ControllerConfig`, `AnomalyConfig` (with sub-configs for each detector), `NotificationConfig`, `WellKnownPorts`, `AIConfig`, `UIConfig`. Implement `DefaultConfig()` returning sensible defaults. Implement `Validate()` checking value ranges, required fields
- [ ] **Task 16:** Create `internal/config/loader.go` — implement `Load(path string) (*Config, error)` that reads YAML, overlays on defaults, validates. Implement `Save(path string, cfg *Config) error`. Handle missing file (create with defaults), invalid YAML (return clear error), partial config (merge with defaults). Write unit tests with temp files
- [ ] **Task 17:** Create `internal/config/watcher.go` — implement native file watcher using `ReadDirectoryChangesW` (per IMPLEMENTATION.md §9.2). Watch config directory for changes to config file. Debounce 500ms. Call `onChange` callback. Handle watcher errors gracefully (log and continue)

### 1.4 Core Infrastructure

- [ ] **Task 18:** Create `internal/storage/store.go` — implement `Store` that holds typed ring buffers for each metric type: `SystemMetrics`, `ProcessMetrics` (per-PID), `NetworkMetrics`, `DiskMetrics`. Methods: `AddSystemMetrics(m)`, `AddProcessMetrics(pid, m)`, `SystemHistory(duration) []TimestampedMetric`, `ProcessHistory(pid, duration)`, `LatestSnapshot() *SystemSnapshot`. Thread-safe with `sync.RWMutex`
- [ ] **Task 19:** Create `internal/event/emitter.go` — implement `Emitter` with `Subscribe(fn func(eventType string, data any))`, `On(eventType string, fn func(data any))`, `Emit(eventType string, data any)`. Non-blocking emission (goroutine per subscriber call with recovery). Used to pipe collector data to SSE hub, anomaly engine, tray
- [ ] **Task 20:** Create `internal/platform/elevation.go` — implement `IsAdmin() bool` using token check (per IMPLEMENTATION.md §10), `RequestElevation() error` using `ShellExecuteW` with "runas" verb. Write test that verifies `IsAdmin()` returns a bool (actual admin check requires manual testing)
- [ ] **Task 21:** Create `internal/server/router.go` — implement custom HTTP router with path parameter support (per IMPLEMENTATION.md §7.1). Pattern syntax: `/api/v1/processes/{pid}/kill`. Methods: `GET`, `POST`, `PUT`, `DELETE`, `Handle(method, pattern, handler)`. `Param(r *http.Request, name string) string` for extracting parameters. Write unit tests for pattern matching, parameter extraction, method filtering, 404 handling
- [ ] **Task 22:** Create helper functions in `internal/server/helpers.go` — `writeJSON(w, data)` (set Content-Type, encode JSON), `writeError(w, status, message)` (JSON error response), `readJSON(r, &target)` (decode request body), `formatBytes(uint64) string` (human-readable: KB, MB, GB)

---

## Phase 2: Collectors (Tasks 23-40)

### 2.1 CPU Collector

- [ ] **Task 23:** Create `internal/collector/cpu.go` — implement `CPUCollector` with `NewCPUCollector()` and `Collect() (*CPUMetrics, error)`. Use `GetSystemTimes()` for total CPU. Store previous ticks for delta calculation. Return `CPUMetrics` with `TotalPercent`, `NumLogical`
- [ ] **Task 24:** Add per-core CPU collection to `cpu.go` — use `NtQuerySystemInformation(SystemProcessorPerformanceInformation)` to get per-logical-processor idle/kernel/user times. Delta calculation per core. Add `PerCore []float64` to `CPUMetrics`
- [ ] **Task 25:** Add CPU name and frequency to `cpu.go` — read CPU name from registry `HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0\ProcessorNameString` (cache, read once). Read current MHz from same registry key. Add `Name string`, `FreqMHz uint32` to `CPUMetrics`

### 2.2 Memory Collector

- [ ] **Task 26:** Create `internal/collector/memory.go` — implement `MemCollector` with `Collect() (*MemoryMetrics, error)` using `GlobalMemoryStatusEx()`. Return all fields: `TotalPhys`, `AvailPhys`, `UsedPhys`, `UsedPercent`, `TotalPageFile`, `AvailPageFile`, `CommitCharge`

### 2.3 Process Collector

- [ ] **Task 27:** Create `internal/collector/process.go` — implement `ProcessCollector` with `Collect() ([]ProcessInfo, error)` using Toolhelp32 snapshot. Enumerate all processes, extract PID, ParentPID, Name, ThreadCount from `PROCESSENTRY32W`
- [ ] **Task 28:** Add per-process CPU calculation to `process.go` — for each process, `OpenProcess` → `GetProcessTimes` → compute CPU% from kernel+user time deltas divided by elapsed time × num CPUs. Store previous times in `states map[uint32]*processState`. Handle access denied gracefully (skip process)
- [ ] **Task 29:** Add per-process memory metrics to `process.go` — `GetProcessMemoryInfo` → extract `WorkingSetSize`, `PrivateUsage`, `PageFaultCount`, `PeakWorkingSetSize`. Add to `ProcessInfo`
- [ ] **Task 30:** Add per-process I/O metrics to `process.go` — `GetProcessIoCounters` → compute delta for `ReadTransferCount`, `WriteTransferCount`, `ReadOperationCount`, `WriteOperationCount`. Store previous values, calculate bytes/sec and ops/sec
- [ ] **Task 31:** Add process path, critical check, priority to `process.go` — `QueryFullProcessImageNameW` for full exe path. `IsProcessCritical` for system process detection. `GetPriorityClass` for current priority. Add `ExePath`, `IsCritical`, `PriorityClass` to `ProcessInfo`
- [ ] **Task 32:** Add process state cleanup to `process.go` — on each collection cycle, remove states for PIDs no longer in snapshot (dead processes). Prevent memory leak in `states` map

### 2.4 Process Tree

- [ ] **Task 33:** Create `internal/collector/process_tree.go` — implement `BuildProcessTree(processes []ProcessInfo) []*ProcessNode` per IMPLEMENTATION.md §2.5. Build parent-child relationships, detect orphans (parent PID not in current snapshot), compute depths, sort children by PID. Implement `TreeStats(node) (childCount, totalMem, totalCPU)` for subtree aggregation

### 2.5 Network Collector

- [ ] **Task 34:** Create `internal/collector/network.go` — implement `NetCollector` with `Collect() (*NetworkMetrics, error)` using `GetIfTable2()`. Parse `MIB_IF_ROW2` array, filter out loopback and down interfaces. Compute delta for `InOctets`/`OutOctets` → bytes/sec. Compute total upload/download. Call `FreeMibTable` when done

### 2.6 Port Collector

- [ ] **Task 35:** Create `internal/collector/ports.go` — implement `PortCollector` with `Collect(processNames map[uint32]string) ([]PortBinding, error)`. Call `GetExtendedTcpTable` for IPv4 and IPv6, `GetExtendedUdpTable` for IPv4 and IPv6. Parse results into `PortBinding` structs. Apply `ntohs()` for port byte order conversion. Map PIDs to process names
- [ ] **Task 36:** Add well-known port labeling and first-seen tracking to `ports.go` — label ports from config's `well_known_ports` map. Track first-seen time for each unique port+PID+state combination using internal map. Add `Label` and `Since` to `PortBinding`

### 2.7 Disk Collector

- [ ] **Task 37:** Create `internal/collector/disk.go` — implement `DiskCollector` with `Collect() (*DiskMetrics, error)`. Enumerate drive letters via `GetLogicalDriveStringsW`, filter to fixed drives (`GetDriveTypeW == DRIVE_FIXED`). Per drive: `GetDiskFreeSpaceExW` for space, `GetVolumeInformationW` for label/filesystem
- [ ] **Task 38:** Add disk I/O counters to `disk.go` — open `\\.\PhysicalDriveN` with `CreateFileW`, call `DeviceIoControl(IOCTL_DISK_PERFORMANCE)`, parse `DISK_PERFORMANCE` struct. Delta calculation for read/write bytes and ops per second. Map physical drives to logical drives

### 2.8 GPU Collector

- [ ] **Task 39:** Create `internal/collector/gpu.go` — implement `GPUCollector` with D3DKMT provider. `initD3DKMT()`: enumerate adapters via `D3DKMTEnumAdapters2`, get adapter LUID and name. `collectD3DKMT()`: query per-segment stats for VRAM (dedicated segments), query per-node stats for utilization (compute from running time deltas). Return `GPUMetrics`
- [ ] **Task 40:** Add optional NVML provider to `gpu.go` — implement `NVMLProvider` that dynamically loads `nvml.dll` via `windows.LoadDLL`. Resolve procs: `nvmlInit_v2`, `nvmlShutdown`, `nvmlDeviceGetCount`, `nvmlDeviceGetHandleByIndex`, `nvmlDeviceGetName`, `nvmlDeviceGetUtilizationRates`, `nvmlDeviceGetMemoryInfo`, `nvmlDeviceGetTemperature`. If NVML available, supplement D3DKMT data with temperature. Graceful fallback if NVML not found

### 2.9 Collector Orchestrator

- [ ] **Task 41:** Create `internal/collector/collector.go` — implement `Collector` orchestrator (per IMPLEMENTATION.md §2.1). Initialize all sub-collectors. `Start(ctx)` launches goroutines with configurable tick intervals. Each tick: collect metrics → store in ring buffer → emit via event emitter. `Snapshot() *SystemSnapshot` returns latest combined snapshot. `UpdateConfig(cfg)` for hot-reload. Separate tick goroutines for: main (CPU+MEM+Process), tree, ports, GPU, disk

---

## Phase 3: Controller Engine (Tasks 42-52)

- [ ] **Task 42:** Create `internal/controller/safety.go` — implement protected process list. `IsSafe(pid uint32) error` checks: PID 0/4, `IsProcessCritical()` API, configurable protected names list. Returns `ErrProtectedProcess` with descriptive message if unsafe. Helper `getProcessName(pid) string`
- [ ] **Task 43:** Create `internal/controller/killer.go` — implement `Kill(pid uint32) error` with safety checks → `OpenProcess(PROCESS_TERMINATE)` → `TerminateProcess(handle, 1)`. Implement `KillTree(pid uint32, tree []*ProcessNode) error` — find node in tree, collect all descendant PIDs, kill bottom-up (children first). Return aggregated errors
- [ ] **Task 44:** Create `internal/controller/suspender.go` — implement `Suspend(pid uint32) error` and `Resume(pid uint32) error`. Use `CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD)` to enumerate threads of target process, call `SuspendThread`/`ResumeThread` on each. Return aggregated errors
- [ ] **Task 45:** Create `internal/controller/priority.go` — implement `SetPriority(pid uint32, level string) error`. Map level strings ("idle", "below_normal", "normal", "above_normal", "high", "realtime") to Windows constants. `OpenProcess(PROCESS_SET_INFORMATION)` → `SetPriorityClass`. Return current priority via `GetPriorityClass`
- [ ] **Task 46:** Create `internal/controller/affinity.go` — implement `SetAffinity(pid uint32, mask uint64) error`. Validate mask against number of logical processors. `OpenProcess(PROCESS_SET_INFORMATION)` → `SetProcessAffinityMask`. Implement `GetAffinity(pid) (uint64, error)` via `GetProcessAffinityMask`
- [ ] **Task 47:** Create `internal/controller/limiter.go` — implement `Limiter` struct managing Job Objects per PID (per IMPLEMENTATION.md §3.2). `getOrCreateJob(pid)` creates Job Object and assigns process. Track active jobs in `map[uint32]*jobEntry`
- [ ] **Task 48:** Add `SetCPULimit(pid, percent)` to `limiter.go` — configure `JOBOBJECT_CPU_RATE_CONTROL_INFORMATION` with `JOB_OBJECT_CPU_RATE_CONTROL_ENABLE | JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP`. CpuRate = percent × 100
- [ ] **Task 49:** Add `SetMemoryLimit(pid, maxBytes)` to `limiter.go` — configure `JOBOBJECT_EXTENDED_LIMIT_INFORMATION` with `JOB_OBJECT_LIMIT_PROCESS_MEMORY`
- [ ] **Task 50:** Add `SetProcessLimit(pid, maxChildren)` to `limiter.go` — configure `JOBOBJECT_BASIC_LIMIT_INFORMATION` with `JOB_OBJECT_LIMIT_ACTIVE_PROCESS` and `ActiveProcessLimit`. This prevents fork bombs by limiting child process count
- [ ] **Task 51:** Add `RemoveLimits(pid)` to `limiter.go` — close Job Object handle (releases all limits automatically), remove from tracking map
- [ ] **Task 52:** Create `internal/controller/controller.go` — facade struct wrapping `Killer`, `Suspender`, `Priority`, `Affinity`, `Limiter`. Single constructor `New(cfg *ControllerConfig)`. Expose all methods. Thread-safe

---

## Phase 4: Anomaly Detection (Tasks 53-68)

- [ ] **Task 53:** Create `internal/anomaly/alert.go` — define `Alert` struct (ID, Type, Severity, Title, Description, PID, Process, Data, timestamps, Dismissed, SnoozedUntil), `Severity` enum (Info, Warning, Critical, Emergency). Implement `AlertManager` with `Add`, `ShouldEmit` (dedup by type+PID, severity escalation), `Dismiss`, `Snooze`, `CheckResolved`, `Active() []Alert`, `History() []Alert`
- [ ] **Task 54:** Create `internal/anomaly/engine.go` — define `Detector` interface with `Name() string`, `Detect(snap *SystemSnapshot) []Alert`, `Reset()`. Implement `Engine` orchestrator that initializes all detectors from config, runs detection loop on configurable interval, emits new/resolved alerts. `UpdateConfig` for hot-reload (re-initialize detectors)
- [ ] **Task 55:** Create `internal/anomaly/spawnstorm.go` — implement `SpawnStormDetector` per IMPLEMENTATION.md §4.3. Track process creation events via PID set diff between snapshots. Group new processes by parent PID. Sliding 60s window for spawn rate. Alert thresholds: `max_children_per_minute` → CRITICAL, `max_total_children` → EMERGENCY. Include spawn rate, total children, total memory in alert data
- [ ] **Task 56:** Create `internal/anomaly/memleak.go` — implement `MemLeakDetector` per IMPLEMENTATION.md §4.4. Per-process `memTracker` with Welford stats and sliding window of memory samples. Linear regression on window to compute growth rate (slope) and confidence (R²). Alert when growth > `min_growth_rate` AND R² > `min_r_squared`. Include projected time to threshold. Minimum 30 samples before analysis
- [ ] **Task 57:** Create `internal/anomaly/hung.go` — implement `HungDetector`. Track per-process CPU delta and I/O delta. If both zero for > `zero_activity_threshold` → check `IsHungAppWindow()` for GUI processes. WARNING at threshold, CRITICAL at `critical_hung_threshold`. Skip processes in `idle_whitelist`. Include inactivity duration and held memory in alert
- [ ] **Task 58:** Create `internal/anomaly/orphan.go` — implement `OrphanDetector`. On each tree rebuild, check if process's ParentPID exists in current snapshot. If parent missing and process wasn't top-level at startup → orphan. WARNING if consuming significant resources (CPU > threshold or memory > threshold). CRITICAL if orphan has its own children (orphan chain)
- [ ] **Task 59:** Create `internal/anomaly/runaway.go` — implement `RunawayCPUDetector`. Track per-process CPU% over rolling window. If CPU > `cpu_threshold` continuously for > `duration_threshold` → WARNING. If exceeds `critical_duration` → CRITICAL. Skip processes in `high_cpu_whitelist`. Track continuous high-CPU duration
- [ ] **Task 60:** Create `internal/anomaly/portconflict.go` — implement `PortConflictDetector` per IMPLEMENTATION.md §4.5. Group bindings by port. Detect: multiple PIDs on same port, stale TIME_WAIT/CLOSE_WAIT exceeding thresholds, port ownership changes (hijack). Track stale state first-seen times
- [ ] **Task 61:** Create `internal/anomaly/netanomaly.go` — implement `NetAnomalyDetector`. Per-process connection count tracking with Welford stats. Alert when connections > mean + `connection_sigma` × stddev. Per-interface bandwidth spike detection (> 5× rolling average). System-wide connection count check against `max_system_connections`
- [ ] **Task 62:** Create `internal/anomaly/newprocess.go` — implement `NewProcessDetector`. Maintain set of known executable paths (learned from first scan). INFO alert when new executable path appears. WARNING if new exe is in a suspicious path (`%TEMP%`, Downloads, etc.)
- [ ] **Task 63:** Create `internal/anomaly/rules.go` — implement rule loading from config. Each detector reads its config section. `ReloadRules(cfg)` method on Engine re-initializes detectors with new config. Validate rule values (positive numbers, valid thresholds)
- [ ] **Task 64:** Write unit tests for `SpawnStormDetector` — test with mock snapshots: normal process creation (no alert), gradual spawn (no alert), rapid spawn exceeding threshold (CRITICAL), exceeding total children (EMERGENCY), spawn rate calculation
- [ ] **Task 65:** Write unit tests for `MemLeakDetector` — test with mock samples: stable memory (no alert), sawtooth pattern (no alert, low R²), monotonic increase (WARNING), monotonic increase above memory threshold (CRITICAL), verify growth rate and R² calculations
- [ ] **Task 66:** Write unit tests for `HungDetector` — test with mock snapshots: active process (no alert), process with zero CPU but positive I/O (no alert), process with zero everything for threshold duration (WARNING), exceeding critical duration (CRITICAL), whitelisted process (no alert)
- [ ] **Task 67:** Write unit tests for `PortConflictDetector` — test: single listener (no alert), multiple PIDs same port (WARNING), stale TIME_WAIT exceeding threshold (WARNING), port ownership change (WARNING)
- [ ] **Task 68:** Write unit tests for `AlertManager` — test dedup logic (same type+PID within window → suppress), severity escalation (higher severity re-emits), dismiss, snooze (suppressed until expiry), auto-resolve

---

## Phase 5: HTTP Server & API (Tasks 69-86)

- [ ] **Task 69:** Create `internal/server/server.go` — implement `Server` struct with `New(cfg, collector, controller, anomaly, ai, sseHub)`. Configure `http.Server` binding to `127.0.0.1:port`. Register all routes. `Start(ctx)` runs server. `Shutdown(timeout)` graceful shutdown. Serve embedded static files from `web/` via `embed.FS`
- [ ] **Task 70:** Create `internal/server/sse.go` — implement `SSEHub` per IMPLEMENTATION.md §7.2. `ServeHTTP` for SSE endpoint. Client registration/deregistration. `Broadcast(eventType, data)` to all clients. Event type filtering via `?types=` query param. Non-blocking send with buffered channels. Auto-drop slow clients
- [ ] **Task 71:** Create `internal/server/middleware.go` — implement logging middleware (method, path, status, duration), recovery middleware (catch panics, return 500), CORS middleware (not needed for localhost, but add no-op for future)
- [ ] **Task 72:** Create `internal/server/api_system.go` — implement handlers: `GET /api/v1/system` (current system metrics), `GET /api/v1/system/history?duration=5m` (historical metrics from ring buffer), `GET /api/v1/system/cpu/cores` (per-core CPU usage)
- [ ] **Task 73:** Create `internal/server/api_process.go` — implement handlers: `GET /api/v1/processes` (all processes, with sort/filter/limit query params), `GET /api/v1/processes/{pid}` (single process), `GET /api/v1/processes/{pid}/history` (process metric history), `GET /api/v1/processes/{pid}/connections` (process connections), `GET /api/v1/processes/{pid}/children` (process children), `GET /api/v1/processes/tree` (full tree)
- [ ] **Task 74:** Create `internal/server/api_process_actions.go` — implement handlers: `POST /processes/{pid}/kill`, `POST /processes/{pid}/kill-tree`, `POST /processes/{pid}/suspend`, `POST /processes/{pid}/resume`, `PUT /processes/{pid}/priority` (body: `{"level":"..."}`) , `PUT /processes/{pid}/affinity` (body: `{"mask":"0x0F"}`), `PUT /processes/{pid}/cpu-limit` (body: `{"percent":30}`), `PUT /processes/{pid}/memory-limit` (body: `{"bytes":...}`), `DELETE /processes/{pid}/limits`
- [ ] **Task 75:** Create `internal/server/api_ports.go` — implement handlers: `GET /api/v1/ports` (all listening ports with PID mapping), `GET /api/v1/ports/{port}` (specific port detail), `GET /api/v1/connections` (all TCP/UDP connections with optional PID filter)
- [ ] **Task 76:** Create `internal/server/api_alerts.go` — implement handlers: `GET /api/v1/alerts` (active alerts), `GET /api/v1/alerts/history` (alert history), `POST /api/v1/alerts/{id}/dismiss`, `POST /api/v1/alerts/{id}/snooze?duration=30m`
- [ ] **Task 77:** Create `internal/server/api_ai.go` — implement handlers: `POST /api/v1/ai/analyze` (body: `{"alert_id":"..."}` or `{"question":"..."}`), `POST /api/v1/ai/chat` (body: `{"message":"..."}`), `GET /api/v1/ai/status` (enabled, API key set, rate limit status), `PUT /api/v1/ai/config` (body: `{"api_key":"...", "language":"tr"}`)
- [ ] **Task 78:** Create `internal/server/api_rules.go` — implement handlers: `GET /api/v1/rules` (current anomaly rules), `PUT /api/v1/rules` (update rules, validate, hot-reload), `POST /api/v1/rules/reload` (force reload from YAML file)
- [ ] **Task 79:** Create `internal/server/api_config.go` — implement handlers: `GET /api/v1/config` (current config, redact API key), `PUT /api/v1/config` (update config, validate, apply non-destructive changes)
- [ ] **Task 80:** Wire all routes in `server.go` — register all API handlers with router. Mount static file handler at `/`. Mount SSE handler at `/api/v1/events`. Apply middleware chain (logging → recovery → handler)
- [ ] **Task 81:** Write integration tests for process API — test list processes (verify JSON structure), test sort/filter query params, test kill (spawn a dummy child process, kill it, verify it's gone), test 404 for invalid PID
- [ ] **Task 82:** Write integration tests for SSE — test client connect, receive events, type filtering, client disconnect cleanup
- [ ] **Task 83:** Write integration tests for port API — test list ports (verify JSON structure), test port detail endpoint
- [ ] **Task 84:** Write integration tests for alert API — test list alerts, dismiss, snooze, history
- [ ] **Task 85:** Write router unit tests — test pattern matching with parameters, method filtering, multiple routes, 404 handling, parameter extraction
- [ ] **Task 86:** Write helper unit tests — test `formatBytes` (0, KB, MB, GB, TB ranges), `writeJSON` output format, `writeError` output format

---

## Phase 6: AI Advisor (Tasks 87-96)

- [ ] **Task 87:** Create `internal/ai/advisor.go` — implement `Advisor` struct with `NewAdvisor(cfg *AIConfig)`, `Analyze(ctx, *AnalysisRequest) (*AnalysisResponse, error)`. HTTP client with 30s timeout. Set headers: `x-api-key`, `anthropic-version: 2023-06-01`. Handle response codes: 200 (parse content), 401 (ErrInvalidAPIKey), 429 (ErrRateLimited), other (generic error with body). Parse `content[].text` from response
- [ ] **Task 88:** Create `internal/ai/prompt.go` — implement `systemPrompt(language string) string` returning Windows expert persona prompt. Implement `buildPrompt(req *AnalysisRequest, cfg *AIConfig) string` that formats: system metrics summary, active alerts, top processes (by CPU + memory), listening ports (if enabled), specific alert detail (if analyzing one), user question. Keep prompt concise (<4K tokens)
- [ ] **Task 89:** Create `internal/ai/parser.go` — implement `parseSuggestions(text string) []Action`. Parse AI response text to extract actionable suggestions. Look for patterns like: kill/terminate PID mentions → Kill action, CPU/memory limit suggestions → Limit action, priority change suggestions → Priority action. Return structured `Action` array with type, label, PID, params
- [ ] **Task 90:** Create `internal/ai/cache.go` — implement `ResponseCache` with `Get(key string) (*AnalysisResponse, bool)`, `Set(key string, resp *AnalysisResponse, ttl time.Duration)`, `Invalidate(key string)`. Cache key derived from alert type + process name + severity hash. TTL-based expiry. Max 100 entries with LRU eviction
- [ ] **Task 91:** Create `internal/ai/ratelimit.go` — implement `RateLimiter` with `NewRateLimiter(maxPerMinute int)`, `Allow() bool`. Token bucket algorithm: refill `maxPerMinute` tokens per minute, consume 1 per request. Thread-safe with mutex
- [ ] **Task 92:** Add chat conversation management to `advisor.go` — maintain in-memory conversation history (last 10 messages per session). Each chat request includes system snapshot as context + conversation history. New method `Chat(ctx, question string, snap *SystemSnapshot) (*AnalysisResponse, error)`
- [ ] **Task 93:** Add `UpdateConfig(cfg *AIConfig)` to advisor — allow runtime config changes (API key, model, language, etc.) without restart. Validate new API key by making a minimal test request
- [ ] **Task 94:** Write unit tests for `prompt.go` — verify prompt structure with mock snapshots. Test with alerts present, without alerts, with user question, language switching (tr/en), process tree inclusion
- [ ] **Task 95:** Write unit tests for `cache.go` — test set/get, TTL expiry, LRU eviction, invalidation, concurrent access
- [ ] **Task 96:** Write unit tests for `ratelimit.go` — test allow/deny, refill over time, burst handling, thread safety

---

## Phase 7: Web Dashboard (Tasks 97-115)

### 7.1 Core Dashboard

- [ ] **Task 97:** Create `web/index.html` — single-page application shell. Include CSS and JS files. Define layout structure: header (title + settings + AI toggle), metrics bar (CPU/MEM/GPU/Disk/Net), alerts panel, tab navigation (Processes/Tree/Ports/AI Chat), main content area. Meta viewport for responsive. No external CDN dependencies
- [ ] **Task 98:** Create `web/css/style.css` — main stylesheet. CSS custom properties for theming (`--bg-primary`, `--text-primary`, `--accent`, `--danger`, `--warning`, `--success`, etc.). CSS Grid for main layout. Flexbox for component layouts. Base font: system-ui stack. Responsive breakpoints. Scrollbar styling. Transition animations for state changes
- [ ] **Task 99:** Create `web/css/theme-light.css` and `web/css/theme-dark.css` — define CSS custom property values for each theme. Light: white backgrounds, dark text, blue accents. Dark: dark gray backgrounds, light text, cyan accents. System theme detection via `prefers-color-scheme` media query
- [ ] **Task 100:** Create `web/js/app.js` — application bootstrap. Initialize SSE connection with auto-reconnect (exponential backoff). Route SSE events to appropriate modules. Theme initialization from localStorage. Tab navigation. Global error handling. State management (current tab, current sort, search filter)
- [ ] **Task 101:** Create `web/js/utils.js` — utility functions: `formatBytes(bytes)` (B/KB/MB/GB/TB), `formatPercent(value)`, `formatDuration(seconds)`, `formatTimestamp(unix)`, `debounce(fn, ms)`, `throttle(fn, ms)`, `escapeHTML(str)`, `createElement(tag, attrs, children)`

### 7.2 Metrics Dashboard

- [ ] **Task 102:** Create `web/js/charts.js` — implement Canvas-based `Sparkline` class (per IMPLEMENTATION.md §12.3). Constructor takes canvas element and options (maxPoints, color, fillColor). `addPoint(value)` method. Efficient redraw with `requestAnimationFrame`. Also implement `GaugeChart` class (circular gauge showing percentage), `AreaChart` class (larger chart for detail views)
- [ ] **Task 103:** Create `web/js/dashboard.js` — implement system metrics bar. CPU gauge + sparkline, Memory gauge + sparkline, GPU gauge (if available), Disk usage bars, Network upload/download sparklines. Subscribe to `system_metrics` SSE events. Update DOM efficiently (only changed values). Per-core CPU detail view (expandable)

### 7.3 Process Table

- [ ] **Task 104:** Create `web/js/processes.js` — implement process table with virtual scrolling (per IMPLEMENTATION.md §12.2). Column headers: PID, Name, CPU%, Memory, Disk I/O, Network, Status. Click column header to sort (toggle asc/desc). Search input for name filtering. Subscribe to `process_list` SSE events. Diff-update: only update changed cells, not full table rebuild
- [ ] **Task 105:** Add right-click context menu to `processes.js` — implement custom context menu (per SPECIFICATION.md §7.2). Menu items: Kill Process, Kill Process Tree, Suspend/Resume, Set Priority (submenu: Idle → Realtime), Set Affinity (submenu: core checkboxes), Limit CPU (input prompt), Limit Memory (input prompt), Open File Location, View Connections, View Children, Ask AI About, Create Auto Rule. Each action calls appropriate REST API endpoint
- [ ] **Task 106:** Add process detail panel to `processes.js` — clicking a process row expands inline detail: full exe path, creation time, thread count, handle count, page faults, peak memory, I/O rates, connection count, priority class, current limits (if any). Small sparklines for CPU and memory history of selected process

### 7.4 Process Tree View

- [ ] **Task 107:** Create `web/js/tree.js` — implement collapsible tree view. Subscribe to `process_tree` SSE events. Render tree with indentation, expand/collapse toggles, resource summaries per node. Highlight anomaly nodes (spawn storm, orphan) with colored badges. Show aggregate stats for subtrees (total children, total memory). Right-click context menu (same as process table). "Kill Tree" button on parent nodes

### 7.5 Port Monitor

- [ ] **Task 108:** Create `web/js/ports.js` — implement port monitor table. Columns: Port, Process, PID, State, Label, Since. Subscribe to `port_map` SSE events. Highlight conflicts (multiple PIDs) and zombies (stale TIME_WAIT/CLOSE_WAIT) with colored rows. Filter: show all / listeners only / conflicts only. Sort by port number. "Kill" action on zombie processes

### 7.6 Alerts Panel

- [ ] **Task 109:** Create `web/js/alerts.js` — implement alerts panel shown above tabs. Subscribe to `alert_new` and `alert_resolved` SSE events. Color-coded by severity (blue=info, yellow=warning, red=critical, red-flashing=emergency). Each alert shows: severity icon, title, description, age, action buttons (Kill, Suspend, Limit, Ask AI, Dismiss, Snooze). Compact list layout. "Clear All" button. Alert count badge on header

### 7.7 AI Chat Panel

- [ ] **Task 110:** Create `web/js/ai-chat.js` — implement AI chat panel (tab). Text input at bottom, messages above. User messages right-aligned, AI responses left-aligned. Markdown-light rendering for AI responses (bold, lists, code blocks). "Analyze" buttons on alerts integrate with this panel. Action buttons parsed from AI response (Kill, Limit, Create Rule) rendered as clickable buttons that call REST API. Loading indicator during API call. Error display for rate limit or API errors
- [ ] **Task 111:** Create `web/js/settings.js` — implement settings panel (modal or tab). Sections: General (theme toggle, refresh rate), AI (enable/disable, API key input masked, model, language, auto-analyze toggle), Monitoring (intervals), Notifications (balloon enable, min severity). Save via `PUT /api/v1/config`. API key input: show/hide toggle, test button

### 7.8 Interactive Components

- [ ] **Task 112:** Create `web/js/context-menu.js` — reusable custom context menu component. Position near cursor (adjust if near screen edge). Submenu support (priority levels, affinity cores). Keyboard navigation (arrow keys, Enter, Escape). Click-outside-to-close. Prevent default browser context menu on process table
- [ ] **Task 113:** Create `web/icons/*.svg` — create inline SVG icons for: CPU, memory, disk, network, GPU, process, alert (info/warning/critical/emergency), kill, suspend, resume, priority, affinity, limit, AI, settings, search, sort (asc/desc), expand/collapse, refresh, close. Minimal, consistent style

### 7.9 Dashboard Integration

- [ ] **Task 114:** Wire embed.FS in server — ensure `web/` directory is embedded via `//go:embed all:web` directive. Serve via `http.FileServer(http.FS(sub))`. Test that all assets load correctly. Add `Cache-Control` headers for static assets (except index.html which should be no-cache for development)
- [ ] **Task 115:** End-to-end dashboard test — manually verify: open browser → dashboard loads → metrics update in real-time → process table populates → right-click works → kill a test process → alert appears for test scenario → port monitor shows listeners → AI chat sends/receives (if API key configured). Test dark/light theme toggle. Test search and sort

---

## Phase 8: System Tray (Tasks 116-121)

- [ ] **Task 116:** Create `internal/tray/tray.go` — implement system tray per IMPLEMENTATION.md §8.1. Register window class, create hidden message window (`HWND_MESSAGE`), create default icon, `Shell_NotifyIconW(NIM_ADD)`. Message pump on locked OS thread. Handle `WM_LBUTTONDBLCLK` (open dashboard), `WM_RBUTTONUP` (show context menu). Graceful cleanup with `NIM_DELETE` on exit
- [ ] **Task 117:** Create `internal/tray/icon.go` — implement dynamic tray icon generation using GDI. Create 16x16 and 32x32 icons showing mini CPU/MEM gauge (two small colored bars). Green → yellow → red gradient based on usage. `CreateCompatibleDC` → `CreateCompatibleBitmap` → draw with GDI → `CreateIconIndirect`. Update icon when metrics change (throttle to max 1 update/second)
- [ ] **Task 118:** Create `internal/tray/notification.go` — implement balloon notification manager. `ShowBalloon(title, text, severity)` using `Shell_NotifyIconW(NIM_MODIFY)` with `NIF_INFO` flag. Map severity to icon (`NIIF_WARNING`, `NIIF_ERROR`). Rate limiter: max 1 balloon per `balloon_rate_limit` (default 30s). Queue pending notifications
- [ ] **Task 119:** Implement tray context menu — `CreatePopupMenu` → `AppendMenuW` for: Open Dashboard, separator, CPU/MEM/GPU info (disabled items), separator, Alerts count, separator, Pause Monitoring toggle, Settings, separator, Exit. `TrackPopupMenu` at cursor position. Handle `WM_COMMAND` for menu item selection
- [ ] **Task 120:** Wire tray into main.go — initialize tray in `main()`, subscribe to `system_metrics` events for tooltip updates (CPU: X% | MEM: Y%), subscribe to `alert_new` for balloon notifications, tray OnOpen → open browser, OnExit → cancel context → graceful shutdown
- [ ] **Task 121:** Test tray on Windows — manually verify: tray icon appears, tooltip shows metrics, right-click menu works, double-click opens browser, balloon notification appears on test alert, icon color changes with load, exit menu item shuts down cleanly

---

## Phase 9: Build, Polish & Testing (Tasks 122-127)

- [ ] **Task 122:** Create `scripts/build.ps1` — PowerShell build script per IMPLEMENTATION.md §13.3. Get version from git tag, get commit hash. Build with ldflags: `-s -w -H=windowsgui` + version/commit/buildTime. Print output size. Optional `-race` flag for development builds
- [ ] **Task 123:** Create `go.mod` and verify dependencies — ensure only `golang.org/x/sys` and `gopkg.in/yaml.v3` are in `go.mod`. Run `go mod tidy`. Verify no indirect dependencies beyond what these two bring. Document any indirect deps
- [ ] **Task 124:** Create `README.md` — project description, features list, single-binary installation instructions (download + run), screenshot placeholder, configuration guide, build from source instructions, API documentation link, keyboard shortcuts, known limitations, license (MIT)
- [ ] **Task 125:** Create `configs/default.yaml` — full default configuration file matching SPECIFICATION.md §10.2. Well-commented with explanations for each setting. Include common well-known ports for developer tools
- [ ] **Task 126:** Full integration testing — build binary, run on Windows, verify: startup without config (creates default), system tray appears, dashboard accessible, all metrics populating, process kill works, CPU/memory limiting works, anomaly detection fires on simulated scenarios (spin up CPU-heavy child processes, spawn many processes), port monitor shows dev server ports, AI chat works with valid API key, config hot-reload works, graceful shutdown
- [ ] **Task 127:** Performance profiling — verify dashboard doesn't freeze with 500+ processes, SSE doesn't leak goroutines, ring buffer memory usage stays bounded, process collection completes within interval, no handle leaks (check with Process Explorer), CPU usage of WindowsTaskManager itself < 2% at idle

---

## Task Dependency Graph

```
Phase 1 (Foundation)
  Tasks 1-10 (WinAPI) ─────────────────────────────┐
  Tasks 11-14 (Stats) ──────────────┐               │
  Tasks 15-17 (Config) ─────────────┤               │
  Tasks 18-22 (Core) ───────────────┤               │
                                    │               │
Phase 2 (Collectors)                │               │
  Tasks 23-41 ──────────────────────┴───────────────┤
    depends on: WinAPI, Stats, Core                 │
                                                    │
Phase 3 (Controller)                                │
  Tasks 42-52 ──────────────────────────────────────┤
    depends on: WinAPI                              │
                                                    │
Phase 4 (Anomaly)                                   │
  Tasks 53-68 ──────────────────────────────────────┤
    depends on: Stats, Collectors, Core             │
                                                    │
Phase 5 (HTTP/API)                                  │
  Tasks 69-86 ──────────────────────────────────────┤
    depends on: Collectors, Controller, Anomaly     │
                                                    │
Phase 6 (AI)                                        │
  Tasks 87-96 ──────────────────────────────────────┤
    depends on: Core (HTTP client only)             │
                                                    │
Phase 7 (Dashboard)                                 │
  Tasks 97-115 ─────────────────────────────────────┤
    depends on: HTTP/API                            │
                                                    │
Phase 8 (Tray)                                      │
  Tasks 116-121 ────────────────────────────────────┤
    depends on: WinAPI, Core                        │
                                                    │
Phase 9 (Polish)                                    │
  Tasks 122-127 ────────────────────────────────────┘
    depends on: ALL above
```

---

## Estimated Timeline

| Phase | Tasks | Duration | Parallel? |
|-------|-------|----------|-----------|
| 1. Foundation | 1-22 | 5-7 days | — |
| 2. Collectors | 23-41 | 5-7 days | — |
| 3. Controller | 42-52 | 3-4 days | Can overlap with Phase 4 |
| 4. Anomaly | 53-68 | 5-6 days | Can overlap with Phase 3 |
| 5. HTTP/API | 69-86 | 5-6 days | — |
| 6. AI Advisor | 87-96 | 3-4 days | Can overlap with Phase 7 |
| 7. Dashboard | 97-115 | 7-8 days | Can overlap with Phase 6 |
| 8. System Tray | 116-121 | 3-4 days | Can overlap with Phase 7 |
| 9. Polish | 122-127 | 3-4 days | — |
| **Total** | **127** | **~8-10 weeks** | |
