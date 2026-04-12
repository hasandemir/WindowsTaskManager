import type { AlertItem, PortBinding, SystemSnapshot } from "../types/api";

const testPortBindings: PortBinding[] = [
  {
    protocol: "tcp",
    local_addr: "127.0.0.1",
    local_port: 3000,
    remote_addr: "",
    remote_port: 0,
    state: "LISTEN",
    pid: 101,
    process: "chrome.exe",
    label: "chrome.exe",
  },
  {
    protocol: "tcp",
    local_addr: "127.0.0.1",
    local_port: 9222,
    remote_addr: "127.0.0.1",
    remote_port: 51001,
    state: "ESTABLISHED",
    pid: 101,
    process: "chrome.exe",
    label: "chrome.exe",
  },
];

export const testSnapshot: SystemSnapshot = {
  timestamp: "2026-04-12T07:00:00Z",
  cpu: {
    total_percent: 41.2,
    per_core: [33, 52],
    num_logical: 2,
    name: "Test CPU",
  },
  memory: {
    used_phys: 8 * 1024 * 1024 * 1024,
    total_phys: 16 * 1024 * 1024 * 1024,
    used_percent: 50,
  },
  gpu: {
    name: "Test GPU",
    utilization: 18,
    vram_used: 2 * 1024 * 1024 * 1024,
    vram_total: 8 * 1024 * 1024 * 1024,
    available: true,
  },
  disk: {
    drives: [],
  },
  network: {
    total_up_bps: 256_000,
    total_down_bps: 1_024_000,
    interfaces: [
      {
        name: "Ethernet",
        type: "ethernet",
        status: "up",
        speed_mbps: 1000,
        in_bps: 1_024_000,
        out_bps: 256_000,
        in_pps: 90,
        out_pps: 45,
      },
      {
        name: "Wi-Fi",
        type: "wifi",
        status: "up",
        speed_mbps: 300,
        in_bps: 128_000,
        out_bps: 64_000,
        in_pps: 12,
        out_pps: 6,
      },
    ],
  },
  processes: [
    {
      pid: 101,
      name: "chrome.exe",
      cpu_percent: 21.7,
      working_set: 2_500_000_000,
      thread_count: 32,
      connections: 14,
      is_critical: false,
    },
    {
      pid: 4,
      name: "System",
      cpu_percent: 2.2,
      working_set: 120_000_000,
      thread_count: 145,
      connections: 0,
      is_critical: true,
    },
  ],
  process_tree: [],
  port_bindings: testPortBindings,
};

export const testAlerts: { active: AlertItem[]; history: AlertItem[] } = {
  active: [
    {
      type: "cpu_spike",
      pid: 101,
      severity: "critical",
      title: "CPU spike detected",
      description: "chrome.exe exceeded the configured CPU threshold.",
    },
    {
      type: "memory_growth",
      pid: 202,
      severity: "warning",
      title: "Memory growth observed",
      description: "Agent memory footprint is still increasing.",
    },
  ],
  history: [
    {
      type: "port_burst",
      severity: "info",
      title: "Port burst resolved",
      description: "A short-lived port burst was observed and cleared.",
    },
  ],
};
