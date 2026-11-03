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
    .language-control{display:flex;gap:7px;align-items:center}
    .language-control select{width:auto;min-width:104px;padding:6px 8px}
  </style>
</head>
<body>
<main>
  <header>
    <div>
      <h1>Codex Quota Guard</h1>
      <p class="muted">Use Codex-reported quota to manage credentials, 429 blocks, scheduler filtering, and warm-up.</p>
    </div>
    <div class="row">
      <label class="language-control muted small"><span>Language</span><select id="language" aria-label="Language"><option value="zh-CN">中文</option><option value="en">English</option></select></label>
      <div class="pill" id="version">not connected</div>
    </div>
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

  <form class="card" id="settings-card">
    <div class="section-head"><div><h2>Quota policy</h2><p class="muted small">Edit these values directly on this page. Saving validates the fields, updates the native CPA plugin configuration, applies the policy immediately, and starts one quota sweep. Connect first to enable editing.</p></div><div class="row"><button type="button" id="reload-settings">Reload</button><button class="primary" type="submit" id="save-settings">Validate and save</button></div></div>
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

  <form class="card" id="warmup-card">
<div class="section-head"><div><h2>Scheduled warm-up</h2><p class="muted small">Warm-up runs only for eligible Codex credentials. It skips manually disabled credentials and policy blocks that are not covered by a manual-enable override.</p></div><div class="row"><button type="button" id="add-schedule">Add schedule</button><button class="primary" type="submit" id="save-warmup">Validate and save</button></div></div>
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
    <div class="section-head"><div><h2>Credentials and last decisions</h2><p class="muted small">Recent sweep decisions and warm-up results. Block reset and manual-enable override reset are available only for plugin-owned state.</p></div><button id="refresh-credentials">Refresh</button></div>
    <div class="table-wrap"><table><thead><tr><th>Credential</th><th>Plan</th><th>5h used</th><th>7d used</th><th>State</th><th>Action and reason</th><th>Warm-up</th><th>Checked</th><th>Operation</th></tr></thead><tbody id="credentials-body"></tbody></table></div>
    <p class="muted small" id="credentials-empty" style="margin-top:10px">No sweep data loaded.</p>
  </section>

  <p class="footer">Codex Quota Guard 2.3.0. Settings use the CPA Management API. Credential files and tokens are never returned to this page.</p>
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
  const LANGUAGE_KEY = "codexQuotaGuardLanguage";
  const ZH_STATIC = {
    "Language": "语言",
    "Use Codex-reported quota to manage credentials, 429 blocks, scheduler filtering, and warm-up.": "根据 Codex 返回的额度管理凭证、429 封锁、调度过滤和定时预热。",
    "not connected": "未连接",
    "Management connection": "管理连接",
    "CPA Management Key": "CPA 管理密钥",
    "Enter the CPA management credential": "输入 CPA 管理凭据",
    "This is the global CLIProxyAPI management credential. It is not an OpenAI key, Codex credential, or plugin key.": "这是 CLIProxyAPI 的全局管理凭据，不是 OpenAI Key、Codex 凭证或插件专用密钥。",
    "Storage": "存储",
    "Remember in this browser": "在此浏览器中记住",
    "By default the key is kept only in this tab session. Browser persistence is opt-in. The plugin never stores this key.": "默认只在当前标签页会话中保存密钥。浏览器持久化需要手动启用，插件本身从不保存该密钥。",
    "Connect and load": "连接并加载",
    "Clear key": "清除密钥",
    "Authentication is handled by the CPA management middleware.": "身份验证由 CPA 管理中间件处理。",
    "Live status": "实时状态",
    "The page reads live state after each connect. It does not embed credential data in the HTML resource.": "页面在每次连接后读取实时状态，不会在 HTML 资源中嵌入凭证数据。",
    "Refresh": "刷新",
    "Run quota sweep now": "立即巡检额度",
    "Run warm-up now": "立即预热",
    "Quota policy": "额度策略",
    "Edit these values directly on this page. Saving validates the fields, updates the native CPA plugin configuration, applies the policy immediately, and starts one quota sweep. Connect first to enable editing.": "可直接在本页修改这些参数。保存时会校验字段、更新 CPA 原生插件配置、立即应用策略并启动一次额度巡检。连接管理密钥后可编辑。",
    "Reload": "重新加载",
    "Validate and save": "校验并保存",
    "Disable 5h at remaining (%)": "5 小时剩余额度低于（%）时禁用",
    "Resume 5h at remaining (%)": "5 小时剩余额度高于（%）时恢复",
    "Sweep interval": "巡检间隔",
    "Duration format, minimum 15s.": "时长格式，最小值为 15 秒。",
    "Initial delay": "启动延迟",
    "Delay before the startup sweep.": "启动后首次巡检前的等待时间。",
    "Enforce the 7-day window": "启用 7 天窗口策略",
    "Disable 7d at remaining (%)": "7 天剩余额度低于（%）时禁用",
    "Resume 7d at remaining (%)": "7 天剩余额度高于（%）时恢复",
    "Keep at least one credential": "至少保留一个可用凭证",
    "Dry run": "演练模式",
    "429 fallback duration": "429 默认封锁时长",
    "Used when a Codex 429 response has no reset header.": "当 Codex 返回 429 且没有重置时间响应头时使用。",
    "Scheduled warm-up": "定时预热",
    "Warm-up runs only for eligible Codex credentials. It skips manually disabled credentials and policy blocks that are not covered by a manual-enable override.": "预热只对符合条件的 Codex 凭证执行。人工禁用凭证会被跳过；存在人工启用保护的凭证不受额度封锁过滤。",
    "Add schedule": "添加日程",
    "Enable scheduled warm-up": "启用定时预热",
    "Warm-up model": "预热模型",
    "IANA timezone": "IANA 时区",
    "Warm-up prompt": "预热提示词",
    "Daily schedules": "每日日程",
    "No schedules configured.": "尚未配置日程。",
    "Credentials and last decisions": "凭证与最近决策",
    "Recent sweep decisions and warm-up results. Block reset and manual-enable override reset are available only for plugin-owned state.": "显示最近巡检决策和预热结果。仅插件拥有的封锁状态和人工启用保护可以清除。",
    "Credential": "凭证",
    "Plan": "套餐",
    "5h used": "5 小时已用",
    "7d used": "7 天已用",
    "State": "状态",
    "Action and reason": "动作与原因",
    "Warm-up": "预热",
    "Checked": "检查时间",
    "Operation": "操作",
    "No sweep data loaded.": "尚未加载巡检数据。",
    "Schedule ID": "日程 ID",
    "Local time": "本地时间",
    "Enabled": "启用",
    "Remove": "删除",
    "Codex Quota Guard 2.3.0. Settings use the CPA Management API. Credential files and tokens are never returned to this page.": "Codex Quota Guard 2.3.0。设置通过 CPA Management API 保存。此页面不会返回凭证文件或令牌。"
  };
  const ZH_MESSAGES = {
    "Management key cleared.": "管理密钥已清除。",
    "Enter and connect a CPA Management Key first.": "请先输入并连接 CPA 管理密钥。",
    "Authentication failed. Enter the current CPA Management Key again.": "身份验证失败，请重新输入当前 CPA 管理密钥。",
    "Enter the CPA Management Key first.": "请先输入 CPA 管理密钥。",
    "Connecting to the CPA Management API...": "正在连接 CPA Management API…",
    "Connected. Live status and effective settings are loaded.": "连接成功，已加载实时状态和当前生效配置。",
    "Live status refreshed.": "实时状态已刷新。",
    "Connect first.": "请先连接。",
    "Saved configuration and live status reloaded.": "已重新加载保存的配置和实时状态。",
    "Reload settings before saving.": "保存前请先重新加载配置。",
    "Functional settings changed after this page loaded them.": "页面加载后，功能配置已被其他操作修改。",
    "These edited fields changed after this page loaded them: {fields}": "以下正在编辑的字段在页面加载后已被其他操作修改：{fields}",
    "No configuration changes to save.": "没有需要保存的配置变更。",
    "{context}: The live configuration changed. Reload settings, review the values, and save again. {message}": "{context}：实时配置已变化。请重新加载、核对数值后再保存。{message}",
    "{context}: {message}": "{context}：{message}",
    "Configuration saved. Waiting for CPA to apply the plugin reconfiguration...": "配置已保存，正在等待 CPA 应用插件重配置…",
    "Configuration saved and applied by the running plugin.": "配置已保存，运行中的插件已应用新配置。",
    "Configuration saved, applied, and one quota sweep started.": "配置已保存并应用，已启动一次额度巡检。",
    "Configuration saved, applied, but the immediate quota sweep could not start: {message}": "配置已保存并应用，但无法启动即时额度巡检：{message}",
    "Configuration is saved, but the running plugin has not reported the new values yet. Refresh status before another edit.": "配置已保存，但运行中的插件尚未报告新值。请在再次编辑前刷新状态。",
    "Configuration saved, but immediate apply failed": "配置已保存，但立即应用失败",
    "{label} started.": "{label}已启动。",
    "{label} failed": "{label}失败",
    "The credential has no stable reference.": "该凭证没有稳定引用，无法执行操作。",
    "Clear plugin-owned quota and 429 blocks for {name}?\n\nReference: {reference}\n\nThis does not enable credentials disabled outside this plugin.": "要清除 {name} 的插件额度封锁和 429 封锁吗？\n\n引用：{reference}\n\n此操作不会启用插件外人工禁用的凭证。",
    "Plugin-owned blocks cleared for {name}.": "已清除 {name} 的插件封锁。",
    "Clear the manual-enable override for {name}? The next sweep may disable this credential again if its quota still meets the disable condition.": "要清除 {name} 的人工启用保护吗？如果额度仍达到禁用条件，下一次巡检可能再次禁用该凭证。",
    "Manual-enable override cleared for {name}.": "已清除 {name} 的人工启用保护。",
    "Restore automatic quota policy for {name}?\n\nReference: {reference}\n\nThis clears the manual-enable protection and plugin-owned blocks. The next sweep may disable this credential again if its quota still meets the disable condition.": "要为 {name} 恢复自动额度策略吗？\n\n引用：{reference}\n\n此操作会清除人工启用保护和插件拥有的封锁。如果额度仍达到禁用条件，下一次巡检可能再次禁用该凭证。",
    "Automatic quota policy restored for {name}.": "已为 {name} 恢复自动额度策略。",
    "Restore policy failed": "恢复自动策略失败",
    "Block reset failed": "清除封锁失败",
    "Key persistence enabled for this browser.": "已为此浏览器启用密钥持久化。",
    "Key persistence disabled; the session copy remains.": "已关闭密钥持久化，当前会话中的副本仍保留。",
    "Connect failed": "连接失败",
    "Refresh failed": "刷新失败",
    "Reload failed": "重新加载失败",
    "Save failed": "保存失败",
    "Quota sweep": "额度巡检",
    "Warm-up": "预热",
    "Worker": "工作线程",
    "running": "运行中",
    "stopped": "已停止",
    "Started {time}": "启动于 {time}",
    "Quota sweep": "额度巡检",
    "idle": "空闲",
    "{cause} · {count} total": "{cause} · 共 {count} 次",
    "no sweep": "尚未巡检",
    "startup": "启动",
    "interval": "定时轮询",
    "manual": "手动",
    "429": "429 回调",
    "Last sweep": "最近巡检",
    "Next {time}": "下次 {time}",
    "No enabled schedule": "没有启用的日程",
    "Actions": "操作",
    "{count} disabled": "禁用 {count} 次",
    "{enabled} enabled · {errors} errors": "启用 {enabled} 次 · 错误 {errors} 次",
    "State": "状态",
    "read-only": "只读",
    "writable": "可写",
    "Saved {time}": "保存于 {time}",
    "Not saved yet": "尚未保存",
    "Warm-up results": "预热结果",
    "{success} / {total}": "{success} / {total}",
    "{count} skipped": "跳过 {count} 次",
    "Health": "健康状态",
    "needs attention": "需要关注",
    "no state errors": "无状态错误",
    "Persistent state and workers report no current error": "持久状态和工作线程当前均无错误",
    "attention": "注意",
    "manual disabled": "人工禁用",
    "manual enable protected": "人工启用保护",
    "5h quota": "5 小时额度",
    "429 active": "429 生效",
    "plugin disabled": "插件已禁用",
    "file disabled": "文件已禁用",
    "runtime disabled": "运行时已禁用",
    "within policy": "符合策略",
    "reset in {duration}": "{duration}后重置",
    "success": "成功",
    "failed": "失败",
    "Reset block": "清除封锁",
    "Resume policy": "恢复自动策略",
    "none": "无",
    "disabled": "已禁用",
    "enabled": "已启用",
    "error": "错误",
    "skipped": "已跳过",
    "rate_limited": "触发限流",
    "{days}d {hours}h": "{days}天 {hours}小时",
    "{hours}h {minutes}m": "{hours}小时 {minutes}分",
    "{minutes}m": "{minutes}分"
  };
  const ZH_REASONS = {
    "cannot identify credential": "无法识别凭证",
    "cannot read credential": "无法读取凭证",
    "cannot parse credential": "无法解析凭证",
    "disabled outside this plugin; manual disable has priority": "该凭证在插件外被禁用，人工禁用优先",
    "credential is manually disabled": "凭证已被人工禁用",
    "runtime disabled outside this plugin": "凭证在插件外被运行时禁用",
    "credential has no access token": "凭证没有访问令牌",
    "cannot read Codex quota; existing blocks are preserved": "无法读取 Codex 额度，现有封锁保持不变",
    "protected: keeping the last Codex credential available": "保护策略：保留最后一个可用的 Codex 凭证",
    "manual enable override: quota disable suppressed": "人工启用保护：已抑制额度禁用",
    "clear guard marker failed": "清除旧守卫标记失败",
    "disable failed": "禁用失败",
    "update guard marker failed": "更新守卫标记失败",
    "block expired": "封锁已到期",
    "enable failed": "启用失败",
    "within policy": "符合策略",
    "manual plugin block reset": "已人工清除插件封锁",
    "waiting for a 5h window reading": "正在等待 5 小时窗口数据"
  };
  const ZH = Object.assign({}, ZH_STATIC, ZH_MESSAGES, ZH_REASONS);
  let LANG = detectLanguage();
  const originalText = new WeakMap();
  const originalAttributes = new WeakMap();
  const els = {};
  let managementKey = "";
  let connectionActive = false;
  let currentStatus = null;
  let noticeState = null;

  function byId(id) { return document.getElementById(id); }
  function detectLanguage() {
    const saved = localStorage.getItem(LANGUAGE_KEY);
    if (saved === "zh-CN" || saved === "en") return saved;
    const browserLanguages = Array.isArray(navigator.languages) && navigator.languages.length
      ? navigator.languages
      : [navigator.language || "en"];
    return browserLanguages.some(value => String(value).toLowerCase().startsWith("zh")) ? "zh-CN" : "en";
  }
  function interpolate(template, values) {
    if (!values) return template;
    return String(template).replace(/\{([a-zA-Z0-9_]+)\}/g, (match, key) => (
      Object.prototype.hasOwnProperty.call(values, key) ? String(values[key]) : match
    ));
  }
  function t(key, values) {
    const template = LANG === "zh-CN" && Object.prototype.hasOwnProperty.call(ZH, key) ? ZH[key] : key;
    return interpolate(template, values);
  }
  function localizeKnownMessage(value) {
    const text = String(value || "");
    if (!text) return "";
    const exact = t(text);
    if (exact !== text) return exact;
    let match = text.match(/^dry run: ([\s\S]+)$/);
    if (match) return (LANG === "zh-CN" ? "演练：" : "dry run: ") + localizeKnownMessage(match[1]);
    match = text.match(/^(5h|7d) window remaining ([\d.]+%) <= ([\d.]+%)$/);
    if (match) return LANG === "zh-CN"
      ? match[1] + " 窗口剩余 " + match[2] + " <= " + match[3]
      : text;
    match = text.match(/^(5h|7d) window remaining ([\d.]+%) < ([\d.]+%)$/);
    if (match) return LANG === "zh-CN"
      ? match[1] + " 窗口剩余 " + match[2] + " < " + match[3]
      : text;
    match = text.match(/^(5h|7d) window recovered to ([\d.]+%) remaining$/);
    if (match) return LANG === "zh-CN"
      ? match[1] + " 窗口已恢复至剩余 " + match[2]
      : text;
    match = text.match(/^429 rate limit active: ([\s\S]+)$/);
    if (match) return LANG === "zh-CN"
      ? "429 限流生效：" + localizeKnownMessage(match[1])
      : text;
    match = text.match(/^HTTP 429 from Codex; (5h|7d) window resets at ([\s\S]+)$/);
    if (match) return LANG === "zh-CN"
      ? "Codex 返回 HTTP 429；" + match[1] + " 窗口将在 " + match[2] + " 重置"
      : text;
    return text;
  }
  function setLocalizedText(node, key) {
    node.dataset.i18nText = key;
    node.textContent = t(key);
    return node;
  }
  function applyLocalizedDom(root) {
    root.querySelectorAll("[data-i18n-text]").forEach(node => {
      node.textContent = t(node.dataset.i18nText || "");
    });
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    let node = walker.nextNode();
    while (node) {
      const parent = node.parentElement;
      const current = originalText.has(node) ? originalText.get(node) : node.nodeValue;
      if (!originalText.has(node)) originalText.set(node, current);
      if (parent && !["SCRIPT", "STYLE", "OPTION"].includes(parent.tagName) && current.trim()) {
        const match = current.match(/^(\s*)([\s\S]*?)(\s*)$/);
        if (match) {
          const translated = LANG === "zh-CN" ? (ZH_STATIC[match[2]] || match[2]) : match[2];
          node.nodeValue = match[1] + translated + match[3];
        }
      }
      node = walker.nextNode();
    }
    root.querySelectorAll("[placeholder], [title], [aria-label]").forEach(element => {
      let values = originalAttributes.get(element);
      if (!values) {
        values = {};
        originalAttributes.set(element, values);
      }
      ["placeholder", "title", "aria-label"].forEach(attribute => {
        if (!element.hasAttribute(attribute)) return;
        if (!Object.prototype.hasOwnProperty.call(values, attribute)) values[attribute] = element.getAttribute(attribute);
        const original = values[attribute] || "";
        const translated = LANG === "zh-CN" ? (ZH_STATIC[original] || original) : original;
        element.setAttribute(attribute, translated);
      });
    });
  }
  function applyLanguage(next, persist) {
    LANG = next === "zh-CN" ? "zh-CN" : "en";
    if (persist) localStorage.setItem(LANGUAGE_KEY, LANG);
    document.documentElement.lang = LANG;
    if (els.language) els.language.value = LANG;
    applyLocalizedDom(document.body);
    renderNotice();
    if (currentStatus) {
      renderStatus(currentStatus);
    } else if (els.version) {
      els.version.textContent = t("not connected");
    }
  }
  function renderNotice() {
    const text = noticeState
      ? (noticeState.key ? t(noticeState.key, noticeState.params) : localizeKnownMessage(noticeState.text))
      : "";
    const node = els.notice;
    node.textContent = text;
    node.className = "notice" + (text ? " show" : "") + (noticeState && noticeState.kind ? " " + noticeState.kind : "");
  }
  function showNotice(text, kind) {
    noticeState = {text: String(text || ""), kind: kind || ""};
    renderNotice();
  }
  function showNoticeKey(key, kind, params) {
    noticeState = {key, kind: kind || "", params: params || null};
    renderNotice();
  }
  function setConnectedUi(connected) {
    connectionActive = Boolean(connected);
    byId("status-card").classList.toggle("hidden", !connectionActive);
    byId("credentials-card").classList.toggle("hidden", !connectionActive);
    byId("settings-card").classList.remove("hidden");
    byId("warmup-card").classList.remove("hidden");
    syncFormEditingState();
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
    showNoticeKey("Management key cleared.", "success");
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
      error.code = data && data.error ? String(data.error) : "";
      throw error;
    }
    return data || {};
  }
  function formatTime(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return date.toLocaleString(LANG, {dateStyle: "medium", timeStyle: "medium"});
  }
  function formatSeconds(value) {
    const seconds = Number(value || 0);
    if (!seconds) return "—";
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days) return t("{days}d {hours}h", {days, hours});
    if (hours) return t("{hours}h {minutes}m", {hours, minutes});
    return t("{minutes}m", {minutes: Math.max(0, minutes)});
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
  let baselineSettings = "";
  let baselineEffectiveSettings = null;
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
  function currentFormSettings() {
    return normalizeSettings(functionalPatch());
  }
  function changedFunctionalFields(current, baseline) {
    if (!baseline) return FUNCTIONAL_FIELDS.slice();
    return FUNCTIONAL_FIELDS.filter(key => stable(current[key]) !== stable(baseline[key]));
  }
  function patchForFields(settings, fields) {
    const patch = {};
    fields.forEach(key => { patch[key] = settings[key]; });
    return patch;
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
    els.disable7d.disabled = !connectionActive || !enabled;
    els.resume7d.disabled = !connectionActive || !enabled;
    els.weeklyFields.classList.toggle("muted", !enabled);
  }
  function syncWarmupControls() {
    const enabled = els.warmupEnabled.checked;
    els.warmupModel.disabled = !connectionActive || !enabled;
    els.warmupTimezone.disabled = !connectionActive || !enabled;
    els.warmupPrompt.disabled = !connectionActive || !enabled;
    els.addSchedule.disabled = !connectionActive || !enabled;
    els.scheduleList.querySelectorAll("input,button").forEach(node => { node.disabled = !connectionActive || !enabled; });
  }
  function syncFormEditingState() {
    [els.settingsCard, els.warmupCard].forEach(card => {
      if (!card) return;
      card.querySelectorAll("input,select,textarea,button").forEach(node => { node.disabled = !connectionActive; });
    });
    if (!connectionActive) return;
    syncWeeklyControls();
    syncWarmupControls();
  }
  function renderSchedules(schedules) {
    els.scheduleList.replaceChildren();
    const items = normalizeSchedules(schedules);
    if (!items.length) {
      const empty = document.createElement("p");
      empty.className = "muted small";
      setLocalizedText(empty, "No schedules configured.");
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
    idField.append(setLocalizedText(document.createElement("span"), "Schedule ID"));
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
    timeField.append(setLocalizedText(document.createElement("span"), "Local time"));
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
    enabledLabel.append(enabled, setLocalizedText(document.createElement("span"), "Enabled"));

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "danger";
    setLocalizedText(remove, "Remove");
    remove.addEventListener("click", () => { row.remove(); ensureScheduleEmpty(); });

    row.append(idField, timeField, enabledLabel, remove);
    return row;
  }
  function ensureScheduleEmpty() {
    if (!els.scheduleList.querySelector("[data-schedule-row]")) {
      const empty = document.createElement("p");
      empty.className = "muted small";
      setLocalizedText(empty, "No schedules configured.");
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
    name.textContent = t(label);
    const primary = document.createElement("div");
    primary.className = "v";
    primary.textContent = value || "—";
    card.append(name, primary);
    if (sub) {
      const note = document.createElement("div");
      note.className = "cell-sub";
      note.textContent = sub;
      if (kind) note.append(" ");
      if (kind) note.append(badge(kind === "bad" ? t("attention") : t(kind), kind));
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
    els.statusGrid.append(statCard("Worker", t(worker), status.started_at ? t("Started {time}", {time: formatTime(status.started_at)}) : "", status.running ? "ok" : "bad"));
    els.statusGrid.append(statCard("Quota sweep", t(status.sweep_running ? "running" : "idle"),
      t("{cause} · {count} total", {cause: localizeKnownMessage(status.last_sweep_cause || "no sweep"), count: Number(status.sweeps || 0)}), status.sweep_running ? "warn" : "ok"));
    els.statusGrid.append(statCard("Last sweep", formatTime(status.last_sweep),
      t("Next {time}", {time: formatTime(status.next_sweep)}), ""));
    els.statusGrid.append(statCard("Warm-up", t(status.warmup_running ? "running" : "idle"),
      status.next_warmup ? t("Next {time}", {time: formatTime(status.next_warmup)}) + (status.next_warmup_id ? " · " + status.next_warmup_id : "") : t("No enabled schedule"), status.warmup_running ? "warn" : ""));
    els.statusGrid.append(statCard("Actions", t("{count} disabled", {count: Number(status.disables || 0)}),
      t("{enabled} enabled · {errors} errors", {enabled: Number(status.enables || 0), errors: Number(status.errors || 0)}), Number(status.errors || 0) ? "warn" : ""));
    els.statusGrid.append(statCard("State", t(status.state_read_only ? "read-only" : "writable"),
      status.state_saved_at ? t("Saved {time}", {time: formatTime(status.state_saved_at)}) : t("Not saved yet"), status.state_read_only ? "bad" : "ok"));
    els.statusGrid.append(statCard("Warm-up results", t("{success} / {total}", {success: Number(status.warmup_success || 0), total: Number(status.warmup_results || 0)}),
      t("{count} skipped", {count: Number(status.warmup_skipped || 0)}), status.warmup_error ? "warn" : ""));
    const warningText = [status.state_error, status.last_sweep_error, status.warmup_error].filter(Boolean).join(" · ");
    els.statusGrid.append(statCard("Health", t(warningText ? "needs attention" : "no state errors"),
      warningText || t("Persistent state and workers report no current error"), warningText ? "bad" : "ok"));

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
    if (report.manual_disabled) wrap.append(badge(t("manual disabled"), "bad"));
    if (report.manual_enabled && !report.manual_disabled) wrap.append(badge(t("manual enable protected"), "warn"));
    if (report.quota_blocked) wrap.append(badge(t("5h quota"), "warn"));
    if (report.rate_limit_blocked) wrap.append(badge(t("429 active"), "warn"));
    if (report.managed) wrap.append(badge(t("plugin disabled"), "warn"));
    if (report.file_disabled && !report.managed && !report.manual_disabled) wrap.append(badge(t("file disabled"), "bad"));
    if (report.runtime_disabled && !report.file_disabled) wrap.append(badge(t("runtime disabled"), "bad"));
    if (!wrap.childNodes.length) wrap.append(badge(t("within policy"), "ok"));
    return wrap;
  }
  function resetCaption(seconds) {
    if (!seconds) return "";
    return t("reset in {duration}", {duration: formatSeconds(seconds)});
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
      actionWrap.append(badge(t(report.action || "none"), actionKind(report.action)));
      const actionText = [localizeKnownMessage(report.reason), localizeKnownMessage(report.error)].filter(Boolean).join(": ");
      action.append(actionWrap, line("", actionText, false));
      const warmup = document.createElement("td");
      const warmupText = report.warmup_last_run_at ? t(report.warmup_last_success ? "success" : "failed") : "—";
      warmup.append(line(warmupText, localizeKnownMessage(report.warmup_last_error) || (report.warmup_last_status ? "HTTP " + report.warmup_last_status : ""), false));
      const checked = document.createElement("td");
      checked.className = "muted";
      checked.textContent = formatTime(report.checked_at);
      const operation = document.createElement("td");
      if (report.can_reset_block) {
        const reset = document.createElement("button");
        reset.type = "button";
        reset.className = "danger";
        reset.textContent = t(report.manual_enabled ? "Resume policy" : "Reset block");
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
    els.saveSettings.disabled = busy || !connectionActive;
    els.saveWarmup.disabled = busy || !connectionActive;
    els.reloadSettings.disabled = busy || !connectionActive;
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
      language: byId("language"),
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
    error.code = "configuration_conflict";
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
    if (error && error.code === "configuration_conflict") {
      showNoticeKey("{context}: The live configuration changed. Reload settings, review the values, and save again. {message}", "error", {
        context: t(context || "Save failed"),
        message: localizeKnownMessage(message)
      });
      return;
    }
    if (context) {
      showNoticeKey("{context}: {message}", "error", {
        context: t(context),
        message: localizeKnownMessage(message)
      });
    } else {
      showNotice(message, "error");
    }
  }
  function setBaselines(rawConfig, status) {
    baselineConfig = cloneJson(rawConfig && typeof rawConfig === "object" ? rawConfig : {}, {});
    const effective = settingsFromConfig(baselineConfig, status && status.settings ? status.settings : {});
    applySettings(effective);
    baselineEffectiveSettings = normalizeSettings(effective);
    baselineSettings = stable(baselineEffectiveSettings);
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
    const current = currentFormSettings();
    const changedFields = changedFunctionalFields(current, baselineEffectiveSettings);
    if (changedFields.length === 0) {
      showNotice(t("No configuration changes to save."), "success");
      return;
    }
    let nativeConfigSaved = false;
    setSaveBusy(true);
    try {
      const results = await Promise.all([
        request(API + "/status"),
        request(CONFIG_API)
      ]);
      const latestStatus = results[0];
      const latestConfig = results[1];
      const latestEffective = normalizeSettings(settingsFromConfig(latestConfig, latestStatus.settings));
      const conflicts = changedFields.filter(key => stable(latestEffective[key]) !== stable(baselineEffectiveSettings[key]));
      if (conflicts.length) {
        throw conflict(t("These edited fields changed after this page loaded them: {fields}", {fields: conflicts.join(", ")}));
      }

      const patch = patchForFields(current, changedFields);
      const expected = normalizeSettings(Object.assign({}, latestEffective, patch));
      await request(API + "/validate-config", {
        method: "POST",
        body: JSON.stringify(patch)
      });
      await request(CONFIG_API, {
        method: "PATCH",
        body: JSON.stringify(patch)
      });
      nativeConfigSaved = true;
      await request(API + "/apply-config", {
        method: "POST",
        body: JSON.stringify(patch)
      });

      baselineConfig = Object.assign({}, latestConfig, patch);
      baselineEffectiveSettings = expected;
      baselineSettings = stable(expected);
      showNotice("Configuration saved. Waiting for CPA to apply the plugin reconfiguration...", "");
      const applied = await waitForSettings(expected, 15000);
      const finalStatus = await request(API + "/status");
      const finalConfig = await request(CONFIG_API);
      renderStatus(finalStatus);
      if (applied) {
        setBaselines(finalConfig, finalStatus);
        let sweepError = null;
        try {
          await request(API + "/sweep", {method: "POST"});
        } catch (error) {
          if (!error || error.status !== 409) sweepError = error;
        }
        if (sweepError) {
          showNoticeKey("Configuration saved, applied, but the immediate quota sweep could not start: {message}", "error", {
            message: localizeKnownMessage(sweepError.message || String(sweepError))
          });
        } else {
          showNoticeKey("Configuration saved, applied, and one quota sweep started.", "success");
          scheduleStatusPoll(500);
        }
      } else {
        applySettings(expected);
        baselineConfig = finalConfig;
        baselineEffectiveSettings = normalizeSettings(settingsFromConfig(finalConfig, finalStatus.settings));
        baselineSettings = stable(expected);
        showNotice("Configuration is saved, but the running plugin has not reported the new values yet. Refresh status before another edit.", "error");
      }
    } catch (error) {
      handleRequestError(error, nativeConfigSaved ? "Configuration saved, but immediate apply failed" : "Save failed");
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
      showNoticeKey("{label} started.", "success", {label: t(label)});
      await refreshStatus(true);
      scheduleStatusPoll(500);
    } catch (error) {
      handleRequestError(error, t(label) + " failed");
    }
  }
  async function resetBlock(report) {
    const name = report.name || report.ref || "this credential";
    const reference = report.ref || report.name || "";
    if (!reference) {
      showNotice("The credential has no stable reference.", "error");
      return;
    }
    const confirmKey = report.manual_enabled
      ? "Restore automatic quota policy for {name}?\n\nReference: {reference}\n\nThis clears the manual-enable protection and plugin-owned blocks. The next sweep may disable this credential again if its quota still meets the disable condition."
      : "Clear plugin-owned quota and 429 blocks for {name}?\n\nReference: {reference}\n\nThis does not enable credentials disabled outside this plugin.";
    const confirmed = window.confirm(t(confirmKey, {name, reference}));
    if (!confirmed) return;
    try {
      await request(API + "/reset-block", {
        method: "POST",
        body: JSON.stringify({ref: reference})
      });
      showNoticeKey(report.manual_enabled ? "Automatic quota policy restored for {name}." : "Plugin-owned blocks cleared for {name}.", "success", {name});
      await refreshStatus(true);
    } catch (error) {
      handleRequestError(error, t(report.manual_enabled ? "Restore policy failed" : "Block reset failed"));
    }
  }
  function bindEvents() {
    els.language.addEventListener("change", () => applyLanguage(els.language.value, true));
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
    els.language.value = LANG;
    applyLanguage(LANG, false);
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
