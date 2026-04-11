# WindowsTaskManager — SPECIFICATION

**Repository:** `github.com/ersinkoc/WindowsTaskManager`
**Language:** Go (pure, zero external dependencies beyond allowed list)
**Platform:** Windows 10/11 (amd64)
**License:** MIT
**Author:** Ersin Koç

---

## 1. Project Overview

WindowsTaskManager is an advanced, AI-powered Windows task manager and system monitor built as a single Go binary. It provides real-time monitoring of CPU, memory, GPU, disk, and network resources at both system and per-process levels, with intelligent anomaly detection, process control capabilities (kill, limit, suspend), port tracking, and an optional AI advisor powered by Anthropic Claude API.

The application runs as a lightweight system tray application with an embedded web dashboard (WebView2 or browser-based) that delivers a modern, responsive, non-blocking user experience.

### Core Philosophy

- **Single binary deployment** — no installers, no runtimes, no DLLs
- **Zero external dependencies** — only `golang.org/x/sys`, `golang.org/x/crypto` (if needed), `gopkg.in/yaml.v3`
- **Pure Windows API** — all metrics collected via direct syscall/Win32 API, no WMI, no PowerShell, no CGo
- **Non-blocking architecture** — dashboard must never freeze; all heavy operations are async
- **Privacy-first AI** — AI advisor is opt-in, API key stored locally, no telemetry

---

## 2. Architecture

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    WindowsTaskManager                        │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  │
│  │ Collector │  │Controller│  │ Anomaly  │  │ AI Advisor │  │
│  │  Engine   │  │  Engine  │  │  Engine  │  │ (Optional) │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └─────┬──────┘  │
│       │              │             │               │         │
│  ┌────▼──────────────▼─────────────▼───────────────▼──────┐  │
│  │                   Core Data Bus                        │  │
│  │          (Ring Buffer + Event Emitter)                  │  │
│  └────────────────────────┬───────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────▼───────────────────────────────┐  │
│  │              HTTP Server (localhost)                    │  │
│  │         REST API + SSE (Server-Sent Events)            │  │
│  └────────────────────────┬───────────────────────────────┘  │
│                           │                                  │
│  ┌────────────────────────▼───────────────────────────────┐  │
│  │          Web Dashboard (embed.FS)                      │  │
│  │     Vanilla HTML + CSS + JavaScript + Canvas           │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌────────────────────────────────────────────────────────┐  │
│  │          System Tray (Shell_NotifyIcon)                │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| Collector Engine | Gathers system and per-process metrics via Windows API at configurable intervals |
| Controller Engine | Executes process actions: kill, suspend/resume, set priority, set affinity, CPU/memory limiting via Job Objects |
| Anomaly Engine | Detects spawn storms, memory leaks, hung processes, orphans, port conflicts, runaway CPU, network anomalies using statistical heuristics |
| AI Advisor | Optional Anthropic Claude API integration for natural-language system analysis, root cause diagnosis, and actionable recommendations |
| Core Data Bus | In-memory ring buffer storing metric history + event emitter for real-time push |
| HTTP Server | `net/http` based, serves embedded dashboard + REST API + SSE stream |
| Web Dashboard | Vanilla JS single-page application with real-time charts, process tree, port monitor, AI chat panel |
| System Tray | Native Windows tray icon with context menu and balloon notifications |

### 2.3 Allowed Dependencies

| Dependency | Purpose | Justification |
|------------|---------|---------------|
| `golang.org/x/sys` | Windows syscall wrappers, registry access | Required for safe Win32 API access |
| `gopkg.in/yaml.v3` | Configuration and rules file parsing | Config format |
| Standard library | Everything else | `net/http`, `embed`, `encoding/json`, `sync`, `context`, etc. |

**No other dependencies.** No CGo. No WMI. No PowerShell exec. No third-party GUI frameworks.

---

## 3. Collector Engine

### 3.1 System-Level Metrics

#### 3.1.1 CPU

- **API:** `GetSystemTimes()` → idle, kernel, user times
- **Calculation:** Compare two snapshots at interval, compute usage percentage
- **Per-core:** `NtQuerySystemInformation(SystemProcessorPerformanceInformation)` for per-logical-processor usage
- **Frequency:** `CallNtPowerInformation(ProcessorInformation)` for current/max MHz
- **Data:**
  - Total CPU usage (%)
  - Per-core usage (%) array
  - Current frequency (MHz)
  - Logical processor count
  - CPU name (from registry: `HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0\ProcessorNameString`)

#### 3.1.2 Memory

- **API:** `GlobalMemoryStatusEx()`
- **Data:**
  - Total physical memory
  - Available physical memory
  - Used physical memory + percentage
  - Commit charge (total, limit, peak)
  - Page file usage

#### 3.1.3 Disk

- **Free space:** `GetDiskFreeSpaceExW()` per drive letter
- **I/O counters:** `DeviceIoControl(IOCTL_DISK_PERFORMANCE)` per physical disk
- **Data per drive:**
  - Drive letter, label, filesystem type
  - Total / free / used space + percentage
  - Read/write bytes per second (delta between snapshots)
  - Read/write IOPS (delta between snapshots)
  - Queue depth

#### 3.1.4 Network

- **Interface stats:** `GetIfTable2()` / `GetIfEntry2()` for all network adapters
- **Data per interface:**
  - Interface name, type (Ethernet/WiFi/Loopback), status (up/down)
  - Bytes sent/received per second (delta)
  - Packets sent/received per second
  - Errors, discards
  - Speed (link speed)
- **Aggregate:** Total upload/download bandwidth across all active interfaces

#### 3.1.5 GPU

- **Primary API:** `D3DKMTQueryStatistics` (works for all GPU vendors: Intel, AMD, NVIDIA)
  - GPU engine utilization per node (3D, Compute, Video Decode, Video Encode, Copy)
  - VRAM usage (dedicated + shared)
  - GPU adapter information
- **Optional NVIDIA:** NVML via `nvml.dll` LoadLibrary + GetProcAddress (no CGo)
  - Temperature
  - Fan speed
  - Power draw
  - Clock speeds
- **Fallback:** If no GPU API available, show "N/A" gracefully
- **Data:**
  - GPU name
  - GPU utilization (%)
  - VRAM used / total
  - Temperature (if available)

### 3.2 Per-Process Metrics

#### 3.2.1 Process Enumeration

- **API:** `CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS)` → `Process32FirstW` / `Process32NextW`
- **Per-process detail via `OpenProcess()`:**
  - `QueryFullProcessImageNameW()` → full executable path
  - `GetProcessTimes()` → creation time, kernel time, user time → compute CPU %
  - `GetProcessMemoryInfo()` → WorkingSetSize, PrivateUsage, PageFaultCount, PeakWorkingSetSize
  - `GetProcessIoCounters()` → ReadBytes, WriteBytes, ReadOperationCount, WriteOperationCount (delta)
  - `IsProcessCritical()` → whether it's a critical system process (protect from accidental kill)

#### 3.2.2 Process Tree

- **Source:** `PROCESSENTRY32.th32ParentProcessID` from Toolhelp32 snapshot
- Build full parent → child tree structure
- Detect orphans: child whose `th32ParentProcessID` no longer exists in current snapshot
- Tree used for cascade kill, spawn storm detection, and UI tree view

#### 3.2.3 Per-Process Network (Connections)

- **API:** `GetExtendedTcpTable(AF_INET/AF_INET6, TCP_TABLE_OWNER_PID_ALL)` and `GetExtendedUdpTable(AF_INET/AF_INET6, UDP_TABLE_OWNER_PID)`
- **Data per connection:**
  - Protocol (TCP/UDP)
  - Local address:port
  - Remote address:port (TCP only)
  - State (LISTEN, ESTABLISHED, TIME_WAIT, CLOSE_WAIT, etc.)
  - Owning PID
- **Aggregation:** connection count per PID, listening ports per PID

#### 3.2.4 Port Map

- Derived from per-process network data
- Map of `port → PID → process name → state`
- Well-known port labeling from configuration
- Conflict detection: multiple PIDs on same port (different states)

### 3.3 Collection Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `interval` | 1000ms | Main collection tick (system + process metrics) |
| `process_tree_interval` | 2000ms | Process tree rebuild interval |
| `port_scan_interval` | 3000ms | Port/connection scan interval |
| `gpu_interval` | 2000ms | GPU metrics interval (GPU APIs can be slower) |
| `history_duration` | 10m | How long to keep per-metric history in ring buffer |
| `max_processes` | 2000 | Maximum processes to track (safety limit) |

### 3.4 Ring Buffer Storage

- In-memory circular buffer per metric type
- Stores timestamped data points for `history_duration`
- Lock-free single-writer, multi-reader design using atomic index
- Used by: Anomaly Engine (statistical analysis), Dashboard (sparkline charts), AI Advisor (context)

---

## 4. Controller Engine

### 4.1 Process Kill

- **API:** `OpenProcess(PROCESS_TERMINATE)` → `TerminateProcess(exitCode=1)`
- **Tree Kill:** Enumerate all descendants via process tree, kill bottom-up (children first, then parent)
- **Safety:** Refuse to kill PID 0, PID 4 (System), csrss.exe, wininit.exe, and any process flagged by `IsProcessCritical()`
- **Protected list:** Configurable list of executable names that cannot be killed (e.g., `svchost.exe`, `lsass.exe`)

### 4.2 Process Suspend / Resume

- **Suspend:** Enumerate all threads of target process via `CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD)`, call `SuspendThread()` on each
- **Resume:** Same enumeration, call `ResumeThread()` on each
- **Use case:** Temporarily freeze a runaway process without killing it

### 4.3 Priority Control

- **API:** `SetPriorityClass(hProcess, priorityClass)`
- **Levels:** Idle (0x40), BelowNormal (0x4000), Normal (0x20), AboveNormal (0x8000), High (0x80), Realtime (0x100)
- **Safety:** Warn user before setting Realtime (can hang system)

### 4.4 CPU Affinity

- **API:** `SetProcessAffinityMask(hProcess, mask)`
- **UI:** Show checkboxes for each logical core, user selects which cores the process can use
- **Use case:** Pin a noisy process to specific cores, freeing others

### 4.5 CPU Limiting via Job Objects

- **Mechanism:** Create a Job Object, assign target process, configure CPU rate limiting
- **API sequence:**
  1. `CreateJobObjectW()` → create job
  2. `AssignProcessToJobObject(job, hProcess)` → assign process
  3. `SetInformationJobObject(job, JobObjectCpuRateControlInformation)` with `JOB_OBJECT_CPU_RATE_CONTROL_ENABLE | JOB_OBJECT_CPU_RATE_CONTROL_HARD_CAP`
  4. `CpuRate` field: percentage × 100 (e.g., 30% = 3000)
- **Granularity:** 1-100% in 1% steps
- **Note:** Job Object CPU rate is per-job, applies across all processes in the job

### 4.6 Memory Limiting via Job Objects

- **API:** `SetInformationJobObject(job, JobObjectExtendedLimitInformation)` with `JOB_OBJECT_LIMIT_PROCESS_MEMORY`
- **Field:** `ProcessMemoryLimit` in bytes
- **Behavior:** When process exceeds limit, its memory allocations fail (process may crash gracefully)

### 4.7 Process Count Limiting via Job Objects

- **API:** `SetInformationJobObject(job, JobObjectBasicLimitInformation)` with `JOB_OBJECT_LIMIT_ACTIVE_PROCESS`
- **Field:** `ActiveProcessLimit` — maximum child processes allowed
- **Use case:** Prevent fork bombs (vitest spawn storm scenario)
- **Note:** Assign parent process to job; all children inherit the job

### 4.8 I/O Rate Limiting

- **API:** `SetIoRateControlInformationJobObject()` (Windows 10+)
- **Fields:** `MaxBandwidth` (bytes/sec), `MaxIops`
- **Note:** Only available on NTFS volumes, Windows 10 1607+

---

## 5. Anomaly Detection Engine

### 5.1 Architecture

The Anomaly Engine runs as a separate goroutine, consuming data from the ring buffer at a configurable analysis interval (default: 2 seconds). It applies multiple independent detectors, each producing alerts with severity levels.

```
Ring Buffer → [Detector Pipeline] → Alert Queue → Dashboard + Tray Notification
                 │
                 ├─ Spawn Storm Detector
                 ├─ Memory Leak Detector
                 ├─ Hung Process Detector
                 ├─ Orphan Detector
                 ├─ Runaway CPU Detector
                 ├─ Port Conflict Detector
                 ├─ Network Anomaly Detector
                 └─ New Process Detector
```

### 5.2 Alert Severity Levels

| Level | Meaning | UI Treatment |
|-------|---------|-------------|
| INFO | Noteworthy event, no action needed | Blue indicator, log only |
| WARNING | Potential issue, user should review | Yellow indicator, dashboard badge |
| CRITICAL | Significant resource impact, action recommended | Red indicator, tray balloon notification |
| EMERGENCY | System stability at risk, immediate action needed | Red flashing, persistent notification, optional auto-action |

### 5.3 Spawn Storm Detector

**Scenario:** A process (e.g., vitest, webpack, npm) spawns many child processes rapidly, consuming system resources.

**Detection logic:**
1. Track process creation events via periodic tree diff (new PIDs between snapshots)
2. Group new processes by parent PID and executable name
3. If a single parent spawns > `max_children_per_minute` (default: 20) children in rolling 60s window → CRITICAL alert
4. If total children of a single parent exceeds `max_total_children` (default: 50) → EMERGENCY alert

**Contextual data:**
- Parent process name and PID
- Number of children spawned
- Total resource consumption of all children (CPU %, memory)
- Rate of spawning (children/second)

**Configurable actions:**
- `alert` — notify only (default)
- `suspend_children` — suspend all new children
- `kill_children` — kill all children, preserve parent
- `limit_children` — apply Job Object with `ActiveProcessLimit`

### 5.4 Memory Leak Detector

**Detection logic:**
1. For each process, maintain rolling window of memory samples (WorkingSetSize) over `window` (default: 5 minutes)
2. Apply Welford's online algorithm for rolling mean and standard deviation
3. Perform simple linear regression (least squares) on the window:
   - Compute slope (bytes/second growth rate)
   - Compute R² (goodness of fit)
4. If slope > `min_growth_rate` (default: 10MB/min) AND R² > `min_r_squared` (default: 0.8) → WARNING
5. If total memory exceeds `memory_threshold` (default: 2GB) with positive slope → CRITICAL

**Contextual data:**
- Process name and PID
- Starting memory, current memory, growth rate
- Projected time until a configurable threshold (e.g., "at this rate, will consume 8GB in 20 minutes")
- R² confidence value

### 5.5 Hung Process Detector

**Detection logic:**
1. Track per-process CPU time delta and I/O delta between collection intervals
2. If a process has CPU delta = 0 AND I/O delta = 0 for > `zero_activity_threshold` (default: 120 seconds):
   - If process has visible windows: call `IsHungAppWindow()` via User32 API
   - If no windows or API confirms hung → WARNING alert
3. If hung duration exceeds `critical_hung_threshold` (default: 300 seconds) → CRITICAL

**Exclusions:** System processes, services that legitimately idle (configured in `idle_whitelist`)

**Contextual data:**
- Process name and PID
- Duration of inactivity
- Memory still held
- Whether process has windows (GUI app vs background)

### 5.6 Orphan Detector

**Detection logic:**
1. On each process tree rebuild, check each process's `ParentProcessID`
2. If parent PID does not exist in current snapshot AND process was not a top-level process at startup → orphan detected
3. If orphan is consuming significant resources (CPU > 1% or Memory > 100MB) → WARNING
4. If orphan has children of its own (orphan chain) → CRITICAL

**Contextual data:**
- Orphan process name, PID, original parent PID
- Resources consumed
- Number of orphan's own children
- How long parent has been gone

### 5.7 Runaway CPU Detector

**Detection logic:**
1. Track per-process CPU % over rolling window
2. If CPU > `cpu_threshold` (default: 90%) continuously for > `duration_threshold` (default: 60 seconds) → WARNING
3. If duration exceeds `critical_duration` (default: 180 seconds) → CRITICAL

**Exclusions:** Processes in `high_cpu_whitelist` (e.g., video encoding, compilation — configurable)

### 5.8 Port Conflict Detector

**Detection logic:**
1. On each port scan, build port → PID mapping
2. Detect scenarios:
   - **Zombie port hold:** Port in TIME_WAIT/CLOSE_WAIT state for > `port_zombie_threshold` (default: 120 seconds)
   - **Port conflict:** Multiple PIDs associated with same port (e.g., one LISTENING, one TIME_WAIT)
   - **Port hijack:** A well-known port previously held by process A is now held by process B (different executable)
   - **Stale listener:** Process is LISTENING on a port but process itself is hung/zombie

**Contextual data:**
- Port number, well-known label (if any)
- Current holder PID and process name
- Previous holder (if hijack detected)
- Connection state and duration

### 5.9 Network Anomaly Detector

**Detection logic:**
1. Track per-process connection count over time (rolling mean + stddev)
2. If current connection count > mean + 3σ → WARNING (possible connection flood)
3. Track per-interface bandwidth; if sudden spike > 5× rolling average → WARNING
4. Track total system connections; if > `max_system_connections` (default: 10000) → CRITICAL

### 5.10 New Process Detector

**Detection logic:**
1. Maintain set of known executable paths (built from first scan + learned over time)
2. When a new executable path appears that was never seen before → INFO alert
3. If new executable is in a temp directory, downloads folder, or other suspicious location → WARNING

**Purpose:** Awareness of newly installed or launched software. Not a security tool, but useful for noticing unexpected programs.

### 5.11 Statistical Utilities

All detectors share common statistical primitives implemented from scratch (no external dependency):

- **Welford's Online Algorithm:** Incremental mean + variance computation, O(1) per update
- **Simple Linear Regression:** Least squares slope + R² computation over sliding window
- **Exponential Moving Average (EMA):** For smoothing noisy metrics
- **Rolling Window:** Fixed-size sliding window with O(1) add/remove

---

## 6. AI Advisor

### 6.1 Overview

The AI Advisor is an **optional** module that connects to the Anthropic Claude API to provide:

1. **Automatic analysis** of critical anomalies (when enabled)
2. **On-demand analysis** via "Analyze" button on any alert or process
3. **Interactive chat** where user can ask natural-language questions about system state

### 6.2 Configuration

```yaml
ai:
  enabled: false
  api_key: ""                          # entered via dashboard settings
  model: "claude-sonnet-4-20250514"
  endpoint: "https://api.anthropic.com/v1/messages"
  auto_analyze_on_critical: true       # auto-send critical alerts to AI
  max_tokens: 1024                     # max response length
  max_requests_per_minute: 5           # rate limiting
  language: "tr"                       # response language (tr/en)
  include_process_tree: true           # include tree in context
  include_port_map: true               # include port map in context
  history_context_duration: 5m         # how much history to include
```

API key is stored in the local config file only. Never transmitted anywhere except Anthropic API. No telemetry.

### 6.3 Context Building

When the AI is invoked, the following context is assembled:

```
System: WindowsTaskManager AI Advisor
You are a Windows system expert. Analyze the following system state and provide:
1. Root cause analysis
2. Impact assessment  
3. Actionable recommendations
4. Specific commands or config changes if applicable
Respond in {language}. Be concise and practical.

---

SYSTEM METRICS (current):
- CPU: {total}% ({cores} cores)
- Memory: {used}/{total} ({percent}%)
- GPU: {gpu_name} {util}% VRAM {vram_used}/{vram_total}
- Disk: {per_drive_summary}
- Network: ↑{upload}/s ↓{download}/s

ANOMALY ALERTS (active):
{formatted_alert_list}

PROCESS TREE (top consumers):
{formatted_process_tree}

PORT MAP (listening + conflicts):
{formatted_port_map}

RESOURCE HISTORY (last 5 minutes):
{per_process_trends}

USER QUESTION (if any):
{user_question}
```

### 6.4 API Communication

- Standard `net/http` POST to Anthropic Messages API
- Headers: `x-api-key`, `anthropic-version: 2023-06-01`, `content-type: application/json`
- Request body: standard Messages API format with `model`, `max_tokens`, `system`, `messages`
- Response parsing: extract `content[0].text` from response JSON
- Error handling: timeout (30s), rate limit (429 → backoff), auth error (401 → notify user), network error → graceful fallback

### 6.5 Response Caching

- Cache AI responses keyed by anomaly type + process name + severity
- TTL: 5 minutes (don't re-analyze same issue repeatedly)
- Cache invalidated when anomaly state changes significantly

### 6.6 AI Chat Panel

The dashboard includes an optional chat panel where users can ask free-form questions:

- "vitest neden bu kadar process açtı?"
- "3000 portunu kim kullanıyor ve neden yanıt vermiyor?"
- "chrome neden 4GB RAM yiyor?"
- "sistem neden yavaşladı?"
- "node.exe processlerini analiz et"

Each chat message includes current system snapshot as context. Conversation history maintained in-memory (last 10 messages) for multi-turn.

### 6.7 AI-Suggested Actions

AI responses may include actionable suggestions. The dashboard parses these and presents them as buttons:

- "Kill PID 1234" → [Kill Process] button
- "Set priority to Below Normal" → [Set Priority] button
- "Create auto-kill rule" → [Create Rule] button
- "Limit CPU to 30%" → [Apply Limit] button

User must explicitly click to execute; AI never auto-executes actions.

---

## 7. Web Dashboard

### 7.1 Technology

- **Embedding:** All web assets compiled into binary via Go `embed.FS`
- **Framework:** None. Vanilla HTML5 + CSS3 + JavaScript (ES2020+)
- **Charts:** Canvas API (custom sparklines, gauges, area charts)
- **Icons:** Inline SVG (no icon font dependencies)
- **Real-time updates:** Server-Sent Events (SSE) — simpler than WebSocket, one-way push
- **User actions:** Standard HTTP POST/PUT/DELETE to REST API

### 7.2 Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  WindowsTaskManager                    [⚙ Settings] [🤖 AI On] │
├─────────────┬──────────────┬──────────────┬────────────────────┤
│  CPU 62%    │  MEM 48%     │  GPU 35%     │  DISK C: R/W       │
│  ▁▃▅▇▆▄▃▅  │  ▂▃▃▄▅▆▇▇   │  ▁▂▃▂▁▂▃▂   │  ▁▁▂▅▃▁▁▂          │
│  12 cores   │  7.6/16 GB   │  RTX 3080    │  NET ↑2.3 ↓15.1    │
├─────────────┴──────────────┴──────────────┴────────────────────┤
│  ⚠ ALERTS (3)                                      [Clear All] │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ 🔴 vitest.exe — Spawn storm: 47 children, 8.2GB total    │  │
│  │    [Kill Tree] [Suspend] [Limit] [🤖 Ask AI]             │  │
│  │ 🟡 chrome.exe — Memory leak: +340MB/5min, R²=0.92        │  │
│  │    [Kill] [Limit Memory] [🤖 Ask AI]                     │  │
│  │ 🟡 :3000 — Port held by zombie node.exe (TIME_WAIT 3min) │  │
│  │    [Kill PID 5600]                                        │  │
│  └───────────────────────────────────────────────────────────┘  │
├────────────────────────────────────────────────────────────────┤
│  TABS: [Processes] [Tree View] [Ports] [🤖 AI Chat]           │
├────────────────────────────────────────────────────────────────┤
│  PROCESSES                                                     │
│  [Search: ________] [Filter: All ▼] [Sort: CPU% ▼]            │
│  ┌──────┬──────────────┬──────┬────────┬────────┬─────┬──────┐ │
│  │ PID  │ Name         │ CPU% │ Memory │ Disk   │ Net │ Stat │ │
│  ├──────┼──────────────┼──────┼────────┼────────┼─────┼──────┤ │
│  │ 4521 │ vitest.exe   │ 45%  │ 8.2GB  │ 12MB/s │ 0   │ ⚠🔴 │ │
│  │ 1100 │ chrome.exe   │ 12%  │ 2.1GB  │ 1MB/s  │ 5.2 │ ⚠🟡 │ │
│  │ 2200 │ code.exe     │  8%  │ 620MB  │ 0.3    │ 0.1 │ ✓   │ │
│  │ 3400 │ node.exe     │  3%  │ 180MB  │ 0.1    │ 0.5 │ ✓   │ │
│  └──────┴──────────────┴──────┴────────┴────────┴─────┴──────┘ │
│                                                                │
│  Right-click menu:                                             │
│  ┌─────────────────────┐                                       │
│  │ Kill Process        │                                       │
│  │ Kill Process Tree   │                                       │
│  │ Suspend / Resume    │                                       │
│  │ ─────────────────── │                                       │
│  │ Set Priority     ▸  │                                       │
│  │ Set Affinity     ▸  │                                       │
│  │ Limit CPU (%)    ▸  │                                       │
│  │ Limit Memory     ▸  │                                       │
│  │ ─────────────────── │                                       │
│  │ Open File Location  │                                       │
│  │ View Connections    │                                       │
│  │ View Children       │                                       │
│  │ 🤖 Ask AI About...  │                                       │
│  │ Create Auto Rule    │                                       │
│  └─────────────────────┘                                       │
├────────────────────────────────────────────────────────────────┤
│  TREE VIEW                                                     │
│  explorer.exe (PID 100)                                        │
│  ├─ cmd.exe (PID 200)                                          │
│  │  └─ node.exe (PID 3200) — 45MB                              │
│  │     └─ vitest.exe (PID 4521) ⚠ SPAWN STORM                 │
│  │        ├─ node.exe (PID 4530) — 180MB [hung]                │
│  │        ├─ node.exe (PID 4531) — 175MB [hung]                │
│  │        └─ ... (45 more, total 8.2GB)                        │
│  ├─ code.exe (PID 2200) — 620MB                                │
│  └─ chrome.exe (PID 1100) — 2.1GB ⚠ LEAK                      │
│     ├─ chrome.exe (PID 1101) — 340MB                           │
│     ├─ chrome.exe (PID 1102) — 280MB                           │
│     └─ chrome.exe (PID 1103) — 520MB                           │
├────────────────────────────────────────────────────────────────┤
│  PORT MONITOR                                                  │
│  ┌───────┬──────────────┬───────┬──────────┬─────────┬────────┐│
│  │ Port  │ Process      │ PID   │ State    │ Label   │ Since  ││
│  ├───────┼──────────────┼───────┼──────────┼─────────┼────────┤│
│  │ :80   │ nginx.exe    │ 1200  │ LISTEN   │ HTTP    │ 2h     ││
│  │ :443  │ nginx.exe    │ 1200  │ LISTEN   │ HTTPS   │ 2h     ││
│  │ :3000 │ node.exe     │ 3400  │ LISTEN   │ DevSrv  │ 15m    ││
│  │ :3000 │ ⚠ node.exe   │ 5600  │ TIME_WAIT│ Zombie  │ 3m     ││
│  │ :5173 │ node.exe     │ 4521  │ LISTEN   │ Vite    │ 5m     ││
│  │ :5432 │ postgres.exe │ 800   │ LISTEN   │ PgSQL   │ 3h     ││
│  │ :8080 │ java.exe     │ 2200  │ LISTEN   │ Spring  │ 1h     ││
│  └───────┴──────────────┴───────┴──────────┴─────────┴────────┘│
├────────────────────────────────────────────────────────────────┤
│  🤖 AI CHAT                                                    │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ You: vitest neden bu kadar process açtı?                   │ │
│  │                                                            │ │
│  │ AI: vitest --pool=forks modunda her test dosyası için      │ │
│  │ ayrı worker fork'luyor. 47 test dosyanız var ve hiçbiri    │ │
│  │ tamamlanmadı. Öneriler:                                    │ │
│  │ 1. pool: 'threads' kullanın                                │ │
│  │ 2. maxWorkers: 4 ile sınırlayın                            │ │
│  │ 3. testTimeout: 30000 ekleyin                              │ │
│  │                                                            │ │
│  │ [Kill All Children] [Create Rule]                          │ │
│  └────────────────────────────────────────────────────────────┘ │
│  [______________________________________________] [Send]       │
└────────────────────────────────────────────────────────────────┘
```

### 7.3 Dashboard Performance Requirements

- **Frame rate:** Dashboard updates at 1-2 FPS (not 60 FPS — we're showing data, not gaming)
- **DOM updates:** Only update changed values (diff-based), never rebuild entire tables
- **Virtualized list:** Process table only renders visible rows (virtual scrolling for 1000+ processes)
- **Chart rendering:** Canvas-based sparklines/gauges, reuse canvas elements
- **SSE reconnect:** Auto-reconnect with exponential backoff on connection loss
- **Memory:** Dashboard JavaScript should use < 50MB heap

### 7.4 Themes

- Light and Dark theme support via CSS custom properties
- Theme preference saved in localStorage
- System theme auto-detection via `prefers-color-scheme`

---

## 8. System Tray

### 8.1 Implementation

- **API:** `Shell_NotifyIconW` with `NIF_ICON | NIF_MESSAGE | NIF_TIP`
- **Icon:** Embedded ICO resource showing mini CPU/MEM gauge (updated dynamically via GDI)
- **Tooltip:** "WindowsTaskManager — CPU: 62% | MEM: 48%"
- **Window:** Hidden message-only window (`CreateWindowExW` with HWND_MESSAGE parent) for tray messages

### 8.2 Context Menu

```
┌─────────────────────┐
│ Open Dashboard      │  → Launch browser / WebView to localhost
│ ─────────────────── │
│ CPU: 62%            │  (disabled, info only)
│ MEM: 7.6 / 16 GB   │  (disabled, info only)
│ GPU: 35%            │  (disabled, info only)
│ ─────────────────── │
│ ⚠ Alerts (3)        │  → Open dashboard alerts tab
│ ─────────────────── │
│ Pause Monitoring    │  → Toggle data collection
│ Settings            │  → Open dashboard settings
│ ─────────────────── │
│ Exit                │  → Graceful shutdown
└─────────────────────┘
```

### 8.3 Balloon Notifications

- Triggered on CRITICAL and EMERGENCY alerts
- Shows: alert title + brief description
- Click on balloon → opens dashboard to relevant alert
- Rate-limited: max 1 balloon per 30 seconds to avoid spam

---

## 9. HTTP Server & API

### 9.1 Server Configuration

- **Bind:** `127.0.0.1:{port}` (localhost only, configurable, default: 19876)
- **Static files:** `embed.FS` serving dashboard from `/`
- **API prefix:** `/api/v1/`
- **SSE endpoint:** `/api/v1/events`
- **CORS:** Not needed (same-origin)

### 9.2 REST API Endpoints

#### System Metrics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/system` | Current system metrics (CPU, MEM, GPU, Disk, Net) |
| GET | `/api/v1/system/history?duration=5m` | Historical system metrics |
| GET | `/api/v1/system/cpu/cores` | Per-core CPU usage |

#### Processes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/processes` | All processes with metrics |
| GET | `/api/v1/processes?sort=cpu&order=desc&limit=50` | Sorted/filtered process list |
| GET | `/api/v1/processes/{pid}` | Single process detail |
| GET | `/api/v1/processes/{pid}/history` | Process metric history |
| GET | `/api/v1/processes/{pid}/connections` | Process network connections |
| GET | `/api/v1/processes/{pid}/children` | Process children |
| GET | `/api/v1/processes/tree` | Full process tree |

#### Process Actions

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/processes/{pid}/kill` | Kill process |
| POST | `/api/v1/processes/{pid}/kill-tree` | Kill process tree |
| POST | `/api/v1/processes/{pid}/suspend` | Suspend process |
| POST | `/api/v1/processes/{pid}/resume` | Resume process |
| PUT | `/api/v1/processes/{pid}/priority` | Set priority (`{"level": "below_normal"}`) |
| PUT | `/api/v1/processes/{pid}/affinity` | Set affinity (`{"mask": "0x0F"}`) |
| PUT | `/api/v1/processes/{pid}/cpu-limit` | Set CPU limit (`{"percent": 30}`) |
| PUT | `/api/v1/processes/{pid}/memory-limit` | Set memory limit (`{"bytes": 1073741824}`) |
| DELETE | `/api/v1/processes/{pid}/limits` | Remove all limits (detach from Job Object) |

#### Ports

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/ports` | All listening ports with PID mapping |
| GET | `/api/v1/ports/{port}` | Detail for specific port |
| GET | `/api/v1/connections` | All TCP/UDP connections |

#### Alerts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/alerts` | Active alerts |
| GET | `/api/v1/alerts/history` | Alert history |
| POST | `/api/v1/alerts/{id}/dismiss` | Dismiss alert |
| POST | `/api/v1/alerts/{id}/snooze?duration=30m` | Snooze alert |

#### AI Advisor

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/ai/analyze` | Analyze current state or specific alert (`{"alert_id": "..."}`) |
| POST | `/api/v1/ai/chat` | Send chat message (`{"message": "..."}`) |
| GET | `/api/v1/ai/status` | AI advisor status (enabled, API key set, last call, rate limit) |
| PUT | `/api/v1/ai/config` | Update AI config (`{"api_key": "...", "language": "tr"}`) |

#### Rules

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/rules` | All anomaly detection rules |
| PUT | `/api/v1/rules` | Update rules |
| POST | `/api/v1/rules/reload` | Hot-reload rules from YAML |

#### Configuration

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/config` | Current configuration |
| PUT | `/api/v1/config` | Update configuration |

### 9.3 SSE Events

Single SSE endpoint `/api/v1/events` pushes all real-time updates:

| Event Type | Payload | Frequency |
|------------|---------|-----------|
| `system_metrics` | System CPU/MEM/GPU/Disk/Net snapshot | Every 1s |
| `process_list` | Full process list with metrics | Every 1s |
| `process_tree` | Process tree structure | Every 2s |
| `port_map` | Port → PID mapping | Every 3s |
| `alert_new` | New alert created | On event |
| `alert_resolved` | Alert auto-resolved | On event |
| `ai_response` | AI analysis result (streamed) | On event |

Client can filter events via query parameter: `/api/v1/events?types=system_metrics,alert_new`

---

## 10. Configuration

### 10.1 Config File Location

- Default: `%APPDATA%\WindowsTaskManager\config.yaml`
- Override via flag: `--config path/to/config.yaml`
- If file doesn't exist, create with defaults on first run

### 10.2 Configuration Schema

```yaml
# WindowsTaskManager Configuration

server:
  host: "127.0.0.1"
  port: 19876
  open_browser: true          # auto-open dashboard on start

monitoring:
  interval: 1000ms            # main collection tick
  process_tree_interval: 2000ms
  port_scan_interval: 3000ms
  gpu_interval: 2000ms
  history_duration: 10m
  max_processes: 2000

controller:
  protected_processes:         # cannot be killed via UI
    - "csrss.exe"
    - "wininit.exe"
    - "lsass.exe"
    - "smss.exe"
    - "services.exe"
    - "svchost.exe"
    - "winlogon.exe"
  confirm_kill_system: true    # require confirmation for system processes

anomaly:
  analysis_interval: 2000ms   # how often detectors run

  spawn_storm:
    enabled: true
    max_children_per_minute: 20
    max_total_children: 50
    action: "alert"

  memory_leak:
    enabled: true
    window: 5m
    min_growth_rate: "10MB/min"
    min_r_squared: 0.8
    memory_threshold: "2GB"
    action: "alert"

  hung_process:
    enabled: true
    zero_activity_threshold: 120s
    critical_hung_threshold: 300s
    idle_whitelist:
      - "SearchIndexer.exe"
      - "spoolsv.exe"
    action: "alert"

  orphan:
    enabled: true
    resource_threshold_cpu: 1       # percent
    resource_threshold_memory: "100MB"
    action: "alert"

  runaway_cpu:
    enabled: true
    cpu_threshold: 90               # percent
    duration_threshold: 60s
    critical_duration: 180s
    high_cpu_whitelist: []
    action: "alert"

  port_conflict:
    enabled: true
    time_wait_threshold: 120s
    close_wait_threshold: 60s
    action: "alert"

  network_anomaly:
    enabled: true
    connection_sigma: 3             # standard deviations
    max_system_connections: 10000
    action: "alert"

  new_process:
    enabled: true
    suspicious_paths:
      - "%TEMP%"
      - "%USERPROFILE%\\Downloads"
    action: "info"

notifications:
  tray_balloon: true
  balloon_rate_limit: 30s
  balloon_min_severity: "critical"

well_known_ports:
  80: "HTTP Server"
  443: "HTTPS Server"
  3000: "Dev Server (React/Next.js)"
  3001: "Dev Server Alt"
  4200: "Angular Dev Server"
  5173: "Vite Dev Server"
  5432: "PostgreSQL"
  6379: "Redis / CacheStorm"
  8080: "HTTP Alt / Spring Boot / Tomcat"
  8443: "HTTPS Alt"
  9090: "Prometheus / Grafana"
  27017: "MongoDB / Mammoth"

ai:
  enabled: false
  api_key: ""
  model: "claude-sonnet-4-20250514"
  endpoint: "https://api.anthropic.com/v1/messages"
  auto_analyze_on_critical: true
  max_tokens: 1024
  max_requests_per_minute: 5
  language: "tr"
  include_process_tree: true
  include_port_map: true
  history_context_duration: 5m

ui:
  theme: "system"               # light | dark | system
  default_sort: "cpu"
  default_sort_order: "desc"
  sparkline_points: 60          # data points in sparklines
  process_table_page_size: 100
  refresh_rate: 1000ms          # dashboard update rate
```

### 10.3 Hot-Reload

- Monitor config file via `ReadDirectoryChangesW()` (native file watcher, no fsnotify)
- On change: validate YAML, apply non-destructive changes without restart
- Changes to `server.port` require restart (logged as warning)
- Changes to `anomaly.*` rules take effect immediately

---

## 11. Project Structure

```
WindowsTaskManager/
├── cmd/
│   └── wtm/
│       └── main.go                   # Entry point, flag parsing, bootstrap
├── internal/
│   ├── collector/
│   │   ├── collector.go              # Collector orchestrator
│   │   ├── cpu.go                    # CPU metrics (system + per-core)
│   │   ├── memory.go                 # Memory metrics
│   │   ├── disk.go                   # Disk space + I/O
│   │   ├── network.go                # Interface stats
│   │   ├── gpu.go                    # GPU via D3DKMT + optional NVML
│   │   ├── process.go                # Process enumeration + metrics
│   │   ├── process_tree.go           # Parent-child tree builder
│   │   └── ports.go                  # TCP/UDP port → PID mapping
│   ├── controller/
│   │   ├── killer.go                 # Kill + tree kill
│   │   ├── suspender.go              # Suspend / resume
│   │   ├── priority.go               # Priority class
│   │   ├── affinity.go               # CPU affinity mask
│   │   ├── limiter.go                # CPU + Memory + I/O + Process count limits via Job Objects
│   │   └── safety.go                 # Protected process checks
│   ├── anomaly/
│   │   ├── engine.go                 # Anomaly engine orchestrator
│   │   ├── alert.go                  # Alert types, severity, lifecycle
│   │   ├── spawnstorm.go             # Spawn storm / fork bomb detector
│   │   ├── memleak.go                # Memory leak detector
│   │   ├── hung.go                   # Hung process detector
│   │   ├── orphan.go                 # Orphan process detector
│   │   ├── runaway.go                # Runaway CPU detector
│   │   ├── portconflict.go           # Port conflict / zombie detector
│   │   ├── netanomaly.go             # Network anomaly detector
│   │   ├── newprocess.go             # New process detector
│   │   └── rules.go                  # Rule loading + hot-reload
│   ├── stats/
│   │   ├── welford.go                # Welford's online mean/variance
│   │   ├── regression.go             # Linear regression (least squares)
│   │   ├── ema.go                    # Exponential moving average
│   │   └── ringbuf.go               # Generic ring buffer
│   ├── ai/
│   │   ├── advisor.go                # Anthropic API client
│   │   ├── prompt.go                 # System snapshot → prompt builder
│   │   ├── parser.go                 # Response parser (extract suggestions)
│   │   ├── cache.go                  # Response cache
│   │   └── ratelimit.go             # API rate limiter
│   ├── server/
│   │   ├── server.go                 # HTTP server bootstrap
│   │   ├── router.go                 # Request routing (no third-party router)
│   │   ├── sse.go                    # Server-Sent Events handler
│   │   ├── api_system.go             # /api/v1/system handlers
│   │   ├── api_process.go            # /api/v1/processes handlers
│   │   ├── api_ports.go              # /api/v1/ports handlers
│   │   ├── api_alerts.go             # /api/v1/alerts handlers
│   │   ├── api_ai.go                 # /api/v1/ai handlers
│   │   ├── api_rules.go             # /api/v1/rules handlers
│   │   ├── api_config.go             # /api/v1/config handlers
│   │   └── middleware.go             # Logging, recovery, admin-only check
│   ├── tray/
│   │   ├── tray.go                   # System tray icon + menu
│   │   ├── icon.go                   # Dynamic icon generation (GDI)
│   │   └── notification.go           # Balloon notification manager
│   ├── config/
│   │   ├── config.go                 # Config struct + defaults
│   │   ├── loader.go                 # YAML load + validate
│   │   └── watcher.go               # File watcher (ReadDirectoryChangesW)
│   ├── winapi/
│   │   ├── kernel32.go               # kernel32.dll procs
│   │   ├── ntdll.go                  # ntdll.dll procs
│   │   ├── psapi.go                  # psapi.dll procs
│   │   ├── iphlpapi.go              # iphlpapi.dll procs
│   │   ├── user32.go                 # user32.dll procs
│   │   ├── shell32.go               # shell32.dll procs
│   │   ├── gdi32.go                  # gdi32.dll procs
│   │   ├── d3dkmt.go                # gdi32.dll D3DKMT procs (GPU)
│   │   ├── advapi32.go              # advapi32.dll procs (Job Objects)
│   │   └── types.go                  # Windows struct definitions
│   └── platform/
│       └── elevation.go              # Admin privilege check + self-elevate
├── web/
│   ├── index.html                    # Main dashboard page
│   ├── css/
│   │   ├── style.css                 # Main stylesheet
│   │   ├── theme-light.css           # Light theme variables
│   │   └── theme-dark.css            # Dark theme variables
│   ├── js/
│   │   ├── app.js                    # Application bootstrap + SSE
│   │   ├── dashboard.js              # System metrics panel
│   │   ├── processes.js              # Process table (virtual scroll)
│   │   ├── tree.js                   # Process tree view
│   │   ├── ports.js                  # Port monitor table
│   │   ├── alerts.js                 # Alert panel
│   │   ├── ai-chat.js               # AI chat panel
│   │   ├── charts.js                # Canvas sparklines + gauges
│   │   ├── context-menu.js          # Right-click context menu
│   │   ├── settings.js              # Settings panel
│   │   └── utils.js                 # Formatters, helpers
│   └── icons/
│       └── *.svg                     # Inline SVG icons
├── configs/
│   └── default.yaml                  # Default configuration
├── scripts/
│   └── build.ps1                     # Build script (go build + embed + version)
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── SPECIFICATION.md                  # This file
├── IMPLEMENTATION.md                 # Implementation details (generated next)
└── TASKS.md                          # Task breakdown (generated next)
```

---

## 12. Build & Distribution

### 12.1 Build

```powershell
# Standard build
go build -ldflags="-s -w -H=windowsgui" -o WindowsTaskManager.exe ./cmd/wtm/

# With version info
go build -ldflags="-s -w -H=windowsgui -X main.version=1.0.0 -X main.commit=$(git rev-parse --short HEAD)" -o WindowsTaskManager.exe ./cmd/wtm/
```

- `-H=windowsgui` — no console window on launch
- `-s -w` — strip debug info for smaller binary
- Expected binary size: ~8-15MB (mostly embed.FS web assets)

### 12.2 Requirements

- **OS:** Windows 10 1607+ / Windows 11
- **Privileges:** Administrator recommended (required for: GPU metrics, kill system processes, Job Objects on other users' processes). Non-admin mode works with reduced capabilities.
- **Runtime dependencies:** None (single binary)

### 12.3 Admin Elevation

- On startup, check if running as admin via `IsUserAnAdmin()` or token check
- If not admin: show tray notification "Running with limited capabilities. Right-click → Run as Administrator for full access."
- Specific features requiring admin are individually feature-gated and show appropriate messages

---

## 13. Security Considerations

- **Localhost only:** HTTP server binds to `127.0.0.1`, never `0.0.0.0`
- **No authentication:** Since localhost-only, no auth needed (same trust model as Windows Task Manager)
- **API key storage:** AI API key stored in config YAML on local filesystem with standard file permissions
- **Process actions:** All destructive actions (kill, limit) require explicit user interaction (no auto-kill by default)
- **Protected processes:** Critical system processes cannot be killed via UI
- **No telemetry:** Zero data sent anywhere except optional Anthropic API (user-initiated only)
- **No auto-update:** User downloads new versions manually from GitHub releases

---

## 14. Non-Goals (Explicit Exclusions)

- **Not a security/antivirus tool:** No malware detection, no file scanning
- **Not a service manager:** No Windows Service start/stop/install (may add later)
- **Not a performance profiler:** No CPU profiling, no memory heap analysis, no flame graphs
- **Not cross-platform:** Windows only. No Linux/macOS support planned
- **Not a remote management tool:** Localhost only. No remote access, no multi-machine
- **No installer:** Single binary, no MSI/NSIS/WiX
- **No auto-update mechanism:** Manual GitHub releases
- **No plugin system:** Monolithic single binary (may reconsider later)

---

## 15. Future Considerations (Post-v1.0)

These are explicitly out of scope for v1.0 but may be added later:

- Windows Service management (start/stop/restart services)
- Startup program management
- Event Log viewer integration
- Performance counter integration (PDH API)
- ETW (Event Tracing for Windows) integration for deeper process monitoring
- WebView2 COM integration for native window (instead of browser)
- Export metrics to file (CSV/JSON)
- Custom dashboard layouts (drag-and-drop panels)
- Process grouping / tagging
- Historical data persistence (SQLite)
- Multiple AI provider support (OpenAI, local LLM via Ollama)
