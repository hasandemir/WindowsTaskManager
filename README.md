# Windows Task Manager (WTM)

A single-binary, pure-Go alternative to the built-in Windows Task Manager
that exposes a real-time web dashboard, REST API, anomaly detection engine,
and an optional Anthropic Claude advisor.

- **Pure Go.** No CGo, no WMI, no PowerShell. Direct Win32 API calls via
  `golang.org/x/sys/windows`.
- **Single executable.** The web UI is embedded with `embed.FS`. Drop
  `wtm.exe` anywhere and run.
- **Live everything.** CPU (per-core), memory, GPU (best effort), disks,
  network throughput, processes, process tree, TCP/UDP endpoints.
- **Process control.** Kill / suspend / resume, priority, affinity, and
  Job-Object-based CPU + memory caps — all gated by a configurable
  protection list.
- **Anomaly detection.** Eight built-in detectors: spawn storm, memory
  leak (linear regression with R²), hung process, orphan, runaway CPU,
  port conflicts, network bursts, and new processes from suspicious paths.
- **System tray.** Native NotifyIcon with rate-limited balloon
  notifications. Right-click for menu.
- **AI advisor (optional).** Talk to Anthropic Claude *or any
  OpenAI-compatible provider* (OpenAI, OpenRouter, Groq, DeepSeek,
  Together, Mistral, Fireworks, xAI, Ollama, LM Studio, …) with the
  current snapshot + alerts pre-loaded as context.
- **Configurable.** YAML config with live reload — edit, save, no restart.

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

A reference copy lives at [`configs/default.yaml`](configs/default.yaml).

### AI advisor

Two providers are supported:

- `provider: anthropic` → Anthropic Messages API (`/v1/messages`)
- `provider: openai`    → any OpenAI-compatible `/v1/chat/completions`
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
  model: claude-sonnet-4-20250514
  language: tr

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
context with every request. Responses are cached for 60s and rate-limited
to `max_requests_per_minute`.

## Architecture

```
cmd/wtm                 # entry point + flag parsing + supervision
internal/winapi         # raw Win32 syscall wrappers (kernel32, ntdll, ...)
internal/stats          # Welford, linear regression, EMA, ring buffer
internal/config         # YAML loader + file watcher
internal/event          # tiny pub/sub emitter
internal/metrics        # shared metric struct definitions
internal/storage        # in-memory ring storage for snapshots + per-PID history
internal/collector      # CPU/mem/process/tree/net/ports/disk/GPU collectors
internal/controller     # kill/suspend/priority/affinity/Job Object limits
internal/anomaly        # detection engine + 8 detectors + alert store
internal/server         # HTTP router, REST handlers, SSE hub, static UI
internal/ai             # Anthropic Messages API client + cache + rate limit
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
| GET | `/api/v1/stream` | Server-Sent Events (snapshot + alerts) |
| POST | `/api/v1/ai/analyze` | `{"prompt":"..."}` |
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

## License

MIT — see [LICENSE](LICENSE) if present.
Author: **Ersin Koç**
