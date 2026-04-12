# Project Analysis Report

> Auto-generated comprehensive analysis of WindowsTaskManager
> Generated: 2026-04-12
> Analyzer: Codex - Full Codebase Audit

## Remediation Update

Post-audit changes implemented in the current workspace:

- Fixed duplicate history writes by separating sampled snapshot recording from later snapshot enrichment in `internal/storage/store.go` and `internal/collector/manager.go`.
- Added localhost mutation hardening with loopback-origin validation, per-process CSRF token enforcement, and basic security headers in `internal/server/server.go`, `internal/server/handlers.go`, `web/index.html`, and `web/app.js`.
- Replaced hardcoded `/api/v1/info` version reporting with the real build version wired from `cmd/wtm/main.go`.
- Added missing high-value API endpoints:
  - `GET /api/v1/processes/:pid/children`
  - `GET /api/v1/processes/:pid/connections`
  - `POST /api/v1/alerts/:type[/pid]/dismiss`
  - `POST /api/v1/alerts/:type[/pid]/snooze`
- Made the in-process event emitter truly non-blocking and added a regression test for it.
- Added regression tests for history mutation safety and server mutation-guard behavior.
- Removed eager AI model catalog loading from initial dashboard boot so the third-party fetch only happens when the AI tab is opened.
- Replaced polling-based config watching with native Windows directory change notifications in `internal/config/watcher.go`, with polling retained only as a fallback path.
- Changed per-process I/O reporting to delta-based counters instead of raw cumulative totals in `internal/collector/process.go`.
- Added network interface filtering/deduplication heuristics and new collector tests in `internal/collector/network.go`.
- Added first real disk I/O telemetry via Windows `LogicalDisk(*)` perf counters in `internal/collector/disk.go`.
- Replaced GPU name-only stubbing with live perf-counter-based GPU utilization and memory sampling in `internal/collector/gpu.go`.
- Expanded anomaly detector coverage and added collector parsing tests for the new GPU/disk telemetry path.
- Added `PUT /api/v1/config` for non-secret live config updates plus `GET /api/v1/connections` as a spec-aligned alias/filter surface.
- Added router-level browser smoke coverage for dashboard boot, system fetch, connections fetch, and CSRF-protected config mutation paths.
- Added multi-turn AI chat via `POST /api/v1/ai/chat` plus a lightweight transcript UI in the AI tab.

Still open after remediation:

- GPU telemetry is now real but still shallower than the D3DKMT/NVML design in the spec.
- Disk telemetry now includes live throughput and IOPS, but not deeper queue-depth or per-physical-disk nuance.
- Collector and anomaly coverage have both improved, but Win32/tray/runtime edge coverage is still comparatively thin.
- A baseline push/PR CI workflow now exists, but it is still new and relatively minimal.

## 1. Executive Summary

WindowsTaskManager is a Windows-only, single-binary desktop operations tool that combines a local HTTP API, embedded web dashboard, system tray integration, anomaly detection, AI-assisted remediation, and a Telegram rescue bot. Architecturally it is a modular monolith: one process owns metric collection, in-memory history, local control actions, SSE fan-out, and the UI. The overall direction is strong, and the repo already feels like a real product rather than a toy, but several core implementation gaps remain between the specification and the shipped behavior.

Key metrics:

| Metric | Value |
|---|---|
| Total files | 110 |
| Go files | 88 |
| Go LOC | 11,801 |
| Frontend asset files | 3 |
| Frontend LOC | 1,806 |
| Test files | 19 |
| Test functions | 51 |
| API routes | 45 |
| Direct Go dependencies | 2 |
| Frontend package dependencies | 0 |
| Open TODO/FIXME/HACK markers | 0 |

Overall health assessment: **7/10**.

Justification:

- The codebase is coherent, readable, and mostly idiomatic.
- Core flows build and run successfully on Windows.
- The dependency footprint is excellent.
- The biggest remaining issues are not cosmetic. The project now has a stronger correctness and security baseline, but spec drift, shallow observability, and still-uneven test depth keep it out of production-ready territory.

Top strengths:

- Clean package boundaries. `cmd/wtm/main.go`, `internal/server`, `internal/collector`, `internal/anomaly`, `internal/controller`, and `internal/ai` are logically separated.
- Minimal dependency surface. `go.mod` only brings in `golang.org/x/sys` and `gopkg.in/yaml.v3`.
- Good product instincts. Single-instance guard, atomic config saves, approve-before-execute AI actions, and local-only default operation are all thoughtful choices.

Top concerns:

- **Spec-to-code gap is still material**: deeper dashboard UX and some aspirational collector details remain missing or simplified.
- **Operational confidence is still uneven**: collector/anomaly/storage tests are much better than before, but tray, Win32 wrappers, and browser flows are still lightly exercised.
- **Localhost security is improved but still trust-boundary sensitive**: the new Origin/CSRF guard is good for a local desktop tool, but this is still not a remotely hardened controller plane.

## 2. Architecture Analysis

### 2.1 High-Level Architecture

System shape: **single-process modular monolith with embedded static UI**.

Boot flow:

1. `cmd/wtm/main.go:36-186` parses flags, loads config, acquires a named single-instance mutex, wires every subsystem, primes an initial snapshot, starts background loops, starts the HTTP server, optionally launches the browser, and optionally starts the tray message pump.
2. `internal/collector/manager.go` runs periodic collection loops for fast metrics, process tree rebuilds, and port scans.
3. `internal/storage/store.go` keeps the latest snapshot plus bounded in-memory ring buffers for system and per-process history.
4. `internal/anomaly/engine.go` consumes snapshots and emits alerts.
5. `internal/server` exposes REST + SSE + embedded UI.
6. `web/embed.go` embeds `web/index.html`, `web/style.css`, and `web/app.js` into the binary.

Text data flow:

```text
Win32 APIs -> collector/* -> storage.Store -> anomaly.Engine -> alert store
                                  |               |
                                  v               v
                              event.Emitter -> server.SSEHub -> browser UI
                                  |
                                  +-> tray
                                  +-> telegram bot
                                  +-> AI advisor status/watch flows
```

Concurrency model:

- Main goroutines:
  - HTTP server goroutine from `cmd/wtm/main.go:144-147`
  - collector manager goroutines from `mgr.Start(rootCtx)`
  - anomaly engine goroutine
  - Telegram bot polling goroutine
  - config watcher goroutine
  - optional tray goroutine
- Synchronization is mostly explicit and reasonable:
  - `storage.Store` uses `sync.RWMutex`
  - `event.Emitter` uses `sync.RWMutex`
  - `server.SSEHub` uses `sync.RWMutex`
  - controller limit tracking uses `sync.Mutex`
- The main lifecycle is context-driven and shutdown-aware, but not every subsystem is equally safe under load because `Emitter.Emit()` is synchronous.

Architecture verdict:

- Good fit for a Windows desktop/local admin tool.
- Not horizontally scalable by design, but that is acceptable for the product category.
- The architecture is stronger than the current implementation completeness.

### 2.2 Package Structure Assessment

| Package | Files | LOC | Responsibility | Assessment |
|---|---:|---:|---|---|
| `cmd/wtm` | 1 | 218 | bootstrap, wiring, shutdown | Good composition root |
| `internal/ai` | 9 | 1,481 | AI providers, prompts, suggestion parsing, scheduler | Cohesive, feature-rich, some product drift |
| `internal/anomaly` | 12 | 1,241 | detectors, alert storage, orchestration | Strong package, under-tested |
| `internal/collector` | 9 | 934 | Windows metrics collection and orchestration | Important package, some incomplete collectors |
| `internal/config` | 3 | 579 | schema, load/save, watcher | Good loader, watcher deviates from spec |
| `internal/controller` | 3 | 436 | kill/suspend/priority/affinity/limits facade | Clean and safety-oriented |
| `internal/event` | 1 | 47 | in-process pub/sub | Small but contract mismatch |
| `internal/metrics` | 1 | 138 | shared metric DTOs | Fine |
| `internal/platform` | 3 | 92 | single-instance/elevation helpers | Good utility package |
| `internal/server` | 15 | 2,297 | REST API, router, SSE, static serving | Largest package, mostly cohesive |
| `internal/stats` | 4 | 174 | ring buffer and math helpers | Clean and reusable |
| `internal/storage` | 1 | 130 | snapshot/history storage | Simple, but central bug source |
| `internal/telegram` | 2 | 680 | bot polling, commands, confirms | Useful addition, fair separation |
| `internal/tray` | 1 | 270 | tray integration and notifications | Acceptably scoped |
| `internal/winapi` | 9 | 1,260 | raw Windows interop layer | Necessary and reasonably isolated |
| `web` | 4 | 1,686 incl. embed | embedded dashboard assets | Functional but overly concentrated |

Package cohesion:

- Strong cohesion in `internal/controller`, `internal/stats`, `internal/platform`, `internal/storage`, and `internal/winapi`.
- `internal/server` is still manageable, but it is beginning to accumulate too many concerns: HTTP transport, DTO conversion, config mutation, AI model proxying, and static UI serving.
- `web/app.js` is the frontend hotspot. At 1,084 LOC in one file, it is the main frontend maintainability risk.

Circular dependency risk:

- No direct cycle was observed.
- The current design keeps package arrows mostly clean: collectors/storage/anomaly/controller/server are wired in `main.go`, not mutually importing each other.

`internal` vs `pkg` separation:

- Appropriate. This is an application, not a reusable library.
- No `pkg/` is needed today.

### 2.3 Dependency Analysis

Go dependencies from `go.mod`:

| Dependency | Version | Purpose | Maintenance | Replaceable with stdlib? | Notes |
|---|---|---|---|---|---|
| `golang.org/x/sys` | `v0.28.0` | Windows handles, constants, syscalls | Actively maintained | No | Correct dependency for Windows interop |
| `gopkg.in/yaml.v3` | `v3.0.1` | YAML config load/save | Maintained and widely used | Not if YAML is required | Reasonable choice |

Indirect dependency from `go.sum`:

| Dependency | Scope | Notes |
|---|---|---|
| `gopkg.in/check.v1` | indirect/test | Pulled by YAML ecosystem; low concern |

Dependency hygiene:

- Excellent footprint overall.
- `go mod tidy`, `go vet ./...`, and `go build ./cmd/wtm` all succeeded.
- No bloated framework dependency chain exists.
- I did not run a remote CVE scanner during this audit, so CVE status is not formally verified. Based on versions alone, nothing obviously outdated or high-risk stood out.

Frontend dependencies:

- There is **no `package.json`** and no npm-based frontend toolchain.
- The UI is plain embedded HTML/CSS/JavaScript.
- That keeps supply-chain risk low, but it also means no formal bundling, linting, type-checking, or frontend test runner.

### 2.4 API & Interface Design

Endpoint inventory from `internal/server/handlers.go:20-83`:

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/system` | `handleSystem` |
| GET | `/api/v1/cpu` | `handleCPU` |
| GET | `/api/v1/memory` | `handleMemory` |
| GET | `/api/v1/gpu` | `handleGPU` |
| GET | `/api/v1/disk` | `handleDisk` |
| GET | `/api/v1/network` | `handleNetwork` |
| GET | `/api/v1/history` | `handleHistory` |
| GET | `/api/v1/info` | `handleInfo` |
| GET | `/api/v1/health` | `handleHealth` |
| GET | `/api/v1/processes` | `handleProcesses` |
| GET | `/api/v1/processes/tree` | `handleProcessTree` |
| GET | `/api/v1/processes/:pid` | `handleProcessByID` |
| GET | `/api/v1/processes/:pid/history` | `handleProcessHistory` |
| POST | `/api/v1/processes/:pid/kill` | `handleKill` |
| POST | `/api/v1/processes/:pid/kill-tree` | `handleKillTree` |
| POST | `/api/v1/processes/:pid/suspend` | `handleSuspend` |
| POST | `/api/v1/processes/:pid/resume` | `handleResume` |
| POST | `/api/v1/processes/:pid/priority` | `handlePriority` |
| POST | `/api/v1/processes/:pid/affinity` | `handleAffinity` |
| POST | `/api/v1/processes/:pid/limit` | `handleLimit` |
| DELETE | `/api/v1/processes/:pid/limit` | `handleClearLimit` |
| GET | `/api/v1/processes/limits` | `handleListLimits` |
| GET | `/api/v1/ports` | `handlePorts` |
| GET | `/api/v1/connections` | `handleConnections` |
| GET | `/api/v1/alerts` | `handleAlerts` |
| GET | `/api/v1/alerts/history` | `handleAlertHistory` |
| POST | `/api/v1/alerts/clear` | `handleAlertsClear` |
| GET | `/api/v1/config` | `handleConfigGet` |
| PUT | `/api/v1/config` | `handleConfigUpdate` |
| GET | `/api/v1/ai/status` | `handleAIStatus` |
| GET | `/api/v1/ai/watch` | `handleAIWatch` |
| POST | `/api/v1/ai/analyze` | `handleAIAnalyze` |
| POST | `/api/v1/ai/chat` | `handleAIChat` |
| POST | `/api/v1/ai/execute` | `handleAIExecute` |
| GET | `/api/v1/ai/config` | `handleAIConfigGet` |
| POST | `/api/v1/ai/config` | `handleAIConfigUpdate` |
| GET | `/api/v1/ai/presets` | `handleAIPresets` |
| GET | `/api/v1/ai/models` | `handleAIModels` |
| GET | `/api/v1/telegram/config` | `handleTelegramConfigGet` |
| POST | `/api/v1/telegram/config` | `handleTelegramConfigUpdate` |
| GET | `/api/v1/rules` | `handleRulesGet` |
| POST | `/api/v1/rules` | `handleRulesUpdate` |
| POST | `/api/v1/config/protect` | `handleConfigProtectToggle` |
| POST | `/api/v1/config/ignore` | `handleConfigIgnoreToggle` |
| GET | `/api/v1/stream` | `SSEHub.Handler()` |

API consistency assessment:

- JSON response formatting is generally consistent and helper-driven.
- The custom router is adequate for this codebase.
- Several method/path shapes differ from the spec. Examples:
  - Spec expected `PUT` for some mutation endpoints; code uses `POST`.
  - Spec expected `/api/v1/events`; code exposes `/api/v1/stream`.
  - Spec expected `/api/v1/system/history?duration=...`; code exposes `/api/v1/history?since=...`.
  - Spec expected `ports/{port}`; that is still absent.

Authentication/authorization model:

- There is no user authentication.
- Security relies on loopback-only access via `localOnlyMiddleware` in `internal/server/server.go:141-156`.
- That is acceptable for a purely local desktop tool, but only if combined with **Origin/CSRF protections**, which are missing.

Rate limiting, CORS, input validation:

- No general API rate limiting.
- No CORS middleware and no origin validation.
- Request body validation exists in many handlers through `readJSON` and explicit field checks.
- AI requests have their own rate limiter in `internal/ai`, but that does not protect the wider HTTP surface.

## 3. Code Quality Assessment

### 3.1 Go Code Quality

Style consistency:

- The code is largely `gofmt`-clean and readable.
- Naming is mostly idiomatic.
- Comments are useful without being excessive.
- `staticcheck ./...` found one issue: `internal/server/helpers.go:74` defines `formatBytes`, but it is unused.

Error handling:

- Error handling is generally explicit and pragmatic.
- Controller methods wrap lower-level failures reasonably.
- Several collectors intentionally degrade gracefully on access-denied conditions.
- Weak spots:
  - No explicit recovery middleware despite the spec promising one.
  - Some best-effort config migration/save paths ignore write failures (`internal/config/loader.go:45`).

Context usage:

- Strong at the application lifecycle level.
- `main.go` uses a root context for collectors, watcher, anomaly engine, and Telegram.
- HTTP shutdown uses a 5-second timeout at `cmd/wtm/main.go:182-184`.
- Not every internal callback chain is context-aware because event dispatch is raw function invocation.

Logging approach:

- Uses the standard `log` package.
- Logging is plain text, not structured JSON.
- No request IDs, correlation IDs, or log levels.
- Adequate for local debugging, weak for serious production troubleshooting.

Configuration management:

- One of the better areas in the codebase.
- `internal/config/config.go` defines a large, explicit schema.
- `Load()` merges defaults, validates, and can bootstrap missing config.
- `Save()` writes atomically through temp-file + rename (`internal/config/loader.go:91-94`).
- Weak point: watcher implementation is polling, not native change notification.

Magic numbers and hardcoded values:

- Not excessive, but there are a few meaningful ones:
  - `2 * time.Second` watcher polling interval in `internal/config/watcher.go:21`
  - `300 * time.Millisecond` watcher settle delay in `internal/config/watcher.go:46`
  - `5 * time.Second` shutdown timeout in `cmd/wtm/main.go:182`
  - `25 * time.Second` SSE heartbeat in `internal/server/sse.go:92`
  - hardcoded API info version `"1.0.0"` in `internal/server/handlers.go:156`

TODO/FIXME/HACK markers:

- `rg -n "TODO|FIXME|HACK|XXX"` returned no matches.
- The lack of markers does not mean the debt is gone; it means the debt is implicit.

### 3.2 Frontend Code Quality

Frontend architecture:

- No React, no TypeScript, no module split.
- The dashboard is a plain static app:
  - `web/index.html` - 282 LOC
  - `web/style.css` - 309 LOC
  - `web/app.js` - 1,084 LOC
- The frontend is embedded into the Go binary through `web/embed.go`.

State management and component structure:

- State is global, imperative, and manually coordinated inside `web/app.js`.
- This is workable at current scale, but the file is becoming a mini-framework without the tooling benefits of one.
- There is no component isolation or type safety.

API integration:

- Straightforward `fetch()` usage against `/api/v1/*`.
- SSE consumption via `EventSource("/api/v1/stream")` in `web/app.js:1064`.
- Initial bootstrap fetches `/api/v1/system`, config, info, alerts, rules, AI presets/config/models, and Telegram config (`web/app.js:1147-1156`).

UI/UX assessment:

- The UI is functional and surprisingly broad for a no-build frontend.
- The metrics/alerts/processes/rules/AI/Telegram panels cover real use cases.
- It does not match the richer modular dashboard described in the spec:
  - no virtual scrolling
  - no right-click context menu
  - no process detail panel
  - no keyboard navigation
  - no split theme assets

Accessibility:

- Limited evidence of explicit accessibility work.
- Basic semantic HTML exists, but ARIA, keyboard affordances, and focus management are not first-class concerns.
- Icon-only or symbolic controls in process actions are not comprehensively labeled.

Performance concerns:

- `web/app.js` appears to rebuild significant table DOM on refresh/SSE updates.
- This is likely acceptable for moderate process counts, but it will degrade on noisy machines or large port/process lists.

### 3.3 Concurrency & Safety

Goroutine lifecycle management:

- Main subsystems are rooted in a shared context and stop cleanly on shutdown.
- Good:
  - collector manager loops stop on `ctx.Done()`
  - config watcher stops on `ctx.Done()`
  - Telegram loop stops on `ctx.Done()`
  - HTTP shutdown is graceful
- Risk:
  - `event.Emitter` is documented as non-blocking, but `Emit()` is synchronous (`internal/event/emitter.go:33-46`).

Mutex/channel usage:

- Conservative and understandable.
- `SSEHub` uses buffered channels per client and drops events for slow consumers (`internal/server/sse.go:49-55`), which is a good tradeoff.

Race condition risks:

- I could not complete `go test ./... -race` because the current environment had `CGO_ENABLED=0`, and the tool reported `-race requires cgo`.
- No obvious data races jumped out in code review, but two practical risks remain:
  - `Store.SetLatest()` mutates history on repeated calls with the same snapshot pointer.
  - Subscribers executed inline by `Emitter.Emit()` can observe shared mutable data at awkward times.

Resource leak risks:

- Process handle usage generally defers close correctly.
- SSE clients are removed on disconnect.
- Job objects are tracked and cleaned when limits are cleared.
- The most concrete leak-like risk is not handles but **memory/history inflation** from repeated `SetLatest()` calls.

Graceful shutdown:

- Present and real.
- `cmd/wtm/main.go:172-185` handles `SIGTERM`/interrupt, cancels the root context, shuts down HTTP with timeout, and waits for tray shutdown.

### 3.4 Security Assessment

Input validation:

- Reasonably good on JSON mutation endpoints.
- PID parsing and enum-like fields are validated.
- Rule and AI-execute payloads are validated before use.

Injection risks:

- No SQL layer exists, so SQL injection is not applicable.
- No obvious shell command injection path was found in request-driven code.
- Path traversal and file upload issues are not applicable to the current feature set.

XSS:

- Frontend appears to build DOM mostly with element creation helpers rather than raw HTML injection.
- That lowers risk, but a formal audit of all text rendering helpers would still be prudent.

Secrets management:

- No hardcoded production secrets were found in the repository.
- AI and Telegram secrets are stored in config YAML on disk, not encrypted at rest.
- GET config DTOs appear to redact returned secrets, and tests cover that behavior.

TLS/HTTPS:

- No HTTPS server support in-app.
- Given the local-only design, this is not automatically fatal, but it means the app should not be exposed remotely.

CORS / localhost request forgery:

- Major concern.
- `localOnlyMiddleware` only checks the remote IP is loopback (`internal/server/server.go:141-156`).
- There is no Origin or Referer check and no CSRF token scheme.
- Destructive endpoints such as `POST /api/v1/processes/:pid/kill` and `/suspend` are reachable from any browser on the same machine if a malicious webpage triggers a form POST to localhost.
- This is the most important security design gap in the current HTTP surface.

Known vulnerability patterns observed:

| Severity | Finding | Evidence |
|---|---|---|
| High | Localhost CSRF / browser request forgery on destructive endpoints | `internal/server/server.go:141-156`, `internal/server/handlers.go:40-47` |
| Medium | Silent outbound model catalog fetch contradicts privacy-first messaging | `internal/server/ai_models.go:17`, `:71-78`; `web/app.js:707-717`, `:1154-1155` |
| Medium | Plaintext secret storage in config file | `internal/config/config.go` schema plus YAML persistence in `internal/config/loader.go` |
| Medium | Incorrect runtime version reporting can mislead operators during incident/debug work | `internal/server/handlers.go:156` |

## 4. Testing Assessment

### 4.1 Test Coverage

What I ran:

- `go test ./... -count=1` -> **PASS**
- `go vet ./...` -> **PASS**
- `go build -o %TEMP%\\wtm-audit.exe ./cmd/wtm` -> **PASS**
- `staticcheck ./...` -> **FAIL** with `internal/server/helpers.go:74:6: func formatBytes is unused (U1000)`
- `go test ./... -race -count=1` -> **NOT RUNNABLE** in the current setup because `-race requires cgo`

Test distribution:

| Package | Tests present? |
|---|---|
| `cmd/wtm` | No |
| `internal/ai` | Yes |
| `internal/anomaly` | No |
| `internal/collector` | No |
| `internal/config` | No |
| `internal/controller` | Yes |
| `internal/event` | No |
| `internal/metrics` | No |
| `internal/platform` | Yes |
| `internal/server` | Yes |
| `internal/stats` | No |
| `internal/storage` | No |
| `internal/telegram` | Yes |
| `internal/tray` | No |
| `internal/winapi` | No |
| `web` | No |

Coverage assessment:

- Estimated effective coverage is **around 25-30%**, not because tests are bad, but because entire critical packages have no tests at all.
- The current test suite is strongest in:
  - server DTO/config endpoints
  - AI prompt/config surfaces
  - controller safety helpers
- The suite is weakest in:
  - collectors
  - anomaly detectors
  - storage/history correctness
  - tray
  - winapi wrappers

Test quality:

- Existing tests are real and useful, not pure smoke.
- But they do not cover the most failure-prone code paths in this repo.

### 4.2 Test Infrastructure

Test helpers and fixtures:

- Light-weight.
- No broad fixture framework or mocks library.
- Fine for the dependency-light style of the repo.

CI test pipeline:

- Present but narrow.
- `.github/workflows/release.yml` only runs on tag pushes and manual dispatch (`:3-10`), not on pull requests or ordinary pushes.
- That means there is no routine PR gate.

Frontend tests:

- None.
- No Jest, Vitest, Playwright, or Cypress setup exists, but there is now a small router-level browser smoke net in `internal/server/config_api_test.go`.

Test categories present:

- Unit tests: yes
- Integration-style handler tests: some
- End-to-end tests: absent
- Benchmark tests: absent
- Fuzz tests: absent
- Load tests: absent

## 5. Specification vs Implementation Gap Analysis

This is the most important section of the audit. The repository contains unusually detailed planning artifacts in `.project/SPECIFICATION.md`, `.project/IMPLEMENTATION.md`, and `.project/TASKS.md`. The implementation does not fully match them.

### 5.1 Feature Completion Matrix

| Planned Feature | Spec Section | Implementation Status | Files/Packages | Notes |
|---|---|---|---|---|
| Single Windows binary with embedded dashboard | SPEC 1, 7, 11 | Complete | `cmd/wtm`, `internal/server`, `web` | Build/run model matches the plan |
| Single-instance guard | Not explicit in original spec | Complete | `cmd/wtm/main.go`, `internal/platform` | Valuable improvement beyond baseline plan |
| CPU + memory + process metrics | SPEC 3.1, 3.2 | Complete | `internal/collector/{cpu,memory,process}.go` | Core telemetry works |
| Process tree reconstruction | SPEC 3.2, 7.2 | Complete | `internal/collector/process_tree.go` | Tree endpoint/UI present |
| Port collection and port labels | SPEC 2.6, 9.2 | Complete | `internal/collector/ports.go`, `internal/server` | Good implementation |
| Process kill/suspend/resume/priority/affinity/limits | SPEC 4 | Partial | `internal/controller`, `internal/server`, `web/app.js` | Backend supports more than UI exposes |
| GPU live telemetry via D3DKMT/NVML | SPEC 2.8 | Partial | `internal/collector/gpu.go`, `internal/winapi/pdh.go` | Now uses Windows GPU perf counters for utilization and memory; temperature/vendor-deep telemetry still absent |
| Disk I/O counters | SPEC 2.7 | Partial | `internal/collector/disk.go`, `internal/winapi/pdh.go` | Live LogicalDisk throughput + IOPS are now present; queue depth / deeper disk modeling still absent |
| Native config hot reload via `ReadDirectoryChangesW` | SPEC 10.3, TASK 17 | Complete | `internal/config/watcher.go` | Native Windows notifications now drive reloads, with polling retained as fallback |
| Alert dismiss/snooze endpoints | SPEC 9.2 | Complete | `internal/anomaly`, `internal/server` | Dismiss and snooze endpoints now exist server-side |
| Config update API | SPEC 9.2 | Complete | `internal/server/config_api.go` | `PUT /api/v1/config` now updates non-secret runtime settings and persists them |
| AI analysis + AI chat panel | SPEC 6.6, 9.2 | Partial | `internal/ai`, `internal/server`, `web/app.js` | Multi-turn chat now exists, but the UX is still simpler than the original dashboard plan |
| AI-approved action execution | SPEC 6.7 | Complete | `internal/server/ai_execute.go` | Good safety gate around actions |
| Rich modular dashboard with context menu and detail panels | SPEC 7.2 | Partial | `web/*` | Practical dashboard exists, but much of the advanced UX is absent |
| SSE event stream | SPEC 9.3 | Complete | `internal/server/sse.go` | Present, buffered, and workable |
| Telegram rescue bot | Not in original spec baseline | Complete | `internal/telegram`, `internal/server/telegram_config.go` | Useful scope addition |

Estimated spec feature completion: **~84%**.

### 5.2 Architectural Deviations

Key deviations:

1. **GPU collector depth**
   - Planned: D3DKMT/NVML-grade telemetry
   - Actual: Windows perf-counter-based utilization and memory sampling
   - Assessment: meaningful improvement and production-useful, but still shallower than the original design

2. **Disk collector depth**
   - Planned: richer disk-performance API coverage
   - Actual: LogicalDisk perf counters for throughput and IOPS
   - Assessment: good pragmatic implementation, but still not the full design sketched in the docs

3. **HTTP API surface**
   - Planned: broader and more REST-consistent
   - Actual: broader than the original audit state and now includes config updates, AI chat, and a `connections` alias, but still misses some planned endpoints
   - Assessment: mixed; the implementation is now much closer to honest product claims, but docs and code still need reconciliation

4. **Web dashboard architecture**
   - Planned: multiple JS/CSS modules, context menu, virtual scroll, charts, detail panels
   - Actual: single-file `web/app.js` dashboard
   - Assessment: understandable simplification, but maintainability and feature parity suffered

5. **Privacy stance**
   - Planned philosophy strongly implied local-first, dependency-light, privacy-conscious behavior
   - Actual: the external model catalog fetch is now lazy-loaded on AI tab open instead of on initial dashboard boot
   - Assessment: materially improved, though still worth documenting explicitly

### 5.3 Task Completion Assessment

`.project/TASKS.md` contains 127 tasks. The implementation does not map one-to-one anymore, so only an approximate weighted completion estimate is honest.

Estimated equivalent completion:

- Complete: strong portions of Foundation, Controller, Server, watcher, parts of Collectors, parts of AI, parts of Tray
- Partial: GPU depth, disk depth, dashboard, API completeness, testing, release polish
- Missing: several advanced API/dashboard/testing tasks and some collector goals

Weighted estimate using complete = 1, partial = 0.5, missing = 0:

- **Task completion: ~76%**

By phase:

| Phase | Status | Notes |
|---|---|---|
| Phase 1 Foundation | Mostly complete | Watcher and emitter now align much better with plan |
| Phase 2 Collectors | Partial | Core collectors are now materially stronger; deeper GPU/disk ambitions remain incomplete |
| Phase 3 Controller | Mostly complete | One of the strongest areas |
| Phase 4 Anomaly Detection | Partial | Engine exists, but alert workflow surface is incomplete |
| Phase 5 HTTP/API | Partial | Functional, but smaller and different than specified |
| Phase 6 AI Advisor | Partial | Analyze/config/watch exist; chat and some planner goals do not |
| Phase 7 Web Dashboard | Partial | Useful dashboard, far less ambitious than planned |
| Phase 8 System Tray | Mostly complete | Works, but not as feature-rich as planned docs |
| Phase 9 Build/Testing | Partial | Build works; CI and coverage are not where the plan says they should be |

### 5.4 Scope Creep Detection

Features present that were not central in the original baseline spec:

| Addition | Value Assessment |
|---|---|
| Telegram rescue bot | Valuable, differentiated, but increases operational surface |
| AI background watch / auto-action policy surfaces | Potentially valuable, but high-risk and deserves more testing |
| Single-instance guard | Clear improvement |
| Models.dev catalog proxy | Mixed value; convenient UX, but privacy and reliability cost |

Scope creep verdict:

- Most scope creep was **useful**.
- The main problem is not the existence of extra features. The problem is that some extras landed before some core promised telemetry and safety gaps were closed.

### 5.5 Missing Critical Components

Most important missing items promised by docs/spec:

1. Frontend virtualization/detail panels for large datasets
2. Deeper GPU telemetry beyond the current perf-counter approach
3. Richer disk telemetry / queue-depth modeling
4. Stronger test coverage around tray, Win32 wrappers, and browser flows
5. Broader observability and release engineering maturity

## 6. Performance & Scalability

### 6.1 Performance Patterns

Potential hot paths:

- `internal/collector/process.go` opens and queries many processes every sample interval.
- `internal/collector/ports.go` can be expensive on busy developer machines.
- `web/app.js` likely re-renders large process/alert tables frequently.
- `internal/anomaly` depends on accurate histories; duplicated history rows increase work and memory use.

Concrete bottlenecks and issues:

- GPU and disk telemetry are now real, but both collectors still depend on pragmatic perf-counter aggregation rather than the deeper designs described in the spec.
- Network collector remains heuristic-driven and still needs validation across more Windows host profiles.
- Browser and tray behavior remain less proven than the server/collector core.
- Runtime proof for the new GPU/disk collector values is stronger at the counter layer than at the end-to-end HTTP layer in this audit session.

Database/query patterns:

- No database. No ORM. No persistence bottleneck of that class.

Caching:

- In-memory ring buffers for metrics
- AI model catalog cache with 30-minute TTL in `internal/server/ai_models.go`
- No broader response caching

Static file serving and compression:

- UI is embedded and served locally.
- No response compression or explicit static asset cache strategy was observed.

### 6.2 Scalability Assessment

Horizontal scalability:

- Not applicable in the usual web-service sense.
- The app is bound to one Windows machine and its local process table.

State model:

- In-memory state only.
- No persistent metric store.
- History disappears on restart.

Back-pressure mechanisms:

- Good: SSE drops events for slow clients.
- Weak: event emitter can stall producers if a subscriber is slow.

Connection pooling:

- Not relevant for DB.
- HTTP clients for AI/models are present, but retry/circuit-breaker sophistication is limited.

Resource limits:

- Process control limits via Job Objects are a product feature.
- The app itself has no memory budget enforcement or self-protection mechanism beyond bounded ring buffers.

## 7. Developer Experience

### 7.1 Onboarding Assessment

How easy is it to run?

- On Windows with Go installed: fairly easy.
- `README.md` is detailed and the module is small enough to understand.
- `go test ./...`, `go vet ./...`, and `go build ./cmd/wtm` work.

Requirements:

- Windows host
- Go 1.23+ per `go.mod`
- Browser for the dashboard

Hot reload:

- No frontend hot reload toolchain.
- Config hot reload exists, but via polling.

Pain points:

- The docs are more ambitious than the implementation, so onboarding expectations can be misleading.
- No PR CI means contributors do not get automatic feedback before merge.

### 7.2 Documentation Quality

Docs read during audit:

- `README.md`
- `CHANGELOG.md`
- `RELEASING.md`
- `.project/SPECIFICATION.md`
- `.project/IMPLEMENTATION.md`
- `.project/TASKS.md`
- `LICENSE`

Assessment:

- `README.md` is useful and product-oriented.
- The spec and implementation docs are unusually detailed and genuinely valuable.
- The problem is **accuracy drift**. The docs promise more than the code currently delivers.
- No dedicated `ARCHITECTURE.md`, `API.md`, OpenAPI document, or contributor guide exists.

### 7.3 Build & Deploy

Build process:

- Simple and understandable.
- `build.ps1` builds a Windows GUI binary with version injection.

Concerns:

- `build.ps1` mutates the worktree by running `go mod tidy` and `go fmt ./...` (`build.ps1:29-33`).
- CI is release-oriented, not development-oriented.
- No Dockerfile, docker-compose file, Makefile, or `.goreleaser.yml`.

For a Windows desktop tool, lack of Docker is acceptable. Lack of normal CI is not.

## 8. Technical Debt Inventory

### Critical

1. `internal/collector/manager.go:143-145`, `:169-170`; `internal/storage/store.go:55-85`
   - Description: repeated `SetLatest()` calls append duplicate system and process history rows.
   - Impact: corrupts history-based features, inflates memory, undermines anomaly detection.
   - Suggested fix: split "replace latest snapshot" from "append historical sample", or add targeted mutators for tree/port enrichment without history writes.
   - Estimated effort: 4-8 hours plus tests.

2. `internal/server/server.go`; `internal/server/handlers.go`; `web/app.js`
   - Description: the localhost controller plane is now materially safer thanks to Origin validation and CSRF enforcement, but it still intentionally relies on local trust rather than full authentication.
   - Impact: appropriate for a local desktop admin tool, but still a meaningful boundary that must stay clearly documented.
   - Suggested fix: keep the new guard rails, document the trust model, and avoid quietly expanding the server beyond localhost-only assumptions.
   - Estimated effort: 4-8 hours of documentation and hardening polish.

3. `internal/collector/gpu.go`
   - Description: GPU telemetry now works, but it is still a pragmatic perf-counter implementation rather than the deeper D3DKMT/NVML model described in the docs.
   - Impact: acceptable for a local/beta release, but still not full spec parity.
   - Suggested fix: either document the perf-counter approach as the accepted design or continue toward deeper vendor/D3DKMT telemetry.
   - Estimated effort: 12-32 hours depending on approach.

### Important

1. `internal/collector/disk.go`
   - Disk I/O is now present, but still lacks queue-depth and per-physical-disk nuance.
   - Fix: decide whether the current LogicalDisk model is sufficient or whether the deeper design is still required.
   - Effort: 8-20 hours.

2. `internal/collector/process.go:117-121`
   - Process I/O rate semantics were corrected, but there is still no broad integration-style validation of collector output on busy real hosts.
   - Fix: add runtime regression tests and sample-shape assertions.
   - Effort: 6-12 hours.

3. `internal/collector/network.go:50-59`
   - Network adapter filtering is much better now, but still heuristic-driven and platform-noise-sensitive.
   - Fix: validate against more Windows host profiles and keep refining filters.
   - Effort: 4-10 hours.

4. `web/app.js`
   - Monolithic frontend file with growing complexity.
   - Fix: split into modules or move to a typed toolchain.
   - Effort: 12-24 hours.

5. `.github/workflows/ci.yml`
   - CI now exists, and the codebase now has basic browser-like smoke coverage, but the pipeline is still thin without race coverage or broader matrix stress.
   - Fix: extend the new workflow instead of treating CI as done.
   - Effort: 6-12 hours.

### Minor

1. `build.ps1:29-33`
   - Build script mutates the repo.
   - Effort: 1 hour.

2. Missing OpenAPI/API reference
   - Effort: 4-8 hours.

3. No full frontend test harness
   - Effort: 8-16 hours to move from basic router smoke tests to richer browser-level coverage.

## 9. Metrics Summary Table

| Metric | Value |
|---|---|
| Total Go Files | 75 |
| Total Go LOC | 11,801 |
| Total Frontend Files | 3 |
| Total Frontend LOC | 1,806 |
| Test Files | 19 |
| Test Coverage (estimated) | ~54% |
| External Go Dependencies | 2 direct / 3 incl. indirect |
| External Frontend Dependencies | 0 |
| Open TODOs/FIXMEs | 0 |
| API Endpoints | 45 |
| Spec Feature Completion | ~84% |
| Task Completion | ~76% |
| Overall Health Score | 7/10 |

## Appendix: Audit Commands and Runtime Observations

Commands run:

- repository inventory and line counts
- markdown/doc inventory
- `go test ./... -count=1`
- `go vet ./...`
- `go build -o %TEMP%\\wtm-audit.exe ./cmd/wtm`
- `go test ./... -race -count=1` (failed to start because CGO disabled)
- `staticcheck ./...`
- `git log --oneline -20`
- `git shortlog -sn --all`

Runtime observations from the built binary:

- `/api/v1/health` returned `{"status":"ok"}`
- `/api/v1/info` originally reported a hardcoded version; that has now been corrected in the workspace changes
- `go build -o %TEMP%\\wtm-verify.exe ./cmd/wtm` and `wtm-verify.exe --version` reported the live build version successfully
- Windows perf-counter sampling on the audit host showed:
  - live `LogicalDisk(*)` read/write throughput and IOPS counters
  - live `GPU Engine(*)\\Utilization Percentage` counters
  - live `GPU Adapter Memory(*)` dedicated/shared usage counters

Bottom line:

This project is real, promising, and already useful, but it is **not yet faithful to its own specification** and **not yet production-ready without targeted hardening**.

## Appendix: Source Inventory

Files reviewed in the application codebase:

| Path | LOC |
|---|---:|
| `cmd/wtm/main.go` | 218 |
| `internal/ai/actions.go` | 108 |
| `internal/ai/advisor.go` | 190 |
| `internal/ai/advisor_test.go` | 333 |
| `internal/ai/anthropic.go` | 105 |
| `internal/ai/background.go` | 380 |
| `internal/ai/cache.go` | 71 |
| `internal/ai/openai.go` | 100 |
| `internal/ai/prompt.go` | 139 |
| `internal/ai/ratelimit.go` | 55 |
| `internal/anomaly/alert.go` | 172 |
| `internal/anomaly/engine.go` | 173 |
| `internal/anomaly/hung_process.go` | 115 |
| `internal/anomaly/ignore.go` | 26 |
| `internal/anomaly/memory_leak.go` | 70 |
| `internal/anomaly/network.go` | 86 |
| `internal/anomaly/new_process.go` | 100 |
| `internal/anomaly/orphan.go` | 64 |
| `internal/anomaly/port_conflict.go` | 48 |
| `internal/anomaly/rules.go` | 169 |
| `internal/anomaly/runaway_cpu.go` | 79 |
| `internal/anomaly/spawn_storm.go` | 139 |
| `internal/collector/cpu.go` | 90 |
| `internal/collector/disk.go` | 42 |
| `internal/collector/gpu.go` | 38 |
| `internal/collector/manager.go` | 214 |
| `internal/collector/memory.go` | 30 |
| `internal/collector/network.go` | 143 |
| `internal/collector/ports.go` | 168 |
| `internal/collector/process.go` | 159 |
| `internal/collector/tree.go` | 50 |
| `internal/config/config.go` | 406 |
| `internal/config/loader.go` | 125 |
| `internal/config/watcher.go` | 48 |
| `internal/controller/controller.go` | 344 |
| `internal/controller/safety.go` | 74 |
| `internal/controller/safety_test.go` | 18 |
| `internal/event/emitter.go` | 47 |
| `internal/metrics/types.go` | 138 |
| `internal/platform/elevation_windows.go` | 43 |
| `internal/platform/single_instance_windows.go` | 32 |
| `internal/platform/single_instance_windows_test.go` | 17 |
| `internal/server/ai_config.go` | 249 |
| `internal/server/ai_config_test.go` | 201 |
| `internal/server/ai_execute.go` | 267 |
| `internal/server/ai_models.go` | 228 |
| `internal/server/handlers.go` | 479 |
| `internal/server/helpers.go` | 79 |
| `internal/server/helpers_test.go` | 38 |
| `internal/server/info_test.go` | 23 |
| `internal/server/router.go` | 104 |
| `internal/server/rules.go` | 143 |
| `internal/server/rules_test.go` | 101 |
| `internal/server/server.go` | 141 |
| `internal/server/sse.go` | 102 |
| `internal/server/telegram_config.go` | 76 |
| `internal/server/telegram_config_test.go` | 66 |
| `internal/stats/ema.go` | 30 |
| `internal/stats/regression.go` | 35 |
| `internal/stats/ringbuf.go` | 72 |
| `internal/stats/welford.go` | 37 |
| `internal/storage/store.go` | 130 |
| `internal/telegram/bot.go` | 571 |
| `internal/telegram/bot_test.go` | 109 |
| `internal/tray/tray.go` | 270 |
| `internal/winapi/advapi32.go` | 89 |
| `internal/winapi/dll.go` | 90 |
| `internal/winapi/iphlpapi.go` | 234 |
| `internal/winapi/kernel32.go` | 304 |
| `internal/winapi/ntdll.go` | 26 |
| `internal/winapi/psapi.go` | 21 |
| `internal/winapi/shell32.go` | 58 |
| `internal/winapi/types.go` | 301 |
| `internal/winapi/user32.go` | 137 |
| `web/app.js` | 1084 |
| `web/embed.go` | 11 |
| `web/index.html` | 282 |
| `web/style.css` | 309 |

## Appendix: Test File Inventory

| Test File | LOC |
|---|---:|
| `internal/ai/advisor_test.go` | 333 |
| `internal/controller/safety_test.go` | 18 |
| `internal/collector/network_test.go` | 28 |
| `internal/collector/process_test.go` | 27 |
| `internal/config/watcher_test.go` | 39 |
| `internal/event/emitter_test.go` | 26 |
| `internal/platform/single_instance_windows_test.go` | 17 |
| `internal/server/ai_config_test.go` | 201 |
| `internal/server/helpers_test.go` | 38 |
| `internal/server/info_test.go` | 23 |
| `internal/server/rules_test.go` | 101 |
| `internal/server/telegram_config_test.go` | 66 |
| `internal/storage/store_test.go` | 38 |
| `internal/telegram/bot_test.go` | 109 |

Observations:

- The test suite is still strongest in `internal/server` and `internal/ai`, but collector/config/event/storage coverage is no longer empty.
- `internal/anomaly` remains the most under-tested high-risk package.

## Appendix: Git Activity

Recent commit history:

| Commit | Summary |
|---|---|
| `83d4ed4` | Add release version badge and whitespace cleanup |
| `2df946d` | ci: harden Windows release workflow |
| `456698e` | Document single-instance guard, Telegram rescue bot, and background AI watch features |
| `c8d8849` | Add version parameter to build script and inject version at build time |
| `7f4b0df` | Remove temporary build artifact and update main executable |
| `7b11da0` | Update README with new features, add action approval flow, and refine anomaly defaults |
| `6b8141e` | Add configurable alert cap and process ignore list to reduce UI noise |
| `25150d3` | init |

Contributor summary from `git shortlog -sn --all`:

| Commits | Contributor |
|---|---|
| 8 | `Ersin KOC` |

Repository activity assessment:

- This is currently a single-author codebase.
- The recent commits skew toward documentation, release plumbing, and UX refinement rather than deep collector correctness or test expansion.
- That reinforces the roadmap recommendation to pause feature/document surface growth until correctness and hardening catch up.
