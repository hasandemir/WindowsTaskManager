// Windows Task Manager dashboard.
// Single ES module — talks to /api/v1/* and /api/v1/stream (SSE).

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => Array.from(document.querySelectorAll(sel));

const state = {
  snapshot: null,
  history: { cpu: [], mem: [] },
  maxHistory: 60,
  filter: "",
  sort: "cpu",
  activeTab: "overview",
  // Minimum ms between full DOM re-renders. User-adjustable from the topbar.
  // 1000ms is comfortable on modest hardware; the SSE stream still comes at
  // its own cadence (monitoring.interval), we just coalesce extra paints.
  renderMinInterval: parseInt(localStorage.getItem("wtm.renderMs") || "1000", 10),
  renderPending: false,
  lastRender: 0,
  lastUpdate: 0,
};

function fmtBytes(n) {
  if (!n || n === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(n) / Math.log(1024));
  return `${(n / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}
function fmtRate(bps) {
  return `${fmtBytes(bps)}/s`;
}

function el(tag, attrs, ...children) {
  const node = document.createElement(tag);
  if (attrs) {
    for (const [k, v] of Object.entries(attrs)) {
      if (k === "class") node.className = v;
      else if (k === "dataset") Object.assign(node.dataset, v);
      else if (k.startsWith("on")) node.addEventListener(k.slice(2), v);
      else node.setAttribute(k, v);
    }
  }
  for (const c of children) {
    if (c == null) continue;
    if (typeof c === "string" || typeof c === "number") {
      node.appendChild(document.createTextNode(String(c)));
    } else {
      node.appendChild(c);
    }
  }
  return node;
}
function clear(node) { while (node.firstChild) node.removeChild(node.firstChild); }

function setupTabs() {
  $$(".tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      $$(".tab").forEach((b) => b.classList.remove("active"));
      $$(".tab-pane").forEach((p) => p.classList.remove("active"));
      btn.classList.add("active");
      const tab = btn.dataset.tab;
      $(`#tab-${tab}`).classList.add("active");
      state.activeTab = tab;
      // Render the freshly-visible tab immediately so the user doesn't see a
      // stale snapshot while waiting for the next render tick.
      if (state.snapshot) renderAll(state.snapshot);
    });
  });
}

function setConn(online) {
  const elx = $("#conn");
  elx.textContent = online ? "online" : "offline";
  elx.className = "badge " + (online ? "online" : "offline");
}

function tickClock() {
  $("#clock").textContent = new Date().toLocaleTimeString();
}
setInterval(tickClock, 1000);
tickClock();

// ----- rendering -----

function renderOverview(snap) {
  $("#cpu-total").textContent = snap.cpu.total_percent.toFixed(1);
  $("#cpu-name").textContent = snap.cpu.name || "—";
  $("#mem-pct").textContent = snap.memory.used_percent.toFixed(1);
  $("#mem-used").textContent = fmtBytes(snap.memory.used_phys);
  $("#mem-total").textContent = fmtBytes(snap.memory.total_phys);
  $("#gpu-pct").textContent = snap.gpu && snap.gpu.utilization ? snap.gpu.utilization.toFixed(0) : "—";
  $("#gpu-name").textContent = snap.gpu ? snap.gpu.name : "—";
  $("#net-down").textContent = fmtRate(snap.network.total_down_bps || 0);
  $("#net-up").textContent = fmtRate(snap.network.total_up_bps || 0);
  $("#net-iface").textContent = (snap.network.interfaces || []).length;

  state.history.cpu.push(snap.cpu.total_percent);
  state.history.mem.push(snap.memory.used_percent);
  if (state.history.cpu.length > state.maxHistory) state.history.cpu.shift();
  if (state.history.mem.length > state.maxHistory) state.history.mem.shift();
  drawSpark($("#cpu-spark"), state.history.cpu, "#58a6ff");
  drawSpark($("#mem-spark"), state.history.mem, "#3fb950");

  const cores = $("#cores");
  clear(cores);
  (snap.cpu.per_core || []).forEach((p, i) => {
    const fill = el("span", { class: "fill" });
    fill.style.width = Math.min(100, p) + "%";
    cores.appendChild(el("div", { class: "core" },
      String(i),
      el("br"),
      `${p.toFixed(0)}%`,
      el("div", { class: "bar" }, fill),
    ));
  });
}

function drawSpark(canvas, data, color) {
  if (!canvas) return;
  const w = canvas.width = canvas.clientWidth;
  const h = canvas.height;
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, w, h);
  if (data.length < 2) return;
  ctx.strokeStyle = color;
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  data.forEach((v, i) => {
    const x = (i / (state.maxHistory - 1)) * w;
    const y = h - (v / 100) * h;
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });
  ctx.stroke();
}

function renderProcesses(snap) {
  let procs = (snap.processes || []).slice();
  const filter = state.filter.toLowerCase();
  if (filter) procs = procs.filter((p) => p.name.toLowerCase().includes(filter));
  procs.sort((a, b) => {
    switch (state.sort) {
      case "cpu": return b.cpu_percent - a.cpu_percent;
      case "memory": return b.working_set - a.working_set;
      case "name": return a.name.localeCompare(b.name);
      case "pid": return a.pid - b.pid;
      case "threads": return b.thread_count - a.thread_count;
    }
    return 0;
  });
  procs = procs.slice(0, 250);
  const tbody = $("#proc-body");
  clear(tbody);
  for (const p of procs) {
    const actions = el("td", { class: "actions" });
    for (const act of ["kill", "suspend", "resume"]) {
      actions.appendChild(el("button", { dataset: { act, pid: String(p.pid) } }, act[0].toUpperCase() + act.slice(1)));
    }
    tbody.appendChild(el("tr", null,
      el("td", null, String(p.pid)),
      el("td", null, p.name),
      el("td", null, p.cpu_percent.toFixed(1)),
      el("td", null, fmtBytes(p.working_set)),
      el("td", null, String(p.thread_count)),
      el("td", null, p.is_critical ? "critical" : (p.status || "")),
      actions,
    ));
  }
}

function renderTree(snap) {
  const root = $("#tree-root");
  clear(root);
  if (!snap.process_tree || snap.process_tree.length === 0) {
    root.appendChild(el("li", null, "(no tree)"));
    return;
  }
  for (const node of snap.process_tree) root.appendChild(renderNode(node));
}
function renderNode(node) {
  const text = `${node.process.name} (PID ${node.process.pid}) ${node.process.cpu_percent.toFixed(1)}%${node.is_orphan ? " [orphan]" : ""}`;
  const li = el("li", null, text);
  if (node.children && node.children.length > 0) {
    const ul = el("ul");
    for (const c of node.children) ul.appendChild(renderNode(c));
    li.appendChild(ul);
  }
  return li;
}

function renderPorts(snap) {
  const tbody = $("#ports-body");
  clear(tbody);
  const rows = (snap.port_bindings || []).slice(0, 500);
  for (const b of rows) {
    tbody.appendChild(el("tr", null,
      el("td", null, b.protocol),
      el("td", null, `${b.local_addr}:${b.local_port}`),
      el("td", null, (b.remote_addr || "") + (b.remote_port ? ":" + b.remote_port : "")),
      el("td", null, b.state || ""),
      el("td", null, String(b.pid)),
      el("td", null, b.process || ""),
      el("td", null, b.label || ""),
    ));
  }
}

function renderDisks(snap) {
  const wrap = $("#disks");
  clear(wrap);
  for (const d of (snap.disk && snap.disk.drives) || []) {
    const fill = el("span", { class: "fill" });
    fill.style.width = Math.min(100, d.used_pct) + "%";
    wrap.appendChild(el("div", { class: "disk" },
      el("strong", null, d.letter),
      " ",
      d.label || "",
      ` (${d.fs_type})`,
      el("div", { class: "muted" }, `${fmtBytes(d.used_bytes)} / ${fmtBytes(d.total_bytes)} (${d.used_pct.toFixed(1)}%)`),
      el("div", { class: "bar" }, fill),
    ));
  }
}

async function loadAlerts() {
  try {
    const [active, history] = await Promise.all([
      fetch("/api/v1/alerts").then((r) => r.json()),
      fetch("/api/v1/alerts/history").then((r) => r.json()),
    ]);
    renderAlertList($("#alerts-active"), active);
    renderAlertList($("#alerts-history"), (history || []).slice().reverse().slice(0, 50));
  } catch (e) { /* ignore */ }
}
function renderAlertList(ul, items) {
  clear(ul);
  if (!items || items.length === 0) {
    ul.appendChild(el("li", { class: "muted" }, "— none —"));
    return;
  }
  for (const a of items) {
    ul.appendChild(el("li", { class: a.severity || "" },
      el("div", { class: "alert-title" }, `[${a.severity}] ${a.title}`),
      el("div", { class: "alert-desc" }, a.description || ""),
    ));
  }
}

// ----- Rules -----

let rulesModel = [];

const metricOptions = [
  ["cpu_percent", "CPU %"],
  ["memory_bytes", "Memory (bytes)"],
  ["private_bytes", "Private (bytes)"],
  ["thread_count", "Threads"],
];
const opOptions = [">=", ">", "<=", "<"];
const actionOptions = ["alert", "kill", "suspend"];

async function loadRules() {
  try {
    const r = await fetch("/api/v1/rules").then((r) => r.json());
    rulesModel = (r.rules || []).map(normalizeRule);
    renderRules();
  } catch (e) { /* ignore */ }
}

function normalizeRule(r) {
  return {
    name: r.name || "",
    enabled: r.enabled !== false,
    match: r.match || "",
    metric: r.metric || "memory_bytes",
    op: r.op || ">=",
    threshold: Number(r.threshold) || 0,
    for_seconds: Number(r.for_seconds) || 0,
    action: r.action || "alert",
    cooldown_seconds: Number(r.cooldown_seconds) || 60,
  };
}

function renderRules() {
  const tbody = $("#rules-body");
  if (!tbody) return;
  clear(tbody);
  if (rulesModel.length === 0) {
    tbody.appendChild(el("tr", null, el("td", { colspan: "10", class: "muted" }, "No rules defined. Click + Add rule to create one.")));
    return;
  }
  rulesModel.forEach((rule, idx) => {
    const row = el("tr");
    row.appendChild(el("td", null, el("input", {
      type: "checkbox",
      ...(rule.enabled ? { checked: "" } : {}),
      oninput: (e) => { rulesModel[idx].enabled = e.target.checked; },
    })));
    row.appendChild(el("td", null, el("input", {
      type: "text", value: rule.name, placeholder: "e.g. chrome mem cap",
      oninput: (e) => { rulesModel[idx].name = e.target.value; },
    })));
    row.appendChild(el("td", null, el("input", {
      type: "text", value: rule.match, placeholder: "chrome.exe",
      oninput: (e) => { rulesModel[idx].match = e.target.value; },
    })));
    const metricSel = el("select", {
      oninput: (e) => { rulesModel[idx].metric = e.target.value; },
    });
    for (const [v, l] of metricOptions) {
      const opt = el("option", { value: v }, l);
      if (v === rule.metric) opt.selected = true;
      metricSel.appendChild(opt);
    }
    row.appendChild(el("td", null, metricSel));
    const opSel = el("select", {
      oninput: (e) => { rulesModel[idx].op = e.target.value; },
    });
    for (const o of opOptions) {
      const opt = el("option", { value: o }, o);
      if (o === rule.op) opt.selected = true;
      opSel.appendChild(opt);
    }
    row.appendChild(el("td", null, opSel));
    row.appendChild(el("td", null, el("input", {
      type: "number", value: String(rule.threshold), step: "1",
      oninput: (e) => { rulesModel[idx].threshold = Number(e.target.value) || 0; },
    })));
    row.appendChild(el("td", null, el("input", {
      type: "number", value: String(rule.for_seconds), min: "0", max: "86400",
      oninput: (e) => { rulesModel[idx].for_seconds = Number(e.target.value) || 0; },
    })));
    const actSel = el("select", {
      oninput: (e) => { rulesModel[idx].action = e.target.value; },
    });
    for (const a of actionOptions) {
      const opt = el("option", { value: a }, a);
      if (a === rule.action) opt.selected = true;
      actSel.appendChild(opt);
    }
    row.appendChild(el("td", null, actSel));
    row.appendChild(el("td", null, el("input", {
      type: "number", value: String(rule.cooldown_seconds), min: "0",
      oninput: (e) => { rulesModel[idx].cooldown_seconds = Number(e.target.value) || 0; },
    })));
    row.appendChild(el("td", null, el("button", {
      type: "button", class: "action-del",
      onclick: () => { rulesModel.splice(idx, 1); renderRules(); },
    }, "×")));
    tbody.appendChild(row);
  });
}

function addRule() {
  rulesModel.push(normalizeRule({
    name: "new rule " + (rulesModel.length + 1),
    enabled: false,
    match: "",
    metric: "memory_bytes",
    op: ">=",
    threshold: 0,
    for_seconds: 30,
    action: "alert",
    cooldown_seconds: 60,
  }));
  renderRules();
}

async function saveRules() {
  const msg = $("#rules-save-msg");
  msg.textContent = "saving…";
  msg.style.color = "";
  try {
    const r = await fetch("/api/v1/rules", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ rules: rulesModel }),
    });
    const data = await r.json();
    if (!r.ok) throw new Error((data.error && data.error.message) || r.statusText);
    const stamp = new Date().toLocaleTimeString();
    msg.textContent = `✓ saved ${rulesModel.length} rule(s) at ${stamp}`;
    msg.style.color = "var(--ok)";
  } catch (e) {
    msg.textContent = `✗ ${e.message}`;
    msg.style.color = "var(--crit)";
  }
}

// ----- AI -----

let aiPresets = [];
let aiModels = [];

async function loadAIStatus() {
  try {
    const s = await fetch("/api/v1/ai/status").then((r) => r.json());
    const provider = s.provider || "—";
    const model = s.model || "—";
    updateAIPill(s);
    if (!s.enabled) {
      $("#ai-status").textContent = `AI advisor disabled — provider=${provider} model=${model}`;
      return;
    }
    const cfgState = s.configured ? "configured" : "no key";
    $("#ai-status").textContent =
      `provider=${provider} model=${model} (${cfgState}) tokens=${s.tokens_available}/min cache=${s.cache_size}` +
      (s.last_error ? ` — last error: ${s.last_error}` : "");
  } catch (e) {
    $("#ai-status").textContent = "AI status unavailable";
  }
}

function updateAIPill(s) {
  const pill = $("#ai-pill");
  if (!pill) return;
  const provider = s.provider || "ai";
  const model = s.model || "—";
  pill.hidden = false;
  pill.textContent = `${provider}: ${model}`;
  pill.classList.toggle("disabled", !s.enabled || !s.configured);
  pill.title = s.enabled
    ? `${provider} (${s.configured ? "configured" : "no key"}) — click to open AI tab`
    : "AI advisor disabled — click to configure";
}

async function loadAIPresets() {
  try {
    aiPresets = await fetch("/api/v1/ai/presets").then((r) => r.json());
    const sel = $("#ai-preset");
    clear(sel);
    sel.appendChild(el("option", { value: "" }, "— pick a starter —"));
    for (const p of aiPresets) {
      sel.appendChild(el("option", { value: p.id }, p.label));
    }
  } catch (e) { /* ignore */ }
}

async function loadAIConfig() {
  try {
    const c = await fetch("/api/v1/ai/config").then((r) => r.json());
    $("#ai-enabled").checked = !!c.enabled;
    $("#ai-provider").value = c.provider || "anthropic";
    $("#ai-endpoint").value = c.endpoint || "";
    $("#ai-model").value = c.model || "";
    // Don't wipe what the user just typed (e.g. mid-edit reload). Only clear
    // the password field if the form has a stored key already.
    if (c.api_key && document.activeElement !== $("#ai-apikey")) {
      $("#ai-apikey").value = "";
    }
    const state = $("#ai-apikey-state");
    if (c.api_key) {
      state.textContent = `(current: ${c.api_key} — leave blank to keep)`;
      $("#ai-apikey").placeholder = "leave blank to keep current";
    } else {
      state.textContent = "(no key set)";
      $("#ai-apikey").placeholder = "paste key";
    }
    $("#ai-language").value = c.language || "tr";
    $("#ai-maxtokens").value = c.max_tokens || 1024;
    $("#ai-rpm").value = c.max_requests_per_minute || 5;
    $("#ai-headers").value = headersToText(c.extra_headers);
    refreshModelDatalist();
  } catch (e) { /* ignore */ }
}

async function loadAIModels() {
  try {
    const r = await fetch("/api/v1/ai/models").then((r) => r.json());
    aiModels = Array.isArray(r.models) ? r.models : [];
    if (aiModels.length === 0 && !r.error) {
      // Cold start — models.dev fetch is in flight. Retry shortly.
      setTimeout(loadAIModels, 2500);
      return;
    }
    refreshModelDatalist();
  } catch (e) { /* ignore */ }
}

function refreshModelDatalist() {
  const list = $("#ai-model-list");
  if (!list) return;
  const provider = $("#ai-provider").value || "anthropic";
  const filtered = aiModels.filter((m) => m.format === provider);
  clear(list);
  for (const m of filtered) {
    const ctx = m.context ? ` — ${(m.context / 1000).toFixed(0)}k ctx` : "";
    list.appendChild(el("option", { value: m.id }, `${m.provider}${ctx}`));
  }
  const count = $("#ai-model-count");
  if (count) {
    count.textContent = filtered.length
      ? `(${filtered.length} from models.dev)`
      : aiModels.length ? "(no matches for this provider)" : "(loading models.dev…)";
  }
}

function applyModelEndpoint(modelID) {
  const m = aiModels.find((x) => x.id === modelID);
  if (!m) return;
  // Only fill the endpoint if the user hasn't already set one — we don't
  // want to surprise them by overwriting their custom URL.
  if (!$("#ai-endpoint").value.trim() && m.endpoint) {
    $("#ai-endpoint").value = m.endpoint;
  }
  if (m.format && m.format !== $("#ai-provider").value) {
    $("#ai-provider").value = m.format;
    refreshModelDatalist();
  }
}

function headersToText(obj) {
  if (!obj) return "";
  return Object.entries(obj).map(([k, v]) => `${k}: ${v}`).join("\n");
}
function textToHeaders(text) {
  const out = {};
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim();
    if (!line) continue;
    const idx = line.indexOf(":");
    if (idx <= 0) continue;
    const k = line.slice(0, idx).trim();
    const v = line.slice(idx + 1).trim();
    if (k) out[k] = v;
  }
  return out;
}

function applyPreset(id) {
  const p = aiPresets.find((x) => x.id === id);
  if (!p) return;
  // Non-destructive: if the user already has values in the form, preserve
  // them. Presets are *starting points*, not "reset to defaults" buttons.
  // Only empty fields get filled. The note and key hint always update.
  $("#ai-preset-note").textContent = p.notes || "";
  $("#ai-apikey").placeholder = p.api_key_hint || "paste key";

  const hasValues = $("#ai-endpoint").value.trim() || $("#ai-model").value.trim();
  if (hasValues) {
    // Show an "apply anyway" button so the user can opt in to clobber.
    const note = $("#ai-preset-note");
    note.textContent = `${p.notes || ""} — form already has values; click Apply to overwrite.`;
    note.appendChild(el("button", {
      type: "button",
      class: "preset-apply",
      onclick: () => forceApplyPreset(p),
    }, " Apply"));
    return;
  }
  forceApplyPreset(p);
}

function forceApplyPreset(p) {
  $("#ai-provider").value = p.provider;
  $("#ai-endpoint").value = p.endpoint || "";
  $("#ai-model").value = p.model || "";
  $("#ai-headers").value = headersToText(p.extra_headers || {});
  $("#ai-preset-note").textContent = p.notes || "";
  refreshModelDatalist();
}

async function saveAIConfig() {
  const msg = $("#ai-save-msg");
  const form = $("#ai-settings");
  msg.textContent = "saving…";
  msg.style.color = "";
  form.classList.remove("saved", "save-error");
  const body = {
    enabled: $("#ai-enabled").checked,
    provider: $("#ai-provider").value,
    endpoint: $("#ai-endpoint").value.trim(),
    model: $("#ai-model").value.trim(),
    api_key: $("#ai-apikey").value, // empty = keep current
    language: $("#ai-language").value,
    max_tokens: parseInt($("#ai-maxtokens").value, 10) || 0,
    max_requests_per_minute: parseInt($("#ai-rpm").value, 10) || 0,
    extra_headers: textToHeaders($("#ai-headers").value),
    include_process_tree: true,
    include_port_map: true,
  };
  try {
    const r = await fetch("/api/v1/ai/config", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = await r.json();
    if (!r.ok) throw new Error((data.error && data.error.message) || r.statusText);
    const stamp = new Date().toLocaleTimeString();
    msg.textContent = `✓ saved at ${stamp} — config written to disk`;
    msg.style.color = "var(--ok)";
    form.classList.add("saved");
    setTimeout(() => form.classList.remove("saved"), 2500);
    // Clear the password field only AFTER repaint so the user sees their
    // typed value briefly, then loadAIConfig() will refresh the masked state.
    $("#ai-apikey").blur();
    $("#ai-apikey").value = "";
    loadAIConfig();
    loadAIStatus();
  } catch (e) {
    msg.textContent = `✗ error: ${e.message}`;
    msg.style.color = "var(--crit)";
    form.classList.add("save-error");
  }
}

async function sendAI() {
  const promptText = $("#ai-prompt").value.trim();
  $("#ai-answer").textContent = "thinking…";
  try {
    const r = await fetch("/api/v1/ai/analyze", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ prompt: promptText }),
    });
    const body = await r.json();
    if (!r.ok) throw new Error((body.error && body.error.message) || r.statusText);
    $("#ai-answer").textContent = body.answer || "(empty)";
    loadAIStatus();
  } catch (e) {
    $("#ai-answer").textContent = `error: ${e.message}`;
  }
}

// ----- process actions -----

async function processAction(action, pid) {
  const url = `/api/v1/processes/${pid}/${action}?confirm=true`;
  const opts = { method: "POST" };
  if (action === "priority") {
    const cls = window.prompt("Priority class (idle/below_normal/normal/above_normal/high)", "normal");
    if (!cls) return;
    opts.headers = { "Content-Type": "application/json" };
    opts.body = JSON.stringify({ class: cls });
  }
  try {
    const r = await fetch(url, opts);
    const body = await r.json();
    if (!r.ok) alert((body.error && body.error.message) || r.statusText);
  } catch (e) { alert(e.message); }
}

// ----- entry -----

function applySnapshot(snap) {
  state.snapshot = snap;
  state.lastUpdate = Date.now();
  // Throttle: if we rendered less than `state.renderMinInterval` ms ago,
  // just stash the snapshot and let the next tick absorb it. Keeps the UI
  // responsive on slow machines without dropping data.
  const now = Date.now();
  if (state.renderPending) return;
  const wait = Math.max(0, (state.lastRender || 0) + state.renderMinInterval - now);
  if (wait > 0) {
    state.renderPending = true;
    setTimeout(() => {
      state.renderPending = false;
      renderAll(state.snapshot);
    }, wait);
    return;
  }
  renderAll(snap);
}

function renderAll(snap) {
  if (!snap) return;
  state.lastRender = Date.now();
  renderOverview(snap);
  if (state.activeTab === "processes") renderProcesses(snap);
  if (state.activeTab === "tree") renderTree(snap);
  if (state.activeTab === "ports") renderPorts(snap);
  if (state.activeTab === "disks") renderDisks(snap);
}

function setupSSE() {
  const es = new EventSource("/api/v1/stream");
  es.addEventListener("hello", () => setConn(true));
  // Any inbound data proves the socket is alive — don't gate "online" on hello
  // alone (if the client connects mid-stream or misses hello we'd hang offline).
  es.addEventListener("metrics.snapshot", (e) => {
    setConn(true);
    try { applySnapshot(JSON.parse(e.data)); } catch (err) { /* ignore */ }
  });
  es.addEventListener("anomaly.raised", () => { setConn(true); loadAlerts(); });
  es.addEventListener("anomaly.cleared", () => { setConn(true); loadAlerts(); });
  es.onerror = () => setConn(false);

  // Fallback poll: if the SSE stream is silent (bad middleware, proxy buffering,
  // etc.) pull snapshots directly so the dashboard still lives. We drop the
  // poll as soon as a real SSE snapshot arrives.
  let pollTimer = setInterval(async () => {
    if (state.snapshot && Date.now() - (state.lastUpdate || 0) < 3000) return;
    try {
      const snap = await fetch("/api/v1/system").then((r) => r.json());
      applySnapshot(snap);
      setConn(true);
    } catch (e) { setConn(false); }
  }, 2000);
  es.addEventListener("metrics.snapshot", () => {
    if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
  }, { once: true });
}

async function bootstrap() {
  setupTabs();
  $("#proc-filter").addEventListener("input", (e) => {
    state.filter = e.target.value;
    if (state.snapshot) renderProcesses(state.snapshot);
  });
  $("#proc-sort").addEventListener("change", (e) => {
    state.sort = e.target.value;
    if (state.snapshot) renderProcesses(state.snapshot);
  });
  $("#proc-body").addEventListener("click", (e) => {
    const btn = e.target.closest("button");
    if (!btn) return;
    processAction(btn.dataset.act, parseInt(btn.dataset.pid, 10));
  });
  $("#ai-send").addEventListener("click", sendAI);
  $("#ai-save").addEventListener("click", saveAIConfig);
  $("#ai-preset").addEventListener("change", (e) => applyPreset(e.target.value));

  const rateSel = $("#refresh-rate");
  rateSel.value = String(state.renderMinInterval);
  rateSel.addEventListener("change", (e) => {
    state.renderMinInterval = parseInt(e.target.value, 10) || 1000;
    localStorage.setItem("wtm.renderMs", String(state.renderMinInterval));
  });

  $("#rules-add").addEventListener("click", addRule);
  $("#rules-save").addEventListener("click", saveRules);
  $("#ai-provider").addEventListener("change", refreshModelDatalist);
  $("#ai-model").addEventListener("change", (e) => applyModelEndpoint(e.target.value));
  $("#ai-pill").addEventListener("click", () => {
    const btn = document.querySelector('.tab[data-tab="ai"]');
    if (btn) btn.click();
    const det = $("#ai-settings");
    if (det) det.open = true;
  });

  try {
    const snap = await fetch("/api/v1/system").then((r) => r.json());
    applySnapshot(snap);
  } catch (e) { /* ignore */ }
  loadAlerts();
  loadRules();
  loadAIPresets().then(loadAIConfig);
  loadAIModels();
  loadAIStatus();
  setInterval(loadAlerts, 5000);
  setInterval(loadAIStatus, 10000);
  setupSSE();
}

bootstrap();
