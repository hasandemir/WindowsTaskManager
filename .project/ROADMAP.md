# Project Roadmap

> Based on comprehensive codebase analysis performed on 2026-04-12
> This roadmap prioritizes work needed to bring the project to production quality.

## Current State Assessment

WindowsTaskManager already has a solid modular-monolith foundation, a working local dashboard, live system/process collection, process control actions, anomaly detection, AI-assisted analysis, and a Telegram integration. The main production blockers are no longer broad correctness failures. They are now narrower gaps around spec parity, deeper telemetry depth, test coverage, observability, and release maturity.

What is working well:

- Single-binary local app model
- Minimal dependency surface
- Clean controller package and good safety instincts
- Embedded UI + SSE transport
- Config schema and atomic config persistence

Key blockers for production readiness:

- Spec/docs still over-promise relative to the shipped API/UI
- GPU and disk telemetry are now real, but not yet as deep as the original design docs
- Sparse browser/tray/Win32 edge-case test coverage
- Thin observability and operational diagnostics
- CI is present, but still a minimal baseline

## Remediation Update

Completed from the original critical path:

- [x] Duplicate history insertion fixed in collector/storage flow
- [x] Regression tests added for history mutation safety
- [x] Localhost mutation protection added with CSRF token + Origin validation
- [x] Hardcoded API version removed
- [x] Non-blocking emitter behavior aligned with implementation
- [x] Process children/connections endpoints added
- [x] Alert dismiss/snooze endpoints added
- [x] Lazy-load AI model catalog to reduce unsolicited outbound requests
- [x] Native Windows config watcher added with fallback polling path
- [x] Process I/O reporting changed to delta semantics
- [x] Collector coverage expanded for watcher/network/process behavior
- [x] Collector coverage expanded for anomaly detector behavior
- [x] Disk I/O telemetry added via Windows `LogicalDisk(*)` perf counters
- [x] GPU utilization + memory telemetry added via Windows GPU perf counters
- [x] `PUT /api/v1/config` added for non-secret runtime settings
- [x] Basic router-level browser smoke coverage added for dashboard/system/config flow
- [x] AI chat endpoint and lightweight multi-turn UI added

Current top blockers after this remediation wave:

- [ ] Advanced dashboard/detail UX is still much thinner than the original plan
- [ ] GPU/disk collectors still do not match the full D3DKMT/NVML-level design depth in the spec
- [ ] Win32/tray/browser-flow coverage is still too low
- [x] Push/PR CI baseline added in `.github/workflows/ci.yml`

## Phase 1: Critical Fixes (Week 1-2)

### Must-fix items blocking a trustworthy beta release

- [ ] Reconcile README/spec/task docs with the actual post-remediation codebase so product claims stop drifting. Effort: 8h.
- [ ] Decide whether the new perf-counter GPU/disk collectors are the accepted design, or whether the team is still committing to deeper D3DKMT/NVML work. Effort: 6h.
- [ ] Add runtime validation around collector output shape on real Windows hosts, especially GPU/disk/network. Effort: 10h.
- [ ] Expand the new browser smoke net from route-level checks to richer SSE/process-action UI coverage. Effort: 10h.

## Phase 2: Core Completion (Week 3-6)

### Complete missing core features from specification

- [ ] Continue refining network adapter filtering against more Windows host profiles. Spec/TASK ref: Task 34. Effort: 8h.
- [ ] Decide whether to stop at the current disk I/O model or continue toward queue-depth / deeper disk telemetry. Spec/TASK ref: Task 38. Effort: 12h.
- [ ] Decide whether to stop at the current GPU perf-counter model or continue toward D3DKMT/NVML depth. Spec/TASK ref: Task 39/40. Effort: 16-32h.
- [ ] Add missing alert workflow semantics beyond dismiss/snooze if they are still product requirements. Spec/TASK ref: Task 76. Effort: 8h.
- [ ] Expand the new AI chat into a richer planned UX: better transcript rendering, alert-to-chat linking, and conversation/session controls. Spec/TASK ref: Task 77, 92, 110. Effort: 12-20h.

## Phase 3: Hardening (Week 7-8)

### Security, error handling, edge cases

- [ ] Introduce recovery middleware with consistent JSON error responses in `internal/server`. Effort: 4h.
- [ ] Review all mutation handlers for explicit request validation, confirm semantics, and safe defaults. Effort: 8h.
- [x] Gate external `models.dev` fetch behind an explicit AI UI interaction or config flag to preserve privacy expectations. Effort: 6h.
- [ ] Document secret storage behavior and consider Windows Credential Manager or DPAPI-backed secret storage for AI and Telegram tokens. Effort: 12h.
- [ ] Consider bounded dispatch/backpressure semantics on top of the now-non-blocking emitter if subscriber fan-out grows. Effort: 8h.
- [ ] Add panic-safe handling and timeout discipline around optional external HTTP clients. Effort: 6h.

## Phase 4: Testing (Week 9-10)

### Comprehensive test coverage

- [ ] Continue expanding tests in `internal/collector`, `internal/anomaly`, `internal/storage`, and add stronger `internal/winapi`/tray/browser coverage. Effort: 28h.
- [ ] Add integration tests for key API flows: process actions, alerts workflow, SSE, rules update, AI execute guard rails. Effort: 20h.
- [ ] Add runtime regression tests for process/disk/network collector output shape on Windows. Effort: 16h.
- [ ] Establish at least one frontend smoke test path for dashboard load + SSE + process action visibility. Effort: 12h.
- [ ] Enable `go test -race` in an environment where CGO is available. Effort: 4h.

## Phase 5: Performance & Optimization (Week 11-12)

### Performance tuning and optimization

- [ ] Profile collector loop timings on busy machines and verify headroom at 500+ processes. Effort: 8h.
- [ ] Split `web/app.js` into smaller modules or move to a typed build setup for maintainability and performance tuning. Effort: 20h.
- [ ] Reduce full-table DOM rebuild behavior in the dashboard; move to diffed updates or virtualized rendering for large process lists. Effort: 16h.
- [ ] Add cache headers and light static asset tuning for embedded UI serving. Effort: 4h.
- [ ] Audit memory growth of in-memory history under long-running sessions. Effort: 6h.

## Phase 6: Documentation & DX (Week 13-14)

### Documentation and developer experience

- [ ] Reconcile `README.md`, `.project/SPECIFICATION.md`, `.project/IMPLEMENTATION.md`, and `.project/TASKS.md` with the actual codebase. Effort: 10h.
- [ ] Publish an accurate API reference for the 45 current routes. Effort: 8h.
- [ ] Add `CONTRIBUTING.md` with build, test, release, and Windows requirements. Effort: 4h.
- [ ] Stop mutating the worktree inside `build.ps1`; keep build, tidy, fmt, and verify as separate commands or CI steps. Effort: 2h.
- [ ] Add architecture notes explaining trust boundaries, localhost security assumptions, and current non-goals. Effort: 6h.

## Phase 7: Release Preparation (Week 15-16)

### Final production preparation

- [x] Add normal CI on push/PR for `go test`, `go vet`, `staticcheck`, and `go build`. Effort: 6h.
- [ ] Add a signed release checklist that includes version verification, UI sanity, and config migration checks. Effort: 4h.
- [ ] Consider `.goreleaser.yml` only if multi-artifact releases are needed; otherwise keep release flow simple and documented. Effort: 6h.
- [ ] Embed correct version/build metadata into CLI, API, tray tooltip, and release assets consistently. Effort: 4h.
- [ ] Add observability basics: request IDs, structured logs or at least leveled logging, and richer health output. Effort: 10h.

## Beyond v1.0: Future Enhancements

### Features and improvements for future versions

- [ ] Replace plaintext secret storage with Windows-native secret storage.
- [ ] Add per-process connection detail and richer process drill-down UI.
- [ ] Add typed frontend architecture or a small build step for maintainability.
- [ ] Add optional persistent historical metrics export.
- [ ] Add explicit remote-control mode only if paired with real authentication, TLS, and audit logging.

## Effort Summary

| Phase | Estimated Hours | Priority | Dependencies |
|---|---:|---|---|
| Phase 1 | 30-34h | CRITICAL | None |
| Phase 2 | 64-76h | HIGH | Phase 1 |
| Phase 3 | 34-44h | HIGH | Phase 1 |
| Phase 4 | 64-80h | HIGH | Phase 1-3 |
| Phase 5 | 46-54h | MEDIUM | Phase 1-4 |
| Phase 6 | 24-30h | MEDIUM | Phase 1-4 |
| Phase 7 | 18-24h | HIGH | Phase 1-6 |
| **Total** | **280-342h** |  |  |

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Spec/docs continue to diverge from code and erode trust | High | High | Treat documentation reconciliation as a release requirement |
| GPU/disk depth remains ambiguous and product claims drift again | Medium | High | Decide whether the new perf-counter path is final or only an interim step |
| Browser/tray/Win32 regressions slip through because the new tests still miss edge cases | Medium | High | Add smoke coverage where current unit tests are weakest |
| UI degrades on large developer workstations | Medium | Medium | Profile and optimize rendering before wider release |
| Extra feature surface outpaces testing capacity | High | Medium | Pause net-new features until collector/anomaly/storage coverage improves |

## Delivery Recommendation

The shortest realistic path to a trustworthy v1.0 is not "keep adding features." It is:

1. Lock in the new correctness and local-security baseline so it stays fixed.
2. Decide which post-remediation behaviors are now the product baseline, then update the docs to match.
3. Put collector/anomaly/storage/browser behavior under deeper tests.
4. Expand the new CI baseline so regressions stop shipping quietly.

If those four things happen, this project can move from impressive prototype to credible production-grade Windows utility.
