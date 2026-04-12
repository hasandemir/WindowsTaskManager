import { useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Bot, Cable, Cog, Settings2 } from "lucide-react";
import { EmptyState } from "../components/shared/empty-state";
import { PageHeader } from "../components/shared/page-header";
import { PageSkeleton } from "../components/shared/page-skeleton";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import {
  useAIConfigMutation,
  useAIConfigQuery,
  useAIPresetsQuery,
  useConfigQuery,
  useConfigUpdateMutation,
  useTelegramConfigMutation,
  useTelegramConfigQuery,
} from "../lib/api-client";
import type { AIConfig, TelegramConfig } from "../types/api";

const telegramAlertTypeOptions = [
  { value: "runaway_cpu", label: "Runaway CPU" },
  { value: "memory_leak", label: "Memory leak" },
  { value: "port_conflict", label: "Port conflict" },
  { value: "new_process", label: "Suspicious new process" },
  { value: "network_anomaly", label: "Network anomaly" },
  { value: "network_anomaly_system", label: "System-wide network anomaly" },
  { value: "rule:*", label: "Rule-triggered critical actions" },
];

function headersToText(headers: Record<string, string> | undefined) {
  if (!headers) {
    return "";
  }
  return Object.entries(headers)
    .map(([key, value]) => `${key}: ${value}`)
    .join("\n");
}

function textToHeaders(text: string) {
  const output: Record<string, string> = {};
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line) {
      continue;
    }
    const separatorIndex = line.indexOf(":");
    if (separatorIndex <= 0) {
      continue;
    }
    const key = line.slice(0, separatorIndex).trim();
    const value = line.slice(separatorIndex + 1).trim();
    if (key) {
      output[key] = value;
    }
  }
  return output;
}

function durationToMS(ns: number | undefined) {
  return Math.max(100, Math.round((ns ?? 0) / 1_000_000));
}

function durationToSec(ns: number | undefined) {
  return Math.max(1, Math.round((ns ?? 0) / 1_000_000_000));
}

export function SettingsPage() {
  const { data: aiConfig, isLoading: aiLoading } = useAIConfigQuery();
  const { data: aiPresets = [] } = useAIPresetsQuery();
  const { data: telegramConfig, isLoading: telegramLoading } = useTelegramConfigQuery();
  const { data: appConfig, isLoading: configLoading } = useConfigQuery();
  const aiMutation = useAIConfigMutation();
  const telegramMutation = useTelegramConfigMutation();
  const configMutation = useConfigUpdateMutation();

  const [selectedPreset, setSelectedPreset] = useState("");
  const [aiForm, setAIForm] = useState<AIConfig>({
    enabled: false,
    provider: "anthropic",
    api_key: "",
    model: "",
    endpoint: "",
    extra_headers: {},
    language: "en",
    max_tokens: 1024,
    max_requests_per_minute: 5,
    include_process_tree: true,
    include_port_map: true,
  });
  const [aiHeadersText, setAIHeadersText] = useState("");
  const [telegramForm, setTelegramForm] = useState<TelegramConfig>({
    enabled: false,
    bot_token: "",
    allowed_chat_ids: [],
    api_base_url: "https://api.telegram.org",
    poll_timeout_sec: 25,
    notify_on_critical: true,
    notification_mode: "high_value",
    notification_types: ["runaway_cpu", "memory_leak", "port_conflict", "new_process", "network_anomaly", "network_anomaly_system", "rule:*"],
    require_confirm: true,
    confirm_ttl_sec: 90,
  });
  const [telegramChatIDsText, setTelegramChatIDsText] = useState("");
  const [runtimeForm, setRuntimeForm] = useState({
    openBrowser: true,
    intervalMS: 1000,
    processTreeIntervalMS: 2000,
    portScanIntervalMS: 3000,
    gpuIntervalMS: 2000,
    historyDurationSec: 600,
    maxProcesses: 2000,
    confirmKillSystem: true,
    trayBalloon: true,
    balloonRateLimitSec: 30,
    balloonMinSeverity: "critical",
    theme: "system",
    defaultSort: "cpu",
    defaultSortOrder: "desc",
    sparklinePoints: 60,
    processTablePageSize: 100,
    refreshRateMS: 1000,
  });

  useEffect(() => {
    if (!aiConfig) {
      return;
    }
    setAIForm({ ...aiConfig, api_key: "" });
    setAIHeadersText(headersToText(aiConfig.extra_headers));
  }, [aiConfig]);

  useEffect(() => {
    if (!telegramConfig) {
      return;
    }
    setTelegramForm({ ...telegramConfig, bot_token: "" });
    setTelegramChatIDsText((telegramConfig.allowed_chat_ids ?? []).join(", "));
  }, [telegramConfig]);

  useEffect(() => {
    if (!appConfig) {
      return;
    }
    setRuntimeForm({
      openBrowser: appConfig.Server.OpenBrowser,
      intervalMS: durationToMS(appConfig.Monitoring.Interval),
      processTreeIntervalMS: durationToMS(appConfig.Monitoring.ProcessTreeInterval),
      portScanIntervalMS: durationToMS(appConfig.Monitoring.PortScanInterval),
      gpuIntervalMS: durationToMS(appConfig.Monitoring.GPUInterval),
      historyDurationSec: durationToSec(appConfig.Monitoring.HistoryDuration),
      maxProcesses: appConfig.Monitoring.MaxProcesses,
      confirmKillSystem: appConfig.Controller.ConfirmKillSystem,
      trayBalloon: appConfig.Notifications.TrayBalloon,
      balloonRateLimitSec: durationToSec(appConfig.Notifications.BalloonRateLimit),
      balloonMinSeverity: appConfig.Notifications.BalloonMinSeverity,
      theme: appConfig.UI.Theme,
      defaultSort: appConfig.UI.DefaultSort,
      defaultSortOrder: appConfig.UI.DefaultSortOrder,
      sparklinePoints: appConfig.UI.SparklinePoints,
      processTablePageSize: appConfig.UI.ProcessTablePageSize,
      refreshRateMS: durationToMS(appConfig.UI.RefreshRate),
    });
  }, [appConfig]);

  const aiKeyState = useMemo(() => {
    if (!aiConfig?.api_key) {
      return "No key currently saved";
    }
    return `Current key: ${aiConfig.api_key} — leave blank to keep`;
  }, [aiConfig?.api_key]);

  const telegramTokenState = useMemo(() => {
    if (!telegramConfig?.bot_token) {
      return "No bot token currently saved";
    }
    return `Current token: ${telegramConfig.bot_token} — leave blank to keep`;
  }, [telegramConfig?.bot_token]);

  if (aiLoading || telegramLoading || configLoading) {
    return <PageSkeleton />;
  }

  if (!aiConfig || !telegramConfig || !appConfig) {
    return <EmptyState icon={Settings2} title="Settings are unavailable" description="WTM could not load one or more local configuration endpoints." />;
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Settings" description="Configure AI, Telegram, collector cadence, notifications, and local UI defaults." />

      <div className="grid gap-4 xl:grid-cols-3">
        <MetricCard label="AI provider" value={aiConfig.provider || "anthropic"} badge={aiConfig.enabled ? "Enabled" : "Disabled"} badgeVariant={aiConfig.enabled ? "success" : "warning"} />
        <MetricCard label="Telegram" value={telegramConfig.enabled ? "Online" : "Off"} badge={telegramConfig.notify_on_critical ? "Critical alerts on" : "Quiet"} badgeVariant={telegramConfig.enabled ? "info" : "neutral"} />
        <MetricCard label="Dashboard runtime" value={`${runtimeForm.intervalMS} ms`} badge={runtimeForm.theme} badgeVariant="info" />
      </div>

      <Card className="space-y-5">
        <SectionTitle icon={Bot} title="AI provider settings" description="Choose the provider, model, limits, and context that power the local advisor." />
        <div className="grid gap-4 lg:grid-cols-2">
          <Field label="Preset">
            <select
              aria-label="AI preset"
              className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground"
              value={selectedPreset}
              onChange={(event) => {
                const presetID = event.target.value;
                setSelectedPreset(presetID);
                const preset = aiPresets.find((entry) => entry.id === presetID);
                if (!preset) {
                  return;
                }
                setAIForm((current) => ({
                  ...current,
                  provider: preset.provider,
                  endpoint: preset.endpoint || current.endpoint,
                  model: preset.model || current.model,
                }));
                setAIHeadersText(headersToText(preset.extra_headers));
              }}
            >
              <option value="">Choose a starter preset</option>
              {aiPresets.map((preset) => (
                <option key={preset.id} value={preset.id}>
                  {preset.label}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Provider">
            <select
              aria-label="AI provider"
              className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground"
              value={aiForm.provider}
              onChange={(event) => setAIForm((current) => ({ ...current, provider: event.target.value }))}
            >
              <option value="anthropic">Anthropic</option>
              <option value="openai">OpenAI compatible</option>
            </select>
          </Field>
          <Field label="Model">
            <input aria-label="AI model" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={aiForm.model} onChange={(event) => setAIForm((current) => ({ ...current, model: event.target.value }))} />
          </Field>
          <Field label="Endpoint">
            <input aria-label="AI endpoint" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={aiForm.endpoint} onChange={(event) => setAIForm((current) => ({ ...current, endpoint: event.target.value }))} />
          </Field>
          <Field label="API key">
            <div>
              <input
                aria-label="AI API key"
                type="password"
                placeholder="Leave blank to keep current key"
                className="h-11 w-full rounded-xl border border-border bg-background px-3 text-sm text-foreground"
                value={aiForm.api_key}
                onChange={(event) => setAIForm((current) => ({ ...current, api_key: event.target.value }))}
              />
              <div className="mt-2 text-xs text-secondary">{aiKeyState}</div>
            </div>
          </Field>
          <Field label="Language">
            <input aria-label="AI language" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={aiForm.language} onChange={(event) => setAIForm((current) => ({ ...current, language: event.target.value }))} />
          </Field>
          <Field label="Max tokens">
            <input aria-label="AI max tokens" type="number" min={64} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={aiForm.max_tokens} onChange={(event) => setAIForm((current) => ({ ...current, max_tokens: Number(event.target.value) || 64 }))} />
          </Field>
          <Field label="Requests per minute">
            <input aria-label="AI requests per minute" type="number" min={1} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={aiForm.max_requests_per_minute} onChange={(event) => setAIForm((current) => ({ ...current, max_requests_per_minute: Number(event.target.value) || 1 }))} />
          </Field>
        </div>
        <Field label="Extra headers">
          <textarea aria-label="AI extra headers" className="min-h-28 w-full rounded-2xl border border-border bg-background px-4 py-3 text-sm text-foreground" placeholder="HTTP-Referer: http://localhost" value={aiHeadersText} onChange={(event) => setAIHeadersText(event.target.value)} />
        </Field>
        <div className="grid gap-3 sm:grid-cols-3">
          <Toggle label="Enable AI" checked={aiForm.enabled} onChange={(checked) => setAIForm((current) => ({ ...current, enabled: checked }))} />
          <Toggle label="Include process tree" checked={aiForm.include_process_tree} onChange={(checked) => setAIForm((current) => ({ ...current, include_process_tree: checked }))} />
          <Toggle label="Include port map" checked={aiForm.include_port_map} onChange={(checked) => setAIForm((current) => ({ ...current, include_port_map: checked }))} />
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Button type="button" disabled={aiMutation.isPending} onClick={() => aiMutation.mutate({ ...aiForm, extra_headers: textToHeaders(aiHeadersText) })}>
            Save AI settings
          </Button>
          <span className="text-sm text-secondary">Chat stays on the AI page; provider setup lives here.</span>
        </div>
      </Card>

      <div className="grid gap-6 xl:grid-cols-2">
        <Card className="space-y-5">
          <SectionTitle icon={Cable} title="Telegram settings" description="Bot token, allowed chats, and confirmation behavior." />
          <div className="grid gap-4">
            <Field label="Bot token">
              <div>
                <input aria-label="Telegram bot token" type="password" placeholder="Leave blank to keep current token" className="h-11 w-full rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={telegramForm.bot_token} onChange={(event) => setTelegramForm((current) => ({ ...current, bot_token: event.target.value }))} />
                <div className="mt-2 text-xs text-secondary">{telegramTokenState}</div>
              </div>
            </Field>
            <Field label="Allowed chat IDs">
              <input aria-label="Telegram allowed chat IDs" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={telegramChatIDsText} onChange={(event) => setTelegramChatIDsText(event.target.value)} />
            </Field>
            <Field label="API base URL">
              <input aria-label="Telegram API base URL" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={telegramForm.api_base_url} onChange={(event) => setTelegramForm((current) => ({ ...current, api_base_url: event.target.value }))} />
            </Field>
            <Field label="Notification mode">
              <select aria-label="Telegram notification mode" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={telegramForm.notification_mode} onChange={(event) => setTelegramForm((current) => ({ ...current, notification_mode: event.target.value }))}>
                <option value="high_value">Only high-value critical alerts</option>
                <option value="all_critical">All critical alerts</option>
              </select>
            </Field>
            <Field label="High-value alert types">
              <div className="grid gap-2 sm:grid-cols-2">
                {telegramAlertTypeOptions.map((option) => {
                  const checked = telegramForm.notification_types.includes(option.value);
                  return (
                    <label key={option.value} className="flex items-center gap-3 rounded-xl border border-border bg-background px-3 py-3 text-sm text-foreground">
                      <input
                        aria-label={`Telegram type ${option.label}`}
                        type="checkbox"
                        checked={checked}
                        onChange={(event) =>
                          setTelegramForm((current) => ({
                            ...current,
                            notification_types: event.target.checked
                              ? [...current.notification_types, option.value]
                              : current.notification_types.filter((item) => item !== option.value),
                          }))
                        }
                      />
                      <span>{option.label}</span>
                    </label>
                  );
                })}
              </div>
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Poll timeout (sec)">
                <input aria-label="Telegram poll timeout" type="number" min={5} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={telegramForm.poll_timeout_sec} onChange={(event) => setTelegramForm((current) => ({ ...current, poll_timeout_sec: Number(event.target.value) || 5 }))} />
              </Field>
              <Field label="Confirm TTL (sec)">
                <input aria-label="Telegram confirm TTL" type="number" min={15} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={telegramForm.confirm_ttl_sec} onChange={(event) => setTelegramForm((current) => ({ ...current, confirm_ttl_sec: Number(event.target.value) || 15 }))} />
              </Field>
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <Toggle label="Enable Telegram" checked={telegramForm.enabled} onChange={(checked) => setTelegramForm((current) => ({ ...current, enabled: checked }))} />
            <Toggle label="Notify on critical" checked={telegramForm.notify_on_critical} onChange={(checked) => setTelegramForm((current) => ({ ...current, notify_on_critical: checked }))} />
            <Toggle label="Require confirm" checked={telegramForm.require_confirm} onChange={(checked) => setTelegramForm((current) => ({ ...current, require_confirm: checked }))} />
          </div>
          <div className="rounded-2xl border border-dashed border-border bg-background px-4 py-4 text-sm text-secondary">
            <span className="font-semibold text-foreground">High-value mode</span> suppresses noisy `hung_process` and `spawn_storm` floods and keeps Telegram for genuinely actionable critical alerts.
          </div>
          <Button
            type="button"
            disabled={telegramMutation.isPending}
            onClick={() =>
              telegramMutation.mutate({
                ...telegramForm,
                allowed_chat_ids: telegramChatIDsText
                  .split(",")
                  .map((value) => value.trim())
                  .filter(Boolean)
                  .map((value) => Number(value))
                  .filter((value) => Number.isFinite(value)),
                notification_types: Array.from(new Set(telegramForm.notification_types)),
              })
            }
          >
            Save Telegram settings
          </Button>
        </Card>

        <Card className="space-y-5">
          <SectionTitle icon={Cog} title="Runtime and dashboard" description="Browser launch, collector cadence, notifications, and UI defaults." />
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Collector interval (ms)">
              <input aria-label="Collector interval" type="number" min={100} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.intervalMS} onChange={(event) => setRuntimeForm((current) => ({ ...current, intervalMS: Number(event.target.value) || 100 }))} />
            </Field>
            <Field label="History duration (sec)">
              <input aria-label="History duration" type="number" min={60} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.historyDurationSec} onChange={(event) => setRuntimeForm((current) => ({ ...current, historyDurationSec: Number(event.target.value) || 60 }))} />
            </Field>
            <Field label="Port scan interval (ms)">
              <input aria-label="Port scan interval" type="number" min={250} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.portScanIntervalMS} onChange={(event) => setRuntimeForm((current) => ({ ...current, portScanIntervalMS: Number(event.target.value) || 250 }))} />
            </Field>
            <Field label="GPU interval (ms)">
              <input aria-label="GPU interval" type="number" min={250} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.gpuIntervalMS} onChange={(event) => setRuntimeForm((current) => ({ ...current, gpuIntervalMS: Number(event.target.value) || 250 }))} />
            </Field>
            <Field label="Max processes">
              <input aria-label="Max processes" type="number" min={100} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.maxProcesses} onChange={(event) => setRuntimeForm((current) => ({ ...current, maxProcesses: Number(event.target.value) || 100 }))} />
            </Field>
            <Field label="Refresh rate (ms)">
              <input aria-label="Refresh rate" type="number" min={100} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.refreshRateMS} onChange={(event) => setRuntimeForm((current) => ({ ...current, refreshRateMS: Number(event.target.value) || 100 }))} />
            </Field>
            <Field label="Theme">
              <select aria-label="Default theme" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.theme} onChange={(event) => setRuntimeForm((current) => ({ ...current, theme: event.target.value }))}>
                <option value="system">System</option>
                <option value="light">Light</option>
                <option value="dark">Dark</option>
              </select>
            </Field>
            <Field label="Default process sort">
              <select aria-label="Default process sort" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.defaultSort} onChange={(event) => setRuntimeForm((current) => ({ ...current, defaultSort: event.target.value }))}>
                <option value="cpu">CPU</option>
                <option value="memory">Memory</option>
                <option value="name">Name</option>
                <option value="pid">PID</option>
                <option value="threads">Threads</option>
              </select>
            </Field>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <Toggle label="Open browser on start" checked={runtimeForm.openBrowser} onChange={(checked) => setRuntimeForm((current) => ({ ...current, openBrowser: checked }))} />
            <Toggle label="Confirm kill system" checked={runtimeForm.confirmKillSystem} onChange={(checked) => setRuntimeForm((current) => ({ ...current, confirmKillSystem: checked }))} />
            <Toggle label="Tray balloon alerts" checked={runtimeForm.trayBalloon} onChange={(checked) => setRuntimeForm((current) => ({ ...current, trayBalloon: checked }))} />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Balloon rate limit (sec)">
              <input aria-label="Balloon rate limit" type="number" min={1} className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.balloonRateLimitSec} onChange={(event) => setRuntimeForm((current) => ({ ...current, balloonRateLimitSec: Number(event.target.value) || 1 }))} />
            </Field>
            <Field label="Balloon min severity">
              <select aria-label="Balloon min severity" className="h-11 rounded-xl border border-border bg-background px-3 text-sm text-foreground" value={runtimeForm.balloonMinSeverity} onChange={(event) => setRuntimeForm((current) => ({ ...current, balloonMinSeverity: event.target.value }))}>
                <option value="info">Info</option>
                <option value="warning">Warning</option>
                <option value="critical">Critical</option>
              </select>
            </Field>
          </div>
          <Button
            type="button"
            disabled={configMutation.isPending}
            onClick={() =>
              configMutation.mutate({
                server: { open_browser: runtimeForm.openBrowser },
                monitoring: {
                  interval_ms: runtimeForm.intervalMS,
                  process_tree_interval_ms: runtimeForm.processTreeIntervalMS,
                  port_scan_interval_ms: runtimeForm.portScanIntervalMS,
                  gpu_interval_ms: runtimeForm.gpuIntervalMS,
                  history_duration_sec: runtimeForm.historyDurationSec,
                  max_processes: runtimeForm.maxProcesses,
                },
                controller: { confirm_kill_system: runtimeForm.confirmKillSystem },
                notifications: {
                  tray_balloon: runtimeForm.trayBalloon,
                  balloon_rate_limit_sec: runtimeForm.balloonRateLimitSec,
                  balloon_min_severity: runtimeForm.balloonMinSeverity,
                },
                ui: {
                  theme: runtimeForm.theme,
                  default_sort: runtimeForm.defaultSort,
                  default_sort_order: runtimeForm.defaultSortOrder,
                  sparkline_points: runtimeForm.sparklinePoints,
                  process_table_page_size: runtimeForm.processTablePageSize,
                  refresh_rate_ms: runtimeForm.refreshRateMS,
                },
              })
            }
          >
            Save runtime settings
          </Button>
        </Card>
      </div>
    </div>
  );
}

interface MetricCardProps {
  label: string;
  value: string;
  badge: string;
  badgeVariant: "neutral" | "info" | "success" | "warning";
}

function MetricCard({ label, value, badge, badgeVariant }: MetricCardProps) {
  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-secondary">{label}</div>
          <div className="mt-3 text-2xl font-semibold tracking-tight text-foreground">{value}</div>
        </div>
        <Badge variant={badgeVariant}>{badge}</Badge>
      </div>
    </Card>
  );
}

interface SectionTitleProps {
  icon: typeof Bot;
  title: string;
  description: string;
}

function SectionTitle({ icon: Icon, title, description }: SectionTitleProps) {
  return (
    <div className="flex items-start gap-3">
      <div className="rounded-2xl bg-accent-muted p-3 text-accent">
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <div className="text-lg font-semibold tracking-tight text-foreground">{title}</div>
        <div className="text-sm text-secondary">{description}</div>
      </div>
    </div>
  );
}

interface FieldProps {
  label: string;
  children: ReactNode;
}

function Field({ label, children }: FieldProps) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-foreground">{label}</span>
      {children}
    </label>
  );
}

interface ToggleProps {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}

function Toggle({ label, checked, onChange }: ToggleProps) {
  return (
    <label className="flex min-h-11 items-center justify-between rounded-2xl border border-border bg-background px-4 py-3 text-sm text-foreground">
      <span>{label}</span>
      <input aria-label={label} type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
    </label>
  );
}
