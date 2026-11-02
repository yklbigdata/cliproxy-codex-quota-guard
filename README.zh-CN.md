# Codex Quota Guard

[English](README.md) | 简体中文

`codex-quota-guard` 是 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) v8 原生插件。它根据 Codex 自己报告的额度，把耗尽额度的凭证移出路由池，并在额度恢复后重新启用。

2.1.0 将额度守卫、429 处理、调度器过滤、定时预热、持久状态和交互管理页合并到一个插件。本部署中，它替代 `codex-429-autoban` 与 `codex-5h-quota-warmer`。

## 功能

- 5h 窗口剩余额度 **小于等于 1%** 时禁用凭证。
- 同一窗口恢复到 **大于等于 5%** 时启用凭证。
- 可选启用 7d 窗口策略。
- 根据 Codex 429 响应和 `x-codex-*` 重置响应头封锁凭证。
- 在调度器选择凭证前过滤额度封锁、429 封锁和人工禁用凭证。
- 可为每个符合条件的凭证执行每日预热。
- 重启后恢复封锁、429、归属标记和预热去重状态。
- 不自动启用插件外人工禁用的凭证。
- 可在 CPA 插件页修改功能配置、立即巡检、立即预热，或清除单个凭证的插件封锁。

## 要求

- 支持原生插件、Usage、Scheduler 和 Management API 的 CLIProxyAPI v8。
- 发布包面向 Linux `amd64`。
- 配置的持久状态目录必须可写。

当前版本使用 CLIProxyAPI `v8.0.51` 构建和测试。

## 策略流程

每轮巡检执行以下步骤：

1. 通过 `host.auth.list` 获取凭证，并筛选配置的 provider；
2. 通过 `host.auth.get` 读取凭证；
3. 从配置的 `usage_url` 读取 Codex 额度；
4. 应用 5h 窗口和可选 7d 窗口的滞回策略；
5. 合并 Usage Hook 记录的 429 封锁；
6. 保存持久状态；
7. 仅在运行时状态需要改变时更新凭证。

插件还会在 Scheduler 路径中过滤持久化的额度、429 和人工禁用状态。所有候选均被封锁时，插件拒绝本次选择，不回退到已封锁凭证。

## 归属标记与人工禁用

插件禁用凭证时，会在凭证 JSON 中写入 `codex_quota_guard` 归属标记。只有该标记存在时，插件才会自动重新启用凭证。

凭证为 `disabled: true` 且没有该标记时，插件将其视为人工禁用。人工禁用优先级最高；插件不读取它的额度，也不启用它。

## 429 处理

Usage Hook 读取以下 Codex 响应头：

- `x-codex-primary-used-percent`
- `x-codex-primary-reset-at`
- `x-codex-primary-window-minutes`
- `x-codex-secondary-used-percent`
- `x-codex-secondary-reset-at`
- `x-codex-secondary-window-minutes`

插件根据使用率和重置时间选择活动窗口。响应头缺少可用重置时间时，使用 `rate_limit_fallback`。

## 持久状态

容器内默认路径：

```text
/CLIProxyAPI/plugins/state/codex-quota-guard/state.json
```

使用常见 CLIProxyAPI bind mount 时，宿主机路径：

```text
/opt/cliproxyapi/plugins/state/codex-quota-guard/state.json
```

状态文件采用临时文件、文件同步、rename 和目录同步的原子写入流程。Linux 构建使用独占文件锁。目录权限为 `0700`，状态文件权限为 `0600`。

状态只保存非敏感调度数据：凭证标识、观测到的额度百分比、重置时间、封锁原因和预热状态。状态不保存 access token、refresh token、ID token、API Key、Management Key 或请求正文。

## 配置

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-quota-guard:
      enabled: true

      providers: ["codex"]

      disable_below_remaining_percent: 1
      resume_above_remaining_percent: 5

      enforce_weekly_window: false
      weekly_disable_below_remaining_percent: 1
      weekly_resume_above_remaining_percent: 5

      poll_interval: 60s
      initial_delay: 15s
      http_timeout: 25s
      keep_at_least_one: true
      dry_run: false

      max_credentials: 50
      usage_url: "https://chatgpt.com/backend-api/wham/usage"
      user_agent: "codex-cli/0.154.0"

      state_file: /CLIProxyAPI/plugins/state/codex-quota-guard/state.json
      rate_limit_fallback: 5h

      warmup_enabled: true
      warmup_model: gpt-6-luna
      warmup_prompt: "Reply with exactly: quota window activated"
      warmup_timezone: Asia/Singapore
      warmup_schedules:
        - id: daily
          at: "09:00"
          enabled: false
```

`resume_above_remaining_percent` 必须大于 `disable_below_remaining_percent`。7d 策略也使用相同的滞回规则。

`keep_at_least_one` 只保护额度策略，不覆盖 429 封锁或人工禁用。

`warmup_schedules` 为空时，不执行定时预热，但预热子系统保持启用。同一凭证在同一日程、同一本地日期最多执行一次；重启后仍保留该去重结果。

## 管理页

浏览器页面地址：

```text
/v0/resource/plugins/codex-quota-guard/status
```

页面使用 `remote-management.secret-key` 配置的全局 CPA Management Key。该密钥用于认证 `/v0/management` 和 `/v8/management`；它不是 OpenAI Key、Codex OAuth 凭证或代理客户端 API Key。页面通过 `Authorization: Bearer` 请求头把密钥发送给 CPA Management API。

页面默认只在浏览器内存和 `sessionStorage` 中保留密钥。“Remember in this browser” 选项启用后，才会写入该浏览器的 `localStorage`。插件不会把密钥写入 HTML、状态文件、日志或仓库。

页面使用 CPA 原生插件配置接口：

```text
GET   /v0/management/plugins/codex-quota-guard/config
PATCH /v0/management/plugins/codex-quota-guard/config
```

保存前，页面先校验功能配置，再发送 PATCH。PATCH 使用浅合并，因此 `enabled`、`priority` 和其他 CPA 管理的未知字段会保留。页面不暴露 `providers`、`state_file`、`usage_url`、`user_agent`、`max_credentials`、`http_timeout` 等部署内部字段。

页面可立即执行一次额度巡检、执行一次预热，并编辑每日预热日程。保存前会检测其他操作者是否修改过功能配置；发生并发修改时，页面要求重新加载。清除封锁必须指定一个凭证并在浏览器中确认。清除操作只移除插件拥有的额度和 429 封锁；插件外人工禁用的凭证不会被启用。

## 构建

```bash
make test
make vet
make build
```

构建产物：

```text
dist/codex-quota-guard.so
```

## 手动安装

```bash
install -d -m 0700 /opt/cliproxyapi/plugins/state/codex-quota-guard
install -m 0644 dist/codex-quota-guard.so \
  /opt/cliproxyapi/plugins/linux/amd64/codex-quota-guard.so
```

添加配置后，重启一次 CLIProxyAPI。

插件 ID 来自动态库文件名。文件名和配置键必须均为 `codex-quota-guard`。

## 通过插件商店安装

仓库发布 CPA `registry.json`。将以下原始地址加入 `plugins.store-sources`：

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/yklbigdata/cliproxy-codex-quota-guard/main/registry.json"
```

然后可在管理界面安装或更新 `codex-quota-guard`。

## 验证

打开：

```text
http://127.0.0.1:8317/v0/resource/plugins/codex-quota-guard/status
```

使用 CPA Management Key 连接页面后，可查看实时工作状态、当前策略、状态保存情况、预热结果和每个凭证的最近结果。静态页面资源不包含凭证文件或密钥。

检查持久状态且不输出凭证敏感字段：

```bash
sudo stat -c '%a %U:%G %s %n' \
  /opt/cliproxyapi/plugins/state/codex-quota-guard/state.json
sudo jq 'del(.credentials[].email)' \
  /opt/cliproxyapi/plugins/state/codex-quota-guard/state.json
```

重启 CLIProxyAPI 后，确认状态页恢复相同的封锁和预热状态。

## 边界

- 插件信任 Codex 额度接口。额度读取失败时保留现封锁状态，不会因此启用凭证。
- 插件不能启用人工禁用的凭证。
- 插件是原生动态库，权限与 CLIProxyAPI 进程相同。
- 若凭证文件可能在主机外泄露，请轮换凭证。

## 许可证

MIT
