// Windows Task Manager dashboard.
// Single ES module — talks to /api/v1/* and /api/v1/stream (SSE).

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => Array.from(document.querySelectorAll(sel));

const state = {
  snapshot: null,
  selfPID: null,
  history: { cpu: [], mem: [] },
  maxHistory: 60,
  filter: "",
  sort: localStorage.getItem("wtm.sortCol") || "cpu",
  sortDir: localStorage.getItem("wtm.sortDir") || "desc", // "asc" | "desc"
  activeTab: "overview",
  // Minimum ms between full DOM re-renders. User-adjustable from the topbar.
  // 1000ms is comfortable on modest hardware; the SSE stream still comes at
  // its own cadence (monitoring.interval), we just coalesce extra paints.
  renderMinInterval: parseInt(localStorage.getItem("wtm.renderMs") || "1000", 10),
  renderPending: false,
  lastRender: 0,
  lastUpdate: 0,
  // Mirror of the server config for UI decisions. Refreshed on every save and
  // on boot. Lowercased for fast EqualFold-style checks on the process table.
  protectedNames: new Set(),
  ignoredNames: new Set(),
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
function fmtDateTime(v) {
  if (!v) return "—";
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString();
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
  const dir = state.sortDir === "asc" ? 1 : -1;
  procs.sort((a, b) => {
    let cmp = 0;
    switch (state.sort) {
      case "cpu": cmp = a.cpu_percent - b.cpu_percent; break;
      case "memory": cmp = a.working_set - b.working_set; break;
      case "name": cmp = a.name.localeCompare(b.name); break;
      case "pid": cmp = a.pid - b.pid; break;
      case "threads": cmp = a.thread_count - b.thread_count; break;
    }
    return cmp * dir;
  });
  updateSortIndicators();
  procs = procs.slice(0, 250);
  const tbody = $("#proc-body");
  clear(tbody);
  for (const p of procs) {
    const nameLower = (p.name || "").toLowerCase();
    const isProtected = state.protectedNames.has(nameLower);
    const isIgnored = state.ignoredNames.has(nameLower);
    const isCritical = !!p.is_critical;
    const isSelf = state.selfPID != null && p.pid === state.selfPID;
    const noTouch = isCritical || isProtected || isSelf;

    const flagsCell = el("td", { class: "flags" });
    if (isSelf) flagsCell.appendChild(el("span", { class: "flag self", title: "Running Windows Task Manager process" }, "WTM"));
    if (isCritical) flagsCell.appendChild(el("span", { class: "flag crit", title: "Windows critical process" }, "⚠"));
    if (isProtected) flagsCell.appendChild(el("span", { class: "flag prot", title: "Protected — kill/suspend disabled" }, "🛡"));
    if (isIgnored) flagsCell.appendChild(el("span", { class: "flag ign", title: "Ignored by anomaly detectors" }, "🔕"));

    const actions = el("td", { class: "actions" });
    for (const act of ["kill", "suspend", "resume"]) {
      const disabled = noTouch && (act === "kill" || act === "suspend");
      const btnAttrs = { dataset: { act, pid: String(p.pid), name: p.name || "" } };
      if (disabled) {
        btnAttrs.disabled = "";
        btnAttrs.title = isSelf
          ? "WTM cannot kill or suspend its own running process"
          : isCritical
            ? "Windows system process — not allowed"
            : "Protected — remove from protect list to enable";
      }
      actions.appendChild(el("button", btnAttrs, act[0].toUpperCase() + act.slice(1)));
    }
    actions.appendChild(el("button", {
      class: isProtected ? "toggle on" : "toggle",
      dataset: { act: "protect", name: p.name || "", on: isProtected ? "1" : "0" },
      title: isProtected ? "Remove from protect list" : "Prevent kill/suspend for this name",
    }, isProtected ? "🛡 on" : "🛡"));
    actions.appendChild(el("button", {
      class: isIgnored ? "toggle on" : "toggle",
      dataset: { act: "ignore", name: p.name || "", on: isIgnored ? "1" : "0" },
      title: isIgnored ? "Remove from anomaly ignore list" : "Silence anomaly alerts for this name",
    }, isIgnored ? "🔕 on" : "🔕"));

    tbody.appendChild(el("tr", { class: noTouch ? "no-touch" : "" },
      el("td", null, String(p.pid)),
      el("td", null, p.name),
      el("td", null, p.cpu_percent.toFixed(1)),
      el("td", null, fmtBytes(p.working_set)),
      el("td", null, String(p.thread_count)),
      flagsCell,
      actions,
    ));
  }
}

// setSort updates the active sort column + direction. Clicking the same
// column toggles asc/desc; clicking a different column snaps to the sensible
// default for that column (name/pid → asc, numeric → desc) so the first
// click always shows what the user probably wanted.
function setSort(col) {
  if (state.sort === col) {
    state.sortDir = state.sortDir === "asc" ? "desc" : "asc";
  } else {
    state.sort = col;
    state.sortDir = (col === "name" || col === "pid") ? "asc" : "desc";
  }
  localStorage.setItem("wtm.sortCol", state.sort);
  localStorage.setItem("wtm.sortDir", state.sortDir);
  const sel = $("#proc-sort");
  if (sel) sel.value = state.sort;
  if (state.snapshot) renderProcesses(state.snapshot);
}

function updateSortIndicators() {
  const arrow = state.sortDir === "asc" ? "▲" : "▼";
  $$(".proc-table th.sortable").forEach((th) => {
    const ind = th.querySelector(".sort-ind");
    if (!ind) return;
    if (th.dataset.sort === state.sort) {
      ind.textContent = arrow;
      th.classList.add("active");
    } else {
      ind.textContent = "";
      th.classList.remove("active");
    }
  });
}

async function loadConfig() {
  try {
    const c = await fetch("/api/v1/config").then((r) => r.json());
    const prot = (c && c.Controller && c.Controller.ProtectedProcesses) || [];
    const ign = (c && c.Anomaly && c.Anomaly.IgnoreProcesses) || [];
    state.protectedNames = new Set(prot.map((x) => String(x).toLowerCase()));
    state.ignoredNames = new Set(ign.map((x) => String(x).toLowerCase()));
    if (state.snapshot && state.activeTab === "processes") renderProcesses(state.snapshot);
  } catch (e) { /* ignore */ }
}

async function toggleProtect(name, on) {
  try {
    const r = await fetch("/api/v1/config/protect", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, protect: on }),
    });
    if (!r.ok) {
      const b = await r.json().catch(() => ({}));
      alert((b.error && b.error.message) || r.statusText);
      return;
    }
    await loadConfig();
  } catch (e) { alert(e.message); }
}

async function toggleIgnore(name, on) {
  try {
    const r = await fetch("/api/v1/config/ignore", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, ignore: on }),
    });
    if (!r.ok) {
      const b = await r.json().catch(() => ({}));
      alert((b.error && b.error.message) || r.statusText);
      return;
    }
    await loadConfig();
  } catch (e) { alert(e.message); }
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

async function clearAllAlerts() {
  try {
    const r = await fetch("/api/v1/alerts/clear", { method: "POST" });
    if (!r.ok) {
      const b = await r.json().catch(() => ({}));
      alert((b.error && b.error.message) || r.statusText);
      return;
    }
    loadAlerts();
  } catch (e) { alert(e.message); }
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

function textToChatIDs(text) {
  const out = [];
  for (const raw of text.split(/\r?\n|,/)) {
    const trimmed = raw.trim();
    if (!trimmed) continue;
    const n = Number(trimmed);
    if (Number.isInteger(n)) out.push(n);
  }
  return out;
}

function chatIDsToText(ids) {
  return (ids || []).map((x) => String(x)).join("\n");
}

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

async function loadAIWatch() {
  try {
    const s = await fetch("/api/v1/ai/watch").then((r) => r.json());
    renderAIWatch(s || {});
  } catch (e) {
    $("#ai-watch-summary").textContent = "Background watch unavailable";
  }
}

function renderAIWatch(state) {
  const pill = $("#ai-watch-pill");
  const summary = $("#ai-watch-summary");
  const answer = $("#ai-watch-answer");
  const runs = $("#ai-watch-runs");

  const enabled = !!state.enabled;
  const configured = !!state.configured;
  const inFlight = !!state.in_flight;
  const budget = state.budget || {};
  let pillText = "disabled";
  let pillClass = "badge offline";
  if (enabled && configured) {
    pillText = inFlight ? "running" : "armed";
    pillClass = "badge online";
  } else if (enabled && !configured) {
    pillText = "no key";
    pillClass = "badge ai-pill disabled";
  }
  pill.textContent = pillText;
  pill.className = pillClass;

  const parts = [];
  if (!enabled) parts.push("Scheduler disabled");
  else if (!configured) parts.push("Scheduler enabled but AI is not configured");
  else parts.push("Critical alerts trigger a background AI pass");
  if (state.auto_action_enabled) {
    parts.push(state.auto_action_dry_run ? "auto-action policy: dry-run" : "auto-action policy: live mode requested");
  } else {
    parts.push("auto-action policy disabled");
  }
  parts.push(`cycles ${budget.cycles_last_hour || 0}/${budget.max_cycles_per_hour || 0} this hour`);
  parts.push(`tokens ${budget.reserved_tokens_today || 0}/${budget.max_reserved_tokens_per_day || 0} reserved today`);
  if (state.last_skip_reason) parts.push(`last skip: ${state.last_skip_reason}`);
  if (state.last_error) parts.push(`last error: ${state.last_error}`);
  summary.textContent = parts.join(" • ");

  const lastRun = state.last_run;
  if (lastRun && lastRun.answer) {
    answer.textContent = lastRun.answer;
    answer.hidden = false;
  } else {
    answer.textContent = "No completed background analysis yet.";
    answer.hidden = false;
  }

  renderAIActionsIn($("#ai-watch-actions"), lastRun && lastRun.actions, "Background suggestions");

  clear(runs);
  const items = (state.recent_runs || []).slice().reverse();
  if (items.length === 0) {
    runs.appendChild(el("li", { class: "muted" }, "No background runs recorded."));
    return;
  }
  for (const run of items) {
    const title = run.alert_title || run.alert_type || run.trigger || "background run";
    const status = run.error
      ? `error: ${run.error}`
      : `${run.actions ? run.actions.length : 0} action(s), ${run.auto_candidates || 0} dry-run candidate(s)`;
    const meta = [
      `started ${fmtDateTime(run.started_at)}`,
      run.cached ? "cache hit" : `reserved ${run.reserved_tokens || 0} tokens`,
      run.alert_process ? `process ${run.alert_process}` : "",
      run.alert_pid ? `PID ${run.alert_pid}` : "",
    ].filter(Boolean).join(" • ");
    runs.appendChild(el("li", null,
      el("div", { class: "run-head" },
        el("strong", null, title),
        el("span", { class: "muted" }, status),
      ),
      el("div", { class: "run-meta" }, meta),
    ));
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

async function loadInfo() {
  try {
    const info = await fetch("/api/v1/info").then((r) => r.json());
    state.selfPID = Number.isInteger(info.self_pid) ? info.self_pid : null;
    if (state.snapshot && state.activeTab === "processes") renderProcesses(state.snapshot);
  } catch (e) { /* ignore */ }
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

async function loadTelegramConfig() {
  try {
    const c = await fetch("/api/v1/telegram/config").then((r) => r.json());
    $("#tg-enabled").checked = !!c.enabled;
    $("#tg-api-base").value = c.api_base_url || "https://api.telegram.org";
    $("#tg-chat-ids").value = chatIDsToText(c.allowed_chat_ids || []);
    $("#tg-poll-timeout").value = c.poll_timeout_sec || 25;
    $("#tg-confirm-ttl").value = c.confirm_ttl_sec || 90;
    $("#tg-notify-critical").checked = c.notify_on_critical !== false;
    $("#tg-require-confirm").checked = c.require_confirm !== false;
    if (c.bot_token && document.activeElement !== $("#tg-token")) {
      $("#tg-token").value = "";
    }
    const state = $("#tg-token-state");
    if (c.bot_token) {
      state.textContent = `(current: ${c.bot_token} — leave blank to keep)`;
      $("#tg-token").placeholder = "leave blank to keep current";
    } else {
      state.textContent = "(no token set)";
      $("#tg-token").placeholder = "123456:ABC...";
    }
  } catch (e) { /* ignore */ }
}

async function saveTelegramConfig() {
  const msg = $("#tg-save-msg");
  const body = {
    enabled: $("#tg-enabled").checked,
    bot_token: $("#tg-token").value,
    api_base_url: $("#tg-api-base").value.trim(),
    allowed_chat_ids: textToChatIDs($("#tg-chat-ids").value),
    poll_timeout_sec: parseInt($("#tg-poll-timeout").value, 10) || 25,
    notify_on_critical: $("#tg-notify-critical").checked,
    require_confirm: $("#tg-require-confirm").checked,
    confirm_ttl_sec: parseInt($("#tg-confirm-ttl").value, 10) || 90,
  };
  msg.textContent = "saving…";
  msg.style.color = "";
  try {
    const r = await fetch("/api/v1/telegram/config", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = await r.json();
    if (!r.ok) throw new Error((data.error && data.error.message) || r.statusText);
    msg.textContent = `✓ saved at ${new Date().toLocaleTimeString()}`;
    msg.style.color = "var(--ok)";
    $("#tg-token").blur();
    $("#tg-token").value = "";
    loadTelegramConfig();
  } catch (e) {
    msg.textContent = `✗ error: ${e.message}`;
    msg.style.color = "var(--crit)";
  }
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
    language: "en",
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
  clear($("#ai-actions"));
  try {
    const r = await fetch("/api/v1/ai/analyze", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ prompt: promptText }),
    });
    const body = await r.json();
    if (!r.ok) throw new Error((body.error && body.error.message) || r.statusText);
    $("#ai-answer").textContent = body.answer || "(empty)";
    renderAIActionsIn($("#ai-actions"), body.actions || [], "Suggested actions");
    loadAIStatus();
  } catch (e) {
    $("#ai-answer").textContent = `error: ${e.message}`;
  }
}

function renderAIActionsIn(wrap, actions, title) {
  if (!wrap) return;
  clear(wrap);
  if (!actions || actions.length === 0) return;
  wrap.appendChild(el("h3", null, `${title} (${actions.length})`));
  wrap.appendChild(el("p", { class: "muted" }, "Nothing runs until you approve it. System processes cannot be killed — use Protect instead."));
  for (const sug of actions) {
    wrap.appendChild(renderAIActionCard(sug));
  }
}

function renderAIActionCard(sug) {
  const type = String(sug.type || "").toLowerCase();
  const card = el("div", { class: `ai-action-card type-${type}` });
  const header = el("div", { class: "ai-action-head" });
  header.appendChild(el("span", { class: `ai-action-type t-${type}` }, type));
  const targetText = sug.pid
    ? `${sug.name || "?"} (PID ${sug.pid})`
    : (sug.name || (sug.rule && sug.rule.name) || "—");
  header.appendChild(el("span", { class: "ai-action-target" }, targetText));
  card.appendChild(header);

  if (sug.reason) {
    card.appendChild(el("div", { class: "ai-action-reason" }, sug.reason));
  }

  if (type === "add_rule" && sug.rule) {
    const r = sug.rule;
    const forSec = r.for_seconds || r.for || 0;
    const desc = `match=${r.match || "?"} ${r.metric || "?"} ${r.op || ">="} ${r.threshold || 0} for ${forSec}s → ${r.action || "alert"}`;
    card.appendChild(el("div", { class: "ai-action-rule" }, desc));
  }

  if (sug.policy && sug.policy.status) {
    const p = sug.policy;
    let label = p.status;
    if (p.status === "needs_repeat") {
      label = `needs repeat ${p.repeat_count || 0}/${p.required_repeat_count || 0}`;
    } else if (p.status === "dry_run_eligible") {
      label = "dry-run eligible";
    }
    const reason = p.reason ? ` • ${p.reason}` : "";
    card.appendChild(el("div", { class: `ai-action-policy ${p.status}` }, `${label}${reason}`));
  }

  const pidNameLower = (sug.name || "").toLowerCase();
  const protectedByUI = state.protectedNames.has(pidNameLower);
  const targetsSelf = state.selfPID != null && Number(sug.pid || 0) === state.selfPID;
  const controls = el("div", { class: "ai-action-controls" });

  const approveBtn = el("button", { class: "ai-action-approve" }, "Approve");
  if ((type === "kill" || type === "suspend") && (protectedByUI || targetsSelf)) {
    approveBtn.disabled = true;
    approveBtn.title = targetsSelf
      ? "WTM cannot approve kill/suspend against its own running process"
      : "Target is on the protect list — remove it first";
  }
  approveBtn.addEventListener("click", async () => {
    approveBtn.disabled = true;
    approveBtn.textContent = "running…";
    try {
      const r = await fetch("/api/v1/ai/execute", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify(sug),
      });
      const body = await r.json();
      if (!r.ok) throw new Error((body.error && body.error.message) || r.statusText);
      approveBtn.textContent = "✓ done";
      card.classList.add("done");
      if (type === "protect" || type === "ignore" || type === "add_rule") {
        loadConfig();
      }
      loadAIWatch();
    } catch (e) {
      approveBtn.disabled = false;
      approveBtn.textContent = "Approve";
      alert(`error: ${e.message}`);
    }
  });
  controls.appendChild(approveBtn);

  const dismissBtn = el("button", { class: "ai-action-dismiss" }, "Dismiss");
  dismissBtn.addEventListener("click", () => card.remove());
  controls.appendChild(dismissBtn);
  card.appendChild(controls);

  return card;
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
  es.addEventListener("ai.background", () => { setConn(true); loadAIWatch(); loadAIStatus(); });
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
  $("#proc-sort").value = state.sort;
  $("#proc-sort").addEventListener("change", (e) => {
    state.sort = e.target.value;
    localStorage.setItem("wtm.sortCol", state.sort);
    if (state.snapshot) renderProcesses(state.snapshot);
  });
  $$(".proc-table th.sortable").forEach((th) => {
    th.addEventListener("click", () => setSort(th.dataset.sort));
  });
  $("#proc-body").addEventListener("click", (e) => {
    const btn = e.target.closest("button");
    if (!btn || btn.disabled) return;
    const act = btn.dataset.act;
    if (act === "protect") {
      toggleProtect(btn.dataset.name, btn.dataset.on !== "1");
      return;
    }
    if (act === "ignore") {
      toggleIgnore(btn.dataset.name, btn.dataset.on !== "1");
      return;
    }
    processAction(act, parseInt(btn.dataset.pid, 10));
  });
  $("#ai-send").addEventListener("click", sendAI);
  $("#ai-save").addEventListener("click", saveAIConfig);
  $("#tg-save").addEventListener("click", saveTelegramConfig);
  $("#ai-preset").addEventListener("change", (e) => applyPreset(e.target.value));

  const rateSel = $("#refresh-rate");
  rateSel.value = String(state.renderMinInterval);
  rateSel.addEventListener("change", (e) => {
    state.renderMinInterval = parseInt(e.target.value, 10) || 1000;
    localStorage.setItem("wtm.renderMs", String(state.renderMinInterval));
  });

  $("#rules-add").addEventListener("click", addRule);
  $("#rules-save").addEventListener("click", saveRules);
  $("#alerts-clear").addEventListener("click", clearAllAlerts);
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
  loadConfig();
  loadInfo();
  loadAlerts();
  loadRules();
  loadAIPresets().then(loadAIConfig);
  loadAIModels();
  loadTelegramConfig();
  loadAIStatus();
  loadAIWatch();
  setInterval(loadAlerts, 5000);
  setInterval(loadAIStatus, 10000);
  setInterval(loadAIWatch, 10000);
  setInterval(loadConfig, 15000);
  setupSSE();
}

bootstrap();
