# Windows Task Manager (WTM)

A single-binary, pure-Go alternative to the built-in Windows Task Manager
that exposes a real-time web dashboard, REST API, anomaly detection engine,
and an optional LLM advisor with an approve-before-execute action flow.

- **Pure Go.** No CGo, no WMI, no PowerShell. Direct Win32 API calls via
  `golang.org/x/sys/windows`.
- **Single executable.** The web UI is embedded with `embed.FS`. Drop
  `wtm.exe` anywhere and run.
- **Live everything.** CPU (per-core), memory, GPU (best effort), disks,
  network throughput, processes, process tree, TCP/UDP endpoints.
- **Process control.** Kill / suspend / resume, priority, affinity, and
  Job-Object-based CPU + memory caps — all gated by a configurable
  protection list.
- **Per-process toggles.** Click the 🛡 on any row to protect a process
  from kill/suspend, or 🔕 to hide it from the anomaly engine. Both lists
  are persisted to `config.yaml`.
- **Sortable process table.** Click any column header (PID, Name, CPU,
  Memory, Threads) to sort; click again to flip direction. Persists
  across reloads.
- **Anomaly detection — conservative by default.** Eight built-in detectors.
  Only the three that flag *actual* danger fire out of the box — the
  rest are opt-in because they're noisy on normal developer workstations:

  | Detector | Default |
  |---|---|
  | `runaway_cpu` — sustained high CPU | **on** |
  | `memory_leak` — linear-regression R² growth | **on** |
  | `spawn_storm` — fork bomb (with shell/browser whitelist) | **on** |
  | `hung_process` — idle, no prior activity, no I/O | off |
  | `orphan` — parent gone and still burning CPU/RAM | off |
  | `port_conflict` — duplicate listening ports | off |
  | `network_anomaly` — σ-burst with min connection floor | off |
  | `new_process` — first-seen executable from suspicious path | off |

- **Automation rules.** Write YAML (or use the Rules tab) to say *"if
  `chrome.exe` uses ≥ 4 GB for 30 s, kill it"*. Rules hot-reload and
  respect the same protect list as the manual controls.
- **AI advisor with approve-before-execute.** Talk to Anthropic Claude
  *or any OpenAI-compatible provider* (OpenAI, OpenRouter, Groq,
  DeepSeek, Together, Mistral, Fireworks, xAI, Ollama, LM Studio, …).
  The advisor embeds a structured `<actions>` block in its reply; the
  UI parses it into per-card *Approve / Dismiss* buttons. Nothing runs
  until you click Approve. Supported action types: `kill`, `suspend`,
  `protect`, `ignore`, `add_rule`.
- **System tray.** Native NotifyIcon with rate-limited balloon
  notifications. Right-click for menu.
- **Configurable.** YAML config with live reload and schema migration —
  edit, save, no restart.

## Building

```powershell
.\build.ps1
```

This produces `wtm.exe` (~12 MB, no console window).

For a quick dev build:

```bash
go build -o wtm.exe ./cmd/wtm
```

## Running

```powershell
.\wtm.exe
```

The dashboard opens automatically at `http://127.0.0.1:19876`. The tray
icon appears in the notification area; right-click for menu, double-click
to reopen the dashboard.

### Flags

| Flag | Description |
|---|---|
| `--config <path>` | Override config file location |
| `--no-tray` | Disable system tray (run headless) |
| `--no-browser` | Don't auto-open the dashboard |
| `--version` | Print version and exit |

### Config file

On first run, WTM writes a default config to:

```
%APPDATA%\WindowsTaskManager\config.yaml
```

You can also drop a `config.yaml` next to `wtm.exe` and it will be used in
preference. Edits to either file are picked up live.

The file starts with a `schema_version:` field. When you upgrade to a
new WTM binary that ships a breaking defaults change, the loader
rewrites the affected sections in place on first launch — your
`ignore_processes` list and protected-process list are preserved, the
rest is reset to the new defaults.

A reference copy lives at [`configs/default.yaml`](configs/default.yaml).

Notable `anomaly:` knobs:

- `anomaly.ignore_processes` — executable names the engine should skip
  entirely. Populated from the UI's 🔕 toggle too.
- `anomaly.max_active_alerts` — hard cap on the active alert set so a
  misbehaving detector can't drown the UI.

### AI advisor

Two providers are supported:

- `provider: anthropic` → Anthropic Messages API (`/v1/messages`). If
  you point `endpoint:` at an Anthropic-compatible proxy (e.g. z.ai)
  that doesn't already end in `/v1/messages`, WTM appends it for you.
- `provider: openai` → any OpenAI-compatible `/v1/chat/completions`
  endpoint — OpenAI, OpenRouter, Groq, DeepSeek, Together, Mistral,
  Fireworks, xAI, Ollama, LM Studio, vLLM, llama.cpp's server, …

You can configure it three ways:

1. **From the dashboard** — open the **AI** tab, expand
   *Provider settings*, pick a preset (or fill the fields manually) and
   click **Save**. The change is written back to `config.yaml` and
   applied live.
2. **By editing `config.yaml`** — see the `ai:` block in
   [`configs/default.yaml`](configs/default.yaml). Edits are picked up by
   the file watcher within ~2 seconds.
3. **Via the REST API** — `POST /api/v1/ai/config` with the same JSON
   shape returned by `GET /api/v1/ai/config`.

Examples:

```yaml
# Anthropic Claude (default)
ai:
  enabled: true
  provider: anthropic
  api_key: sk-ant-...
  model: claude-sonnet-4-5
  language: en

# OpenAI
ai:
  enabled: true
  provider: openai
  api_key: sk-...
  model: gpt-4o-mini

# OpenRouter (single key, hundreds of models)
ai:
  enabled: true
  provider: openai
  endpoint: https://openrouter.ai/api/v1/chat/completions
  api_key: sk-or-v1-...
  model: anthropic/claude-3.5-sonnet
  extra_headers:
    HTTP-Referer: http://localhost
    X-Title: WTM

# Groq (very fast Llama / Mixtral)
ai:
  enabled: true
  provider: openai
  endpoint: https://api.groq.com/openai/v1/chat/completions
  api_key: gsk_...
  model: llama-3.3-70b-versatile

# DeepSeek
ai:
  enabled: true
  provider: openai
  endpoint: https://api.deepseek.com/v1/chat/completions
  api_key: sk-...
  model: deepseek-chat

# Local Ollama
ai:
  enabled: true
  provider: openai
  endpoint: http://localhost:11434/v1/chat/completions
  api_key: ollama         # any non-empty string
  model: llama3.1

# Local LM Studio
ai:
  enabled: true
  provider: openai
  endpoint: http://localhost:1234/v1/chat/completions
  api_key: lm-studio
  model: local-model
```

The current snapshot, top processes, and active alerts are sent as
context with every request. Responses are cached for 60 s and rate-limited
to `max_requests_per_minute`.

**Action flow.** The advisor is prompted to end its reply with an
`<actions>[...]</actions>` block. The server parses it into
`Suggestion` records with stable hashed IDs and returns them alongside
the human answer. The UI renders each suggestion as a card with Approve
and Dismiss buttons; clicking Approve POSTs the full suggestion to
`POST /api/v1/ai/execute` which dispatches to the controller (kill /
suspend), mutates `config.yaml` (protect / ignore / add_rule), and
applies the change live. `kill` and `suspend` are refused for processes
on the protection list or flagged critical.

## Architecture

```
cmd/wtm                 # entry point + flag parsing + supervision
internal/winapi         # raw Win32 syscall wrappers (kernel32, ntdll, ...)
internal/stats          # Welford, linear regression, EMA, ring buffer
internal/config         # YAML loader + schema migration + file watcher
internal/event          # tiny pub/sub emitter
internal/metrics        # shared metric struct definitions
internal/storage        # in-memory ring storage for snapshots + per-PID history
internal/collector      # CPU/mem/process/tree/net/ports/disk/GPU collectors
internal/controller     # kill/suspend/priority/affinity/Job Object limits
internal/anomaly        # detection engine + 8 detectors + alert store
internal/server         # HTTP router, REST handlers, SSE hub, static UI
internal/ai             # Advisor + anthropic/openai clients + actions parser
internal/tray           # Win32 NotifyIcon + message-pump + balloon notifications
internal/platform       # admin elevation helpers
web/                    # embedded dashboard (HTML / CSS / vanilla JS)
configs/                # reference default config
```

## REST API (selected)

Every endpoint is local-only (loopback). The full list lives in
[`internal/server/handlers.go`](internal/server/handlers.go).

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/system` | Latest full system snapshot |
| GET | `/api/v1/processes?sort=cpu&limit=50` | Process list |
| GET | `/api/v1/processes/{pid}` | Single process |
| GET | `/api/v1/processes/{pid}/history` | Per-process metrics history |
| POST | `/api/v1/processes/{pid}/kill?confirm=true` | Terminate process |
| POST | `/api/v1/processes/{pid}/kill-tree?confirm=true` | Terminate subtree |
| POST | `/api/v1/processes/{pid}/suspend` | Pause all threads |
| POST | `/api/v1/processes/{pid}/resume` | Resume all threads |
| POST | `/api/v1/processes/{pid}/priority` | `{"class":"high"}` |
| POST | `/api/v1/processes/{pid}/affinity` | `{"mask":15}` |
| POST | `/api/v1/processes/{pid}/limit` | `{"cpu_pct":25,"mem_bytes":1073741824}` |
| GET | `/api/v1/ports` | TCP/UDP endpoints with PIDs |
| GET | `/api/v1/alerts` | Active anomaly alerts |
| POST | `/api/v1/alerts/clear` | Wipe the active alert set |
| GET | `/api/v1/stream` | Server-Sent Events (snapshot + alerts) |
| GET | `/api/v1/config` | Current effective config (api keys masked) |
| POST | `/api/v1/config/protect` | Toggle per-process protect list |
| POST | `/api/v1/config/ignore` | Toggle per-process anomaly ignore list |
| POST | `/api/v1/ai/analyze` | `{"prompt":"..."}` — returns answer + actions |
| POST | `/api/v1/ai/execute` | Execute an approved AI suggestion |
| GET | `/api/v1/ai/status` | Provider, model, rate-limit, cache stats |
| GET | `/api/v1/ai/config` | Current AI block (api key masked) |
| POST | `/api/v1/ai/config` | Update provider/model/key/headers; persists to `config.yaml` |
| GET | `/api/v1/ai/presets` | Curated list of provider/model starter templates |

## Permissions

WTM works fine without admin rights but some features will be limited:

- Without elevation, querying memory/IO for system processes returns
  partial data.
- Killing/suspending privileged processes requires admin.
- The protection list (`controller.protected_processes`) is enforced
  regardless of privilege.

The `controller.confirm_kill_system: true` setting requires explicit
`?confirm=true` for any executable under `C:\Windows\`.

## Author & links

Built by **Ersin Koç** — <https://github.com/ersinkoc/WindowsTaskManager>

Issues, PRs and feature requests welcome.

## License

MIT — see [LICENSE](LICENSE) if present.
