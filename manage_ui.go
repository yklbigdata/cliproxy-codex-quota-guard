package main

const managementPageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Codex Quota Guard</title>
  <style>
    :root{color-scheme:light dark;--bg:#f6f7f9;--panel:#fff;--text:#17202a;--muted:#667085;--line:#d8dee7;--accent:#0b6bcb;--accent-2:#0756a4;--good:#137a43;--warn:#9a5b00;--bad:#b42318;--soft:#edf4fc;--shadow:0 10px 30px rgba(24,39,75,.08)}
    @media (prefers-color-scheme:dark){:root{--bg:#101419;--panel:#171d24;--text:#e7edf4;--muted:#9ba8b7;--line:#303b47;--accent:#60a5fa;--accent-2:#93c5fd;--good:#6ee7a1;--warn:#fbbf24;--bad:#fca5a5;--soft:#1b2a3a;--shadow:none}}
    *{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Noto Sans SC",sans-serif}
    main{max-width:1280px;margin:0 auto;padding:28px 20px 64px}header{display:flex;gap:16px;justify-content:space-between;align-items:flex-start;margin-bottom:22px}h1{font-size:1.55rem;margin:0 0 4px}h2{font-size:1.02rem;margin:0 0 14px}h3{font-size:.92rem;margin:0 0 10px}p{margin:.35rem 0}.muted{color:var(--muted)}.small{font-size:.82rem}.mono,code{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.card{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:18px;box-shadow:var(--shadow);margin-bottom:16px}.grid{display:grid;gap:14px}.grid-2{grid-template-columns:repeat(2,minmax(0,1fr))}.grid-4{grid-template-columns:repeat(4,minmax(0,1fr))}.field{display:grid;gap:6px}.field label,.label{font-weight:600;font-size:.86rem}.hint{color:var(--muted);font-size:.78rem}input,select,textarea,button{font:inherit}input[type=text],input[type=password],input[type=number],input[type=time],select,textarea{width:100%;border:1px solid var(--line);border-radius:9px;background:var(--panel);color:var(--text);padding:9px 10px}textarea{min-height:78px;resize:vertical}input:focus,textarea:focus,select:focus,button:focus-visible{outline:2px solid color-mix(in srgb,var(--accent) 55%,transparent);outline-offset:1px}button{border:1px solid var(--line);border-radius:9px;background:var(--panel);color:var(--text);padding:8px 12px;cursor:pointer;font-weight:600}button:hover{border-color:var(--accent)}button.primary{background:var(--accent);border-color:var(--accent);color:#fff}button.primary:hover{background:var(--accent-2)}button.danger{color:var(--bad);border-color:color-mix(in srgb,var(--bad) 40%,var(--line))}button:disabled{opacity:.48;cursor:not-allowed}.row{display:flex;gap:10px;align-items:center;flex-wrap:wrap}.between{justify-content:space-between}.check{display:flex;gap:8px;align-items:center;font-weight:600}.check input{width:16px;height:16px;accent-color:var(--accent)}.notice{min-height:22px;border-radius:9px;padding:9px 12px;background:var(--soft);display:none}.notice.show{display:block}.notice.error{color:var(--bad);background:color-mix(in srgb,var(--bad) 10%,var(--panel))}.notice.success{color:var(--good);background:color-mix(in srgb,var(--good) 10%,var(--panel))}.status-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.stat{border:1px solid var(--line);border-radius:11px;padding:12px;min-height:74px}.stat .v{font-size:1.05rem;font-weight:700;margin-top:3px;overflow-wrap:anywhere}.pill{display:inline-flex;align-items:center;border:1px solid var(--line);border-radius:999px;padding:1px 7px;font-size:.74rem;font-weight:600;white-space:nowrap}.pill.ok{color:var(--good);border-color:color-mix(in srgb,var(--good) 45%,var(--line))}.pill.warn{color:var(--warn);border-color:color-mix(in srgb,var(--warn) 45%,var(--line))}.pill.bad{color:var(--bad);border-color:color-mix(in srgb,var(--bad) 45%,var(--line))}.table-wrap{overflow:auto;border:1px solid var(--line);border-radius:11px}table{width:100%;border-collapse:collapse;font-size:.82rem}th,td{text-align:left;padding:9px 10px;border-bottom:1px solid var(--line);vertical-align:top}th{background:color-mix(in srgb,var(--panel) 80%,var(--bg));position:sticky;top:0;z-index:1;font-size:.76rem;color:var(--muted)}tr:last-child td{border-bottom:0}.cell-main{font-weight:600}.cell-sub{color:var(--muted);font-size:.75rem;overflow-wrap:anywhere}.schedule-row{display:grid;grid-template-columns:minmax(150px,1fr) 140px auto auto;gap:9px;align-items:end;border-top:1px solid var(--line);padding-top:11px;margin-top:11px}.schedule-row:first-of-type{border-top:0;padding-top:0;margin-top:0}.section-head{display:flex;justify-content:space-between;gap:12px;align-items:center;margin-bottom:14px}.hidden{display:none!important}.footer{color:var(--muted);font-size:.78rem;text-align:center;margin-top:24px}@media(max-width:900px){.grid-4,.status-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.grid-2{grid-template-columns:1fr}}@media(max-width:620px){main{padding:18px 12px 44px}header{display:block}.status-grid,.grid-4{grid-template-columns:1fr}.schedule-row{grid-template-columns:1fr 1fr}.card{padding:14px;border-radius:11px}}
  </style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>Codex Quota Guard</h1>
      <p class="muted">Use Codex-reported quota to manage credentials, 429 blocks, scheduler filtering, and warm-up.</p>
    </div>
    <div class="pill" id="version">not connected</div>
  </header>

  <section class="card">
    <h2>Management connection</h2>
    <div class="grid grid-2">
      <div class="field">
        <label for="management-key">CPA Management Key</label>
        <input id="management-key" type="password" autocomplete="current-password" placeholder="Enter the CPA management credential">
        <div class="hint">This is the global CLIProxyAPI management credential. It is not an OpenAI key, Codex credential, or plugin key.</div>
      </div>
      <div class="field">
        <span class="label">Storage</span>
        <label class="check"><input id="remember-key" type="checkbox"> Remember in this browser</label>
        <div class="hint">By default the key is kept only in this tab session. Browser persistence is opt-in. The plugin never stores this key.</div>
      </div>
    </div>
    <div class="row" style="margin-top:12px">
      <button class="primary" id="connect">Connect and load</button>
      <button id="clear-key">Clear key</button>
      <span class="muted small">Authentication is handled by the CPA management middleware.</span>
    </div>
    <div id="notice" class="notice" role="status" aria-live="polite" style="margin-top:12px"></div>
  </section>

  <section class="card hidden" id="status-card">
    <div class="section-head">
      <div><h2>Live status</h2><p class="muted small">The page reads live state after each connect. It does not embed credential data in the HTML resource.</p></div>
      <div class="row"><button id="refresh">Refresh</button><button id="run-sweep">Run quota sweep now</button><button id="run-warmup">Run warm-up now</button></div>
    </div>
    <div class="status-grid" id="status-grid"></div>
  </section>

  <form class="card hidden" id="settings-card">
    <div class="section-head"><div><h2>Quota policy</h2><p class="muted small">These values are saved with a PATCH to the native CPA plugin configuration API. Unknown fields are preserved.</p></div><div class="row"><button type="button" id="reload-settings">Reload</button><button class="primary" type="submit" id="save-settings">Validate and save</button></div></div>
    <div class="grid grid-4">
      <div class="field"><label for="disable-5h">Disable 5h at remaining (%)</label><input id="disable-5h" type="number" min="0" max="100" step="0.1" required></div>
      <div class="field"><label for="resume-5h">Resume 5h at remaining (%)</label><input id="resume-5h" type="number" min="0" max="100" step="0.1" required></div>
      <div class="field"><label for="poll-interval">Sweep interval</label><input id="poll-interval" type="text" placeholder="60s" required><div class="hint">Duration format, minimum 15s.</div></div>
      <div class="field"><label for="initial-delay">Initial delay</label><input id="initial-delay" type="text" placeholder="15s" required><div class="hint">Delay before the startup sweep.</div></div>
    </div>
    <div class="grid grid-2" style="margin-top:14px">
      <div class="field">
        <label class="check"><input id="weekly-enabled" type="checkbox"> Enforce the 7-day window</label>
        <div class="grid grid-2" id="weekly-fields" style="margin-top:10px">
          <div class="field"><label for="disable-7d">Disable 7d at remaining (%)</label><input id="disable-7d" type="number" min="0" max="100" step="0.1" required></div>
          <div class="field"><label for="resume-7d">Resume 7d at remaining (%)</label><input id="resume-7d" type="number" min="0" max="100" step="0.1" required></div>
        </div>
      </div>
      <div class="grid grid-2">
        <label class="check"><input id="keep-one" type="checkbox"> Keep at least one credential</label>
        <label class="check"><input id="dry-run" type="checkbox"> Dry run</label>
        <div class="field" style="grid-column:1/-1"><label for="rate-limit-fallback">429 fallback duration</label><input id="rate-limit-fallback" type="text" placeholder="5h" required><div class="hint">Used when a Codex 429 response has no reset header.</div></div>
      </div>
    </div>
  </form>

  <form class="card hidden" id="warmup-card">
<div class="section-head"><div><h2>Scheduled warm-up</h2><p class="muted small">Warm-up runs only for eligible Codex credentials. It skips manually disabled and actively blocked credentials.</p></div><div class="row"><button type="button" id="add-schedule">Add schedule</button><button class="primary" type="submit" id="save-warmup">Validate and save</button></div></div>
    <div class="grid grid-2">
      <div class="grid">
        <label class="check"><input id="warmup-enabled" type="checkbox"> Enable scheduled warm-up</label>
        <div class="field"><label for="warmup-model">Warm-up model</label><input id="warmup-model" type="text" required></div>
        <div class="field"><label for="warmup-timezone">IANA timezone</label><input id="warmup-timezone" type="text" placeholder="Asia/Singapore" required></div>
      </div>
      <div class="field"><label for="warmup-prompt">Warm-up prompt</label><textarea id="warmup-prompt" required></textarea></div>
    </div>
    <h3 style="margin-top:18px">Daily schedules</h3>
    <div id="schedule-list"><p class="muted small" id="schedule-empty">No schedules configured.</p></div>
  </form>

  <section class="card hidden" id="credentials-card">
    <div class="section-head"><div><h2>Credentials and last decisions</h2><p class="muted small">Recent sweep decisions and warm-up results. Block reset is available only for plugin-owned state.</p></div><button id="refresh-credentials">Refresh</button></div>
    <div class="table-wrap"><table><thead><tr><th>Credential</th><th>Plan</th><th>5h used</th><th>7d used</th><th>State</th><th>Action and reason</th><th>Warm-up</th><th>Checked</th><th>Operation</th></tr></thead><tbody id="credentials-body"></tbody></table></div>
    <p class="muted small" id="credentials-empty" style="margin-top:10px">No sweep data loaded.</p>
  </section>

  <p class="footer">Codex Quota Guard 2.1.0. Settings use the CPA Management API. Credential files and tokens are never returned to this page.</p>
</main>
<script>
` + managementPageScriptCore + managementPageScriptUI + managementPageScriptEvents + `
</script>
</body>
</html>`

func renderManagementPage() string {
	return managementPageHTML
}

const managementPageScriptCore = `
(() => {
  "use strict";
  const API = "/v0/management/plugins/codex-quota-guard";
  const CONFIG_API = "/v0/management/plugins/codex-quota-guard/config";
  const SESSION_KEY = "codexQuotaGuardManagementKey";
  const LOCAL_KEY = "codexQuotaGuardManagementKeyPersistent";
  const els = {};
  let managementKey = "";
  let currentStatus = null;

  function byId(id) { return document.getElementById(id); }
  function showNotice(text, kind) {
    const node = els.notice;
    node.textContent = text || "";
    node.className = "notice" + (text ? " show" : "") + (kind ? " " + kind : "");
  }
  function setConnectedUi(connected) {
    ["status-card", "settings-card", "warmup-card", "credentials-card"].forEach(id => {
      byId(id).classList.toggle("hidden", !connected);
    });
  }
  function readStoredKey() {
    return localStorage.getItem(LOCAL_KEY) || sessionStorage.getItem(SESSION_KEY) || "";
  }
  function storeKey() {
    sessionStorage.setItem(SESSION_KEY, managementKey);
    if (els.remember.checked) localStorage.setItem(LOCAL_KEY, managementKey);
    else localStorage.removeItem(LOCAL_KEY);
  }
  function clearKey() {
    managementKey = "";
    els.key.value = "";
    sessionStorage.removeItem(SESSION_KEY);
    localStorage.removeItem(LOCAL_KEY);
    setConnectedUi(false);
    showNotice("Management key cleared.", "success");
  }
  async function request(path, options) {
    if (!managementKey) throw new Error("Enter and connect a CPA Management Key first.");
    const headers = Object.assign({
      "Accept": "application/json",
      "Authorization": "Bearer " + managementKey
    }, (options && options.headers) || {});
    if (options && options.body !== undefined) headers["Content-Type"] = "application/json";
    const response = await fetch(path, Object.assign({credentials: "same-origin"}, options || {}, {headers}));
    const text = await response.text();
    let data = null;
    if (text) {
      try { data = JSON.parse(text); } catch (_) { data = {raw: text}; }
    }
    if (!response.ok) {
      const message = data && (data.message || data.error || data.raw);
      const error = new Error(message || ("HTTP " + response.status));
      error.status = response.status;
      throw error;
    }
    return data || {};
  }
  function formatTime(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return date.toLocaleString();
  }
  function formatSeconds(value) {
    const seconds = Number(value || 0);
    if (!seconds) return "—";
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days) return days + "d " + hours + "h";
    if (hours) return hours + "h " + minutes + "m";
    return Math.max(0, minutes) + "m";
  }
  function pct(value) {
    return value === null || value === undefined ? "—" : Number(value).toFixed(Number(value) % 1 ? 1 : 0) + "%";
  }
  function badge(text, kind) {
    const node = document.createElement("span");
    node.className = "pill" + (kind ? " " + kind : "");
    node.textContent = text;
    return node;
  }
  function line(main, sub, mono) {
    const wrap = document.createElement("div");
    const primary = document.createElement("div");
    primary.className = mono ? "cell-main mono" : "cell-main";
    primary.textContent = main || "—";
    wrap.append(primary);
    if (sub) {
      const secondary = document.createElement("div");
      secondary.className = "cell-sub";
      secondary.textContent = sub;
      wrap.append(secondary);
    }
    return wrap;
  }
  function stable(value) { return JSON.stringify(value); }
`

const managementPageScriptUI = `
  let baselineConfig = {};
  let baselineRawSettings = "";
  let baselineSettings = "";
  let statusTimer = null;
  let statusBusy = false;
  let scheduleSequence = 0;
  const FUNCTIONAL_FIELDS = [
    "disable_below_remaining_percent",
    "resume_above_remaining_percent",
    "enforce_weekly_window",
    "weekly_disable_below_remaining_percent",
    "weekly_resume_above_remaining_percent",
    "poll_interval",
    "initial_delay",
    "keep_at_least_one",
    "dry_run",
    "rate_limit_fallback",
    "warmup_enabled",
    "warmup_model",
    "warmup_prompt",
    "warmup_timezone",
    "warmup_schedules"
  ];

  function hasOwn(object, key) {
    return object !== null && object !== undefined && Object.prototype.hasOwnProperty.call(object, key);
  }
  function pickValue(source, key, fallback) {
    if (!hasOwn(source, key) || source[key] === null) return fallback;
    return source[key];
  }
  function cloneJson(value, fallback) {
    try { return JSON.parse(JSON.stringify(value)); } catch (_) { return fallback; }
  }
  function normalizeSchedules(items) {
    if (!Array.isArray(items)) return [];
    return items.map(item => {
      const source = item && typeof item === "object" ? item : {};
      return {
        id: String(source.id || "").trim(),
        at: String(source.at || "").trim(),
        enabled: source.enabled === undefined ? true : Boolean(source.enabled)
      };
    });
  }
  function normalizeSettings(settings) {
    const source = settings && typeof settings === "object" ? settings : {};
    return {
      disable_below_remaining_percent: Number(source.disable_below_remaining_percent),
      resume_above_remaining_percent: Number(source.resume_above_remaining_percent),
      enforce_weekly_window: Boolean(source.enforce_weekly_window),
      weekly_disable_below_remaining_percent: Number(source.weekly_disable_below_remaining_percent),
      weekly_resume_above_remaining_percent: Number(source.weekly_resume_above_remaining_percent),
      poll_interval: String(source.poll_interval || ""),
      initial_delay: String(source.initial_delay || ""),
      keep_at_least_one: Boolean(source.keep_at_least_one),
      dry_run: Boolean(source.dry_run),
      rate_limit_fallback: String(source.rate_limit_fallback || ""),
      warmup_enabled: Boolean(source.warmup_enabled),
      warmup_model: String(source.warmup_model || ""),
      warmup_prompt: String(source.warmup_prompt || ""),
      warmup_timezone: String(source.warmup_timezone || ""),
      warmup_schedules: normalizeSchedules(source.warmup_schedules)
    };
  }
  function settingsFromConfig(raw, fallback) {
    const config = raw && typeof raw === "object" ? raw : {};
    const effective = fallback && typeof fallback === "object" ? fallback : {};
    const schedules = hasOwn(config, "warmup_schedules") ? config.warmup_schedules : effective.warmup_schedules;
    return normalizeSettings({
      disable_below_remaining_percent: pickValue(config, "disable_below_remaining_percent", effective.disable_below_remaining_percent),
      resume_above_remaining_percent: pickValue(config, "resume_above_remaining_percent", effective.resume_above_remaining_percent),
      enforce_weekly_window: pickValue(config, "enforce_weekly_window", effective.enforce_weekly_window),
      weekly_disable_below_remaining_percent: pickValue(config, "weekly_disable_below_remaining_percent", effective.weekly_disable_below_remaining_percent),
      weekly_resume_above_remaining_percent: pickValue(config, "weekly_resume_above_remaining_percent", effective.weekly_resume_above_remaining_percent),
      poll_interval: pickValue(config, "poll_interval", effective.poll_interval),
      initial_delay: pickValue(config, "initial_delay", effective.initial_delay),
      keep_at_least_one: pickValue(config, "keep_at_least_one", effective.keep_at_least_one),
      dry_run: pickValue(config, "dry_run", effective.dry_run),
      rate_limit_fallback: pickValue(config, "rate_limit_fallback", effective.rate_limit_fallback),
      warmup_enabled: pickValue(config, "warmup_enabled", effective.warmup_enabled),
      warmup_model: pickValue(config, "warmup_model", effective.warmup_model),
      warmup_prompt: pickValue(config, "warmup_prompt", effective.warmup_prompt),
      warmup_timezone: pickValue(config, "warmup_timezone", effective.warmup_timezone),
      warmup_schedules: schedules
    });
  }
  function rawFunctionalProjection(raw) {
    const config = raw && typeof raw === "object" ? raw : {};
    const out = {};
    FUNCTIONAL_FIELDS.forEach(key => {
      if (!hasOwn(config, key)) return;
      if (key === "warmup_schedules") out[key] = normalizeSchedules(config[key]);
      else out[key] = config[key];
    });
    return out;
  }
  function functionalPatch() {
    return {
      disable_below_remaining_percent: Number(els.disable5h.value),
      resume_above_remaining_percent: Number(els.resume5h.value),
      enforce_weekly_window: els.weeklyEnabled.checked,
      weekly_disable_below_remaining_percent: Number(els.disable7d.value),
      weekly_resume_above_remaining_percent: Number(els.resume7d.value),
      poll_interval: els.pollInterval.value.trim(),
      initial_delay: els.initialDelay.value.trim(),
      keep_at_least_one: els.keepOne.checked,
      dry_run: els.dryRun.checked,
      rate_limit_fallback: els.rateLimitFallback.value.trim(),
      warmup_enabled: els.warmupEnabled.checked,
      warmup_model: els.warmupModel.value.trim(),
      warmup_prompt: els.warmupPrompt.value.trim(),
      warmup_timezone: els.warmupTimezone.value.trim(),
      warmup_schedules: collectSchedules()
    };
  }
  function collectSchedules() {
    const schedules = [];
    els.scheduleList.querySelectorAll("[data-schedule-row]").forEach(row => {
      const id = row.querySelector('[data-role="id"]');
      const at = row.querySelector('[data-role="at"]');
      const enabled = row.querySelector('[data-role="enabled"]');
      schedules.push({
        id: id ? id.value.trim() : "",
        at: at ? at.value.trim() : "",
        enabled: enabled ? enabled.checked : true
      });
    });
    return schedules;
  }
  function applySettings(settings) {
    const value = normalizeSettings(settings);
    els.disable5h.value = String(value.disable_below_remaining_percent);
    els.resume5h.value = String(value.resume_above_remaining_percent);
    els.weeklyEnabled.checked = value.enforce_weekly_window;
    els.disable7d.value = String(value.weekly_disable_below_remaining_percent);
    els.resume7d.value = String(value.weekly_resume_above_remaining_percent);
    els.pollInterval.value = value.poll_interval;
    els.initialDelay.value = value.initial_delay;
    els.keepOne.checked = value.keep_at_least_one;
    els.dryRun.checked = value.dry_run;
    els.rateLimitFallback.value = value.rate_limit_fallback;
    els.warmupEnabled.checked = value.warmup_enabled;
    els.warmupModel.value = value.warmup_model;
    els.warmupPrompt.value = value.warmup_prompt;
    els.warmupTimezone.value = value.warmup_timezone;
    renderSchedules(value.warmup_schedules);
    syncWeeklyControls();
    syncWarmupControls();
  }
  function syncWeeklyControls() {
    const enabled = els.weeklyEnabled.checked;
    els.disable7d.disabled = !enabled;
    els.resume7d.disabled = !enabled;
    els.weeklyFields.classList.toggle("muted", !enabled);
  }
  function syncWarmupControls() {
    const enabled = els.warmupEnabled.checked;
    els.warmupModel.disabled = !enabled;
    els.warmupTimezone.disabled = !enabled;
    els.warmupPrompt.disabled = !enabled;
    els.addSchedule.disabled = !enabled;
    els.scheduleList.querySelectorAll("input,button").forEach(node => { node.disabled = !enabled; });
  }
  function renderSchedules(schedules) {
    els.scheduleList.replaceChildren();
    const items = normalizeSchedules(schedules);
    if (!items.length) {
      const empty = document.createElement("p");
      empty.className = "muted small";
      empty.textContent = "No schedules configured.";
      els.scheduleList.append(empty);
    } else {
      items.forEach(item => els.scheduleList.append(createScheduleRow(item)));
    }
    syncWarmupControls();
  }
  function createScheduleRow(schedule) {
    const row = document.createElement("div");
    row.className = "schedule-row";
    row.dataset.scheduleRow = "true";

    const idField = document.createElement("label");
    idField.className = "field";
    idField.append(document.createTextNode("Schedule ID"));
    const id = document.createElement("input");
    id.type = "text";
    id.required = true;
    id.maxLength = 80;
    id.placeholder = "daily";
    id.value = schedule.id || "";
    id.dataset.role = "id";
    idField.append(id);

    const timeField = document.createElement("label");
    timeField.className = "field";
    timeField.append(document.createTextNode("Local time"));
    const at = document.createElement("input");
    at.type = "time";
    at.required = true;
    at.value = schedule.at || "09:00";
    at.dataset.role = "at";
    timeField.append(at);

    const enabledLabel = document.createElement("label");
    enabledLabel.className = "check";
    const enabled = document.createElement("input");
    enabled.type = "checkbox";
    enabled.checked = schedule.enabled !== false;
    enabled.dataset.role = "enabled";
    enabledLabel.append(enabled, document.createTextNode("Enabled"));

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "danger";
    remove.textContent = "Remove";
    remove.addEventListener("click", () => { row.remove(); ensureScheduleEmpty(); });

    row.append(idField, timeField, enabledLabel, remove);
    return row;
  }
  function ensureScheduleEmpty() {
    if (!els.scheduleList.querySelector("[data-schedule-row]")) {
      const empty = document.createElement("p");
      empty.className = "muted small";
      empty.textContent = "No schedules configured.";
      els.scheduleList.append(empty);
    }
  }
  function addSchedule() {
    els.scheduleList.querySelectorAll("p").forEach(node => node.remove());
    const usedTimes = new Set(collectSchedules().map(item => item.at));
    let at = "09:00";
    for (let hour = 0; hour < 24; hour += 1) {
      const candidate = String(hour).padStart(2, "0") + ":00";
      if (!usedTimes.has(candidate)) { at = candidate; break; }
    }
    const usedIds = new Set(collectSchedules().map(item => item.id.toLowerCase()));
    let id = "schedule-" + Date.now();
    while (usedIds.has(id.toLowerCase())) {
      scheduleSequence += 1;
      id = "schedule-" + Date.now() + "-" + scheduleSequence;
    }
    els.scheduleList.append(createScheduleRow({id, at, enabled: true}));
    syncWarmupControls();
  }
  function statCard(label, value, sub, kind) {
    const card = document.createElement("div");
    card.className = "stat";
    const name = document.createElement("div");
    name.className = "muted small";
    name.textContent = label;
    const primary = document.createElement("div");
    primary.className = "v";
    primary.textContent = value || "—";
    card.append(name, primary);
    if (sub) {
      const note = document.createElement("div");
      note.className = "cell-sub";
      note.textContent = sub;
      if (kind) note.append(" ");
      if (kind) note.append(badge(kind === "bad" ? "attention" : kind, kind));
      card.append(note);
    }
    return card;
  }
  function renderStatus(status) {
    currentStatus = status;
    setConnectedUi(true);
    els.version.textContent = "v" + (status.plugin_version || "unknown");
    els.statusGrid.replaceChildren();
    const worker = status.running ? "running" : "stopped";
    els.statusGrid.append(statCard("Worker", worker, status.started_at ? "Started " + formatTime(status.started_at) : "", status.running ? "ok" : "bad"));
    els.statusGrid.append(statCard("Quota sweep", status.sweep_running ? "running" : "idle",
      (status.last_sweep_cause || "no sweep") + " · " + Number(status.sweeps || 0) + " total", status.sweep_running ? "warn" : "ok"));
    els.statusGrid.append(statCard("Last sweep", formatTime(status.last_sweep),
      "Next " + formatTime(status.next_sweep), ""));
    els.statusGrid.append(statCard("Warm-up", status.warmup_running ? "running" : "idle",
      status.next_warmup ? "Next " + formatTime(status.next_warmup) + (status.next_warmup_id ? " · " + status.next_warmup_id : "") : "No enabled schedule", status.warmup_running ? "warn" : ""));
    els.statusGrid.append(statCard("Actions", Number(status.disables || 0) + " disabled",
      Number(status.enables || 0) + " enabled · " + Number(status.errors || 0) + " errors", Number(status.errors || 0) ? "warn" : ""));
    els.statusGrid.append(statCard("State", status.state_read_only ? "read-only" : "writable",
      status.state_saved_at ? "Saved " + formatTime(status.state_saved_at) : "Not saved yet", status.state_read_only ? "bad" : "ok"));
    els.statusGrid.append(statCard("Warm-up results", String(Number(status.warmup_success || 0)) + " / " + String(Number(status.warmup_results || 0)),
      Number(status.warmup_skipped || 0) + " skipped", status.warmup_error ? "warn" : ""));
    const warningText = [status.state_error, status.last_sweep_error, status.warmup_error].filter(Boolean).join(" · ");
    els.statusGrid.append(statCard("Health", warningText ? "needs attention" : "no state errors",
      warningText || "Persistent state and workers report no current error", warningText ? "bad" : "ok"));

    els.runSweep.disabled = !status.running || Boolean(status.sweep_running);
    els.runWarmup.disabled = !status.running || Boolean(status.warmup_running) || !(status.settings && status.settings.warmup_enabled);
    renderCredentials(status.credentials || []);
    if (status.sweep_running || status.warmup_running) scheduleStatusPoll(1200);
  }
  function actionKind(action) {
    if (action === "disabled") return "warn";
    if (action === "enabled") return "ok";
    if (action === "error") return "bad";
    return "";
  }
  function credentialState(report) {
    const wrap = document.createElement("div");
    wrap.className = "row";
    if (report.manual_disabled) wrap.append(badge("manual disabled", "bad"));
    if (report.quota_blocked) wrap.append(badge("5h quota", "warn"));
    if (report.rate_limit_blocked) wrap.append(badge("429 active", "warn"));
    if (report.managed) wrap.append(badge("plugin disabled", "warn"));
    if (report.file_disabled && !report.managed && !report.manual_disabled) wrap.append(badge("file disabled", "bad"));
    if (report.runtime_disabled && !report.file_disabled) wrap.append(badge("runtime disabled", "bad"));
    if (!wrap.childNodes.length) wrap.append(badge("within policy", "ok"));
    return wrap;
  }
  function resetCaption(seconds) {
    if (!seconds) return "";
    return "reset in " + formatSeconds(seconds);
  }
  function renderCredentials(credentials) {
    els.credentialsBody.replaceChildren();
    const items = Array.isArray(credentials) ? credentials : [];
    els.credentialsEmpty.classList.toggle("hidden", items.length !== 0);
    items.forEach(report => {
      const row = document.createElement("tr");
      const credential = document.createElement("td");
      credential.append(line(report.name || report.ref, report.email || report.provider || report.ref, true));
      const plan = document.createElement("td");
      plan.textContent = report.plan_type || "—";
      const primary = document.createElement("td");
      primary.append(line(pct(report.primary_used_percent), resetCaption(report.primary_reset_in_seconds), false));
      const weekly = document.createElement("td");
      weekly.append(line(pct(report.weekly_used_percent), resetCaption(report.weekly_reset_in_seconds), false));
      const state = document.createElement("td");
      state.append(credentialState(report));
      const action = document.createElement("td");
      const actionWrap = document.createElement("div");
      actionWrap.className = "row";
      actionWrap.append(badge(report.action || "none", actionKind(report.action)));
      action.append(actionWrap, line("", [report.reason, report.error].filter(Boolean).join(": "), false));
      const warmup = document.createElement("td");
      const warmupText = report.warmup_last_run_at ? (report.warmup_last_success ? "success" : "failed") : "—";
      warmup.append(line(warmupText, report.warmup_last_error || (report.warmup_last_status ? "HTTP " + report.warmup_last_status : ""), false));
      const checked = document.createElement("td");
      checked.className = "muted";
      checked.textContent = formatTime(report.checked_at);
      const operation = document.createElement("td");
      if (report.can_reset_block) {
        const reset = document.createElement("button");
        reset.type = "button";
        reset.className = "danger";
        reset.textContent = "Reset block";
        reset.addEventListener("click", () => resetBlock(report));
        operation.append(reset);
      } else {
        operation.textContent = "—";
      }
      row.append(credential, plan, primary, weekly, state, action, warmup, checked, operation);
      els.credentialsBody.append(row);
    });
  }
  function setSaveBusy(busy) {
    els.saveSettings.disabled = busy;
    els.saveWarmup.disabled = busy;
    els.reloadSettings.disabled = busy;
    els.connect.disabled = busy;
  }
  function scheduleStatusPoll(delay) {
    if (statusTimer) clearTimeout(statusTimer);
    statusTimer = setTimeout(async () => {
      statusTimer = null;
      await refreshStatus(true);
      if (currentStatus && (currentStatus.sweep_running || currentStatus.warmup_running)) scheduleStatusPoll(1200);
    }, delay);
  }
`

const managementPageScriptEvents = `
  function initElements() {
    Object.assign(els, {
      version: byId("version"),
      key: byId("management-key"),
      remember: byId("remember-key"),
      connect: byId("connect"),
      clearKey: byId("clear-key"),
      notice: byId("notice"),
      statusGrid: byId("status-grid"),
      refresh: byId("refresh"),
      runSweep: byId("run-sweep"),
      runWarmup: byId("run-warmup"),
      settingsCard: byId("settings-card"),
      reloadSettings: byId("reload-settings"),
      saveSettings: byId("save-settings"),
      disable5h: byId("disable-5h"),
      resume5h: byId("resume-5h"),
      pollInterval: byId("poll-interval"),
      initialDelay: byId("initial-delay"),
      weeklyEnabled: byId("weekly-enabled"),
      weeklyFields: byId("weekly-fields"),
      disable7d: byId("disable-7d"),
      resume7d: byId("resume-7d"),
      keepOne: byId("keep-one"),
      dryRun: byId("dry-run"),
      rateLimitFallback: byId("rate-limit-fallback"),
      warmupCard: byId("warmup-card"),
      saveWarmup: byId("save-warmup"),
      warmupEnabled: byId("warmup-enabled"),
      warmupModel: byId("warmup-model"),
      warmupTimezone: byId("warmup-timezone"),
      warmupPrompt: byId("warmup-prompt"),
      addSchedule: byId("add-schedule"),
      scheduleList: byId("schedule-list"),
      credentialsBody: byId("credentials-body"),
      credentialsEmpty: byId("credentials-empty"),
      refreshCredentials: byId("refresh-credentials")
    });
  }
  function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
  function conflict(message) {
    const error = new Error(message);
    error.status = 409;
    return error;
  }
  function handleRequestError(error, context) {
    const message = error && error.message ? error.message : String(error || "unknown error");
    if (error && error.status === 401) {
      managementKey = "";
      sessionStorage.removeItem(SESSION_KEY);
      localStorage.removeItem(LOCAL_KEY);
      setConnectedUi(false);
      showNotice("Authentication failed. Enter the current CPA Management Key again.", "error");
      return;
    }
    if (error && error.status === 409) {
      showNotice((context ? context + ": " : "") + "The live configuration changed. Reload settings, review the values, and save again. " + message, "error");
      return;
    }
    showNotice((context ? context + ": " : "") + message, "error");
  }
  function setBaselines(rawConfig, status) {
    baselineConfig = cloneJson(rawConfig && typeof rawConfig === "object" ? rawConfig : {}, {});
    baselineRawSettings = stable(rawFunctionalProjection(baselineConfig));
    const effective = settingsFromConfig(baselineConfig, status && status.settings ? status.settings : {});
    applySettings(effective);
    baselineSettings = stable(normalizeSettings(effective));
  }
  async function loadSettings(status) {
    const rawConfig = await request(CONFIG_API);
    setBaselines(rawConfig, status || currentStatus || {});
  }
  async function connect() {
    const candidate = els.key.value;
    if (!candidate || !candidate.trim()) {
      showNotice("Enter the CPA Management Key first.", "error");
      els.key.focus();
      return;
    }
    managementKey = candidate;
    els.connect.disabled = true;
    showNotice("Connecting to the CPA Management API...", "");
    try {
      const results = await Promise.all([
        request(API + "/status"),
        request(CONFIG_API)
      ]);
      storeKey();
      renderStatus(results[0]);
      setBaselines(results[1], results[0]);
      els.key.value = managementKey;
      showNotice("Connected. Live status and effective settings are loaded.", "success");
    } catch (error) {
      if (error && error.status === 401) managementKey = "";
      handleRequestError(error, "Connect failed");
    } finally {
      els.connect.disabled = false;
    }
  }
  async function refreshStatus(silent) {
    if (!managementKey || statusBusy) return;
    statusBusy = true;
    try {
      const status = await request(API + "/status");
      renderStatus(status);
      if (!silent) showNotice("Live status refreshed.", "success");
    } catch (error) {
      if (!silent) handleRequestError(error, "Refresh failed");
    } finally {
      statusBusy = false;
    }
  }
  async function reloadAll() {
    if (!managementKey) {
      showNotice("Connect first.", "error");
      return;
    }
    setSaveBusy(true);
    try {
      const results = await Promise.all([
        request(API + "/status"),
        request(CONFIG_API)
      ]);
      renderStatus(results[0]);
      setBaselines(results[1], results[0]);
      showNotice("Saved configuration and live status reloaded.", "success");
    } catch (error) {
      handleRequestError(error, "Reload failed");
    } finally {
      setSaveBusy(false);
    }
  }
  async function waitForSettings(expected, timeoutMs) {
    const target = stable(normalizeSettings(expected));
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
      try {
        const status = await request(API + "/status");
        renderStatus(status);
        if (stable(normalizeSettings(status.settings)) === target) return true;
      } catch (error) {
        if (error && error.status === 401) throw error;
      }
      await sleep(500);
    }
    return false;
  }
  async function saveSettings(event) {
    event.preventDefault();
    if (!managementKey) {
      showNotice("Connect first.", "error");
      return;
    }
    if (!baselineSettings) {
      showNotice("Reload settings before saving.", "error");
      return;
    }
    setSaveBusy(true);
    try {
      const results = await Promise.all([
        request(API + "/status"),
        request(CONFIG_API)
      ]);
      const latestStatus = results[0];
      const latestConfig = results[1];
      if (stable(normalizeSettings(latestStatus.settings)) !== baselineSettings ||
          stable(rawFunctionalProjection(latestConfig)) !== baselineRawSettings) {
        throw conflict("Functional settings changed after this page loaded them.");
      }

      const patch = functionalPatch();
      const expected = normalizeSettings(Object.assign({}, latestStatus.settings, patch));
      await request(API + "/validate-config", {
        method: "POST",
        body: JSON.stringify(patch)
      });
      await request(CONFIG_API, {
        method: "PATCH",
        body: JSON.stringify(patch)
      });

      baselineConfig = Object.assign({}, latestConfig, patch);
      baselineRawSettings = stable(rawFunctionalProjection(baselineConfig));
      baselineSettings = stable(expected);
      showNotice("Configuration saved. Waiting for CPA to apply the plugin reconfiguration...", "");
      const applied = await waitForSettings(expected, 15000);
      const finalStatus = await request(API + "/status");
      const finalConfig = await request(CONFIG_API);
      renderStatus(finalStatus);
      if (applied) {
        setBaselines(finalConfig, finalStatus);
        showNotice("Configuration saved and applied by the running plugin.", "success");
      } else {
        applySettings(expected);
        baselineConfig = finalConfig;
        baselineRawSettings = stable(rawFunctionalProjection(finalConfig));
        baselineSettings = stable(expected);
        showNotice("Configuration is saved, but the running plugin has not reported the new values yet. Refresh status before another edit.", "error");
      }
    } catch (error) {
      handleRequestError(error, "Save failed");
    } finally {
      setSaveBusy(false);
    }
  }
  async function runAction(path, label) {
    if (!managementKey) {
      showNotice("Connect first.", "error");
      return;
    }
    try {
      await request(path, {method: "POST"});
      showNotice(label + " started.", "success");
      await refreshStatus(true);
      scheduleStatusPoll(500);
    } catch (error) {
      handleRequestError(error, label + " failed");
    }
  }
  async function resetBlock(report) {
    const name = report.name || report.ref || "this credential";
    const reference = report.ref || report.name || "";
    if (!reference) {
      showNotice("The credential has no stable reference.", "error");
      return;
    }
    const confirmed = window.confirm("Clear plugin-owned quota and 429 blocks for " + name + "?\n\nReference: " + reference + "\n\nThis does not enable credentials disabled outside this plugin.");
    if (!confirmed) return;
    try {
      await request(API + "/reset-block", {
        method: "POST",
        body: JSON.stringify({ref: reference})
      });
      showNotice("Plugin-owned blocks cleared for " + name + ".", "success");
      await refreshStatus(true);
    } catch (error) {
      handleRequestError(error, "Block reset failed");
    }
  }
  function bindEvents() {
    els.connect.addEventListener("click", connect);
    els.key.addEventListener("keydown", event => {
      if (event.key === "Enter") {
        event.preventDefault();
        connect();
      }
    });
    els.clearKey.addEventListener("click", clearKey);
    els.remember.addEventListener("change", () => {
      if (!managementKey) return;
      storeKey();
      showNotice(els.remember.checked ? "Key persistence enabled for this browser." : "Key persistence disabled; the session copy remains.", "success");
    });
    els.refresh.addEventListener("click", () => refreshStatus(false));
    els.refreshCredentials.addEventListener("click", () => refreshStatus(false));
    els.reloadSettings.addEventListener("click", reloadAll);
    els.settingsCard.addEventListener("submit", saveSettings);
    els.warmupCard.addEventListener("submit", saveSettings);
    els.runSweep.addEventListener("click", () => runAction(API + "/sweep", "Quota sweep"));
    els.runWarmup.addEventListener("click", () => runAction(API + "/warmup", "Warm-up"));
    els.addSchedule.addEventListener("click", addSchedule);
    els.weeklyEnabled.addEventListener("change", syncWeeklyControls);
    els.warmupEnabled.addEventListener("change", syncWarmupControls);
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden && managementKey) refreshStatus(true);
    });
  }
  function boot() {
    initElements();
    bindEvents();
    setConnectedUi(false);
    syncWeeklyControls();
    syncWarmupControls();
    const storedKey = readStoredKey();
    els.remember.checked = Boolean(localStorage.getItem(LOCAL_KEY));
    if (storedKey) {
      els.key.value = storedKey;
      connect();
    }
  }
  boot();
})();
`
