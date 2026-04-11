# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project aims to follow
Semantic Versioning.

## [Unreleased]

### Added

- Placeholder for upcoming changes.

## [0.1.0] - 2026-04-11

### Added

- Initial public release of Windows Task Manager as a single-binary, pure-Go Windows utility.
- Live local web dashboard with CPU, memory, GPU, disk, network, process list, process tree, ports, alerts, and SSE updates.
- Process control operations including kill, suspend, resume, priority, affinity, and Job Object-based resource limits.
- Per-process protect and ignore toggles persisted into `config.yaml`.
- Conservative anomaly engine with built-in detectors for runaway CPU, memory leaks, spawn storms, and optional detectors for hung, orphaned, port, network, and suspicious new processes.
- User-defined automation rules editable from YAML and the web UI.
- AI advisor with approve-before-execute actions, provider presets, background watch mode, and dry-run auto-action policy evaluation.
- Telegram rescue bot with status, top CPU, alert inspection, and confirm-gated destructive actions.
- Native Windows tray integration with notifications and dashboard reopening.
- GitHub Actions release workflow that tests, vets, builds, and uploads versioned Windows release assets.

### Changed

- Added single-instance startup protection so a second `wtm.exe` copy refuses to launch.
- Added self-protection so the running WTM process cannot be killed or suspended from the UI, AI suggestions, rules, or Telegram.
- Hardened config persistence with atomic saves and better hot-reload propagation.
- Updated AI examples, defaults, and presets to newer model names and current provider recommendations.

### Fixed

- Fixed local-only middleware to correctly allow IPv6 loopback requests.
- Tightened JSON request parsing to reject oversized bodies and trailing garbage.
- Improved runtime config updates for collectors, anomaly analysis, tray sync, and other long-lived loops.

### Security

- Enforced local-only API access and stronger destructive-action guards for protected, critical, and self processes.
- Added Telegram confirmation codes for remote kill and suspend style actions.
