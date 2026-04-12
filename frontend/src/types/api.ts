export interface CpuMetrics {
  total_percent: number;
  per_core: number[];
  num_logical: number;
  name: string;
}

export interface MemoryMetrics {
  used_phys: number;
  total_phys: number;
  used_percent: number;
}

export interface GPUMetrics {
  name: string;
  utilization: number;
  vram_used: number;
  vram_total: number;
  available: boolean;
}

export interface DriveInfo {
  letter: string;
  label: string;
  fs_type: string;
  used_pct: number;
  used_bytes: number;
  total_bytes: number;
  read_bps: number;
  write_bps: number;
}

export interface DiskMetrics {
  drives: DriveInfo[];
}

export interface InterfaceInfo {
  name: string;
  type: string;
  status: string;
  speed_mbps: number;
  in_bps: number;
  out_bps: number;
  in_pps: number;
  out_pps: number;
}

export interface NetworkMetrics {
  interfaces: InterfaceInfo[];
  total_up_bps: number;
  total_down_bps: number;
}

export interface ProcessInfo {
  pid: number;
  name: string;
  cpu_percent: number;
  working_set: number;
  thread_count: number;
  connections: number;
  is_critical: boolean;
}

export interface InfoResponse {
  version: string;
  go_version: string;
  num_cpu: number;
  goroutines: number;
  self_pid: number;
  interval_ms: number;
  history_minutes: number;
  sse_clients: number;
  tracked_pids: number;
}

export interface ProcessNode {
  process: ProcessInfo;
  children?: ProcessNode[];
  is_orphan: boolean;
}

export interface PortBinding {
  protocol: string;
  local_addr: string;
  local_port: number;
  remote_addr: string;
  remote_port: number;
  state: string;
  pid: number;
  process: string;
  label: string;
}

export interface SystemSnapshot {
  timestamp: string;
  cpu: CpuMetrics;
  memory: MemoryMetrics;
  gpu: GPUMetrics;
  disk: DiskMetrics;
  network: NetworkMetrics;
  processes: ProcessInfo[];
  process_tree?: ProcessNode[];
  port_bindings?: PortBinding[];
}

export interface AlertItem {
  type: string;
  pid?: number;
  severity: string;
  title: string;
  description: string;
  action?: string;
}

export interface Rule {
  name: string;
  enabled: boolean;
  match: string;
  metric: string;
  op: string;
  threshold: number;
  for_seconds: number;
  action: string;
  cooldown_seconds: number;
}

export interface RulesResponse {
  rules: Rule[];
}

export interface AIStatus {
  enabled: boolean;
  configured?: boolean;
  provider?: string;
  model?: string;
  tokens_available?: number;
  cache_size?: number;
}

export interface AIConfig {
  enabled: boolean;
  provider: string;
  api_key: string;
  model: string;
  endpoint: string;
  extra_headers: Record<string, string>;
  language: string;
  max_tokens: number;
  max_requests_per_minute: number;
  include_process_tree: boolean;
  include_port_map: boolean;
}

export interface RuleSuggestion {
  name: string;
  enabled: boolean;
  match: string;
  metric: string;
  op: string;
  threshold: number;
  for?: number;
  for_seconds?: number;
  action: string;
  cooldown?: number;
  cooldown_seconds?: number;
}

export interface AISuggestion {
  id: string;
  type: string;
  pid?: number;
  name?: string;
  reason?: string;
  rule?: RuleSuggestion;
}

export interface AIResult {
  answer: string;
  actions?: AISuggestion[];
  cached?: boolean;
}

export interface AIPreset {
  id: string;
  label: string;
  provider: string;
  endpoint: string;
  model: string;
  api_key_hint: string;
  extra_headers?: Record<string, string>;
  notes?: string;
}

export interface TelegramConfig {
  enabled: boolean;
  bot_token: string;
  allowed_chat_ids: number[];
  api_base_url: string;
  poll_timeout_sec: number;
  notify_on_critical: boolean;
  notification_mode: string;
  notification_types: string[];
  require_confirm: boolean;
  confirm_ttl_sec: number;
}

export interface AppConfig {
  Server: {
    OpenBrowser: boolean;
  };
  Monitoring: {
    Interval: number;
    ProcessTreeInterval: number;
    PortScanInterval: number;
    GPUInterval: number;
    HistoryDuration: number;
    MaxProcesses: number;
  };
  Controller: {
    ConfirmKillSystem: boolean;
    ProtectedProcesses: string[];
  };
  Anomaly: {
    IgnoreProcesses: string[];
  };
  Notifications: {
    TrayBalloon: boolean;
    BalloonRateLimit: number;
    BalloonMinSeverity: string;
  };
  UI: {
    Theme: string;
    DefaultSort: string;
    DefaultSortOrder: string;
    SparklinePoints: number;
    ProcessTablePageSize: number;
    RefreshRate: number;
  };
}
