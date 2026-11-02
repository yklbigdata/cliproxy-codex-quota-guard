# Codex Quota Guard

`codex-quota-guard` 是一个 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 原生插件：它完全依据 Codex 自己上报的额度来启用/禁用凭证，而不是根据调用成功失败来推断凭证健康度。

当某个 Codex 凭证的 5h 窗口额度接近耗尽时，插件会禁用该凭证，让后续流量不再路由到它；等到窗口重置、额度恢复后，插件再自动启用该凭证。整个过程不依赖 429、重试次数或错误率。

## 判定流程

每轮巡检插件会：

1. 通过 `host.auth.list` 列出凭证，只保留 `providers` 中声明的 provider（默认 `codex`）；
2. 通过 `host.auth.get` 读取凭证 JSON，取得 access token 与 account id；
3. 通过 `host.http.do` 访问 Codex 客户端与管理界面使用的同一个接口 `https://chatgpt.com/backend-api/wham/usage`；
4. 读取 `rate_limit.primary_window.used_percent`（5h 窗口）与 `rate_limit.secondary_window.used_percent`（7d 窗口）；
5. 重写凭证 JSON 实现禁用或启用，然后轮询 `host.auth.get_runtime`，直到运行时记录与新状态一致。

默认策略：5h 窗口剩余额度 **≤ 1% 时禁用**，恢复到 **≥ 5% 时重新启用**。

## 为什么直接写凭证文件

`host.auth.save` 是插件持久化凭证 JSON 的官方接口，本插件最初也使用它。但在 CLIProxyAPI v7.2.156 到 v7.3.12 上它无法完成禁用：宿主在重建运行时记录时不会应用 `disabled` 字段，随后又把该记录回写磁盘，于是凭证立刻恢复为启用状态，`disabled` 永远留不下来。

因此插件改为原子替换凭证文件（临时文件 + rename，保留原文件权限），再由宿主自己的凭证目录 watcher 生成记录——这条链路是正确识别 `disabled` 的。每次写入后都会通过 `host.auth.get_runtime` 复核；若 5 秒内运行时状态没有跟上，巡检会报错，而不是假装成功。

## 归属标记

插件只会重新启用由它自己禁用的凭证。每次禁用都会在凭证 JSON 里写入标记：

```json
{
  "disabled": true,
  "codex_quota_guard": {
    "managed": true,
    "disabled_at": "2026-09-22T15:04:05Z",
    "reason": "5h window remaining 0.4% <= 1% (plan plus)",
    "remaining_percent": 0.4,
    "window_reset_at": "2026-09-22T19:00:00Z",
    "plugin_version": "1.0.0"
  }
}
```

人工禁用的凭证会被完全跳过：不读取额度、不修改文件、也不会被重新启用。重新启用时会清除该标记并把 `disabled` 置回 `false`。

## 配置

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-quota-guard:
      enabled: true
      priority: 100

      # 需要接管的 provider
      providers: ["codex"]

      # 5h 窗口剩余额度低于该百分比时禁用
      disable_below_remaining_percent: 1
      # 5h 窗口剩余额度恢复到该百分比以上时启用
      resume_above_remaining_percent: 5

      # 是否同时参考 7d 窗口，默认关闭
      enforce_weekly_window: false
      weekly_disable_below_remaining_percent: 1
      weekly_resume_above_remaining_percent: 5

      # 巡检节奏，间隔下限为 15s
      poll_interval: 60s
      initial_delay: 15s
      http_timeout: 25s

      # 同一 provider 不会全部被禁用，剩余额度最多的那个会被保留
      keep_at_least_one: true

      # 只报告决策结果，不写入凭证状态
      dry_run: false

      max_credentials: 50
      usage_url: "https://chatgpt.com/backend-api/wham/usage"
      user_agent: "codex-cli/0.154.0"
```

`resume_above_remaining_percent` 必须大于 `disable_below_remaining_percent`，否则凭证会在每轮巡检中反复切换。

## 安装

先为运行 CLIProxyAPI 的平台编译动态库：

```bash
make build           # 生成 dist/codex-quota-guard.so (linux/amd64)
```

把动态库放到宿主的插件发现目录，并加上配置：

```bash
install -m 0644 dist/codex-quota-guard.so /opt/cliproxyapi/plugins/linux/amd64/codex-quota-guard.so
```

插件 ID 来自文件名，因此文件名必须是 `codex-quota-guard.so`，配置键必须是 `codex-quota-guard`。写入后重启 CLIProxyAPI，或通过插件商店安装，让宿主加载新的动态库。

## 验证

插件注册了一个管理资源页：

```
http://127.0.0.1:8317/v0/resource/plugins/codex-quota-guard/status
```

页面会显示巡检计数、当前策略，以及每个凭证的 5h/7d 用量、运行时禁用状态、插件归属标记和最近一次决策。同样的信息也会写进宿主日志：

```
codex-quota-guard: credential disabled   credential=codex-...json reason="5h window remaining 0.4% <= 1%"
codex-quota-guard: credential re-enabled credential=codex-...json reason="5h window recovered to 12% remaining"
```

要快速验证而不用等到真实耗尽，可以临时把 `disable_below_remaining_percent` 调到高于某个凭证当前用量，重启后确认该凭证被禁用；再调回默认值，下一轮巡检就会自动启用它。

## 边界

- 插件信任 Codex 额度接口。该接口读取失败时凭证保持原状态，插件不会在读取失败时重新启用凭证。
- 禁用是硬过滤：额度耗尽的凭证会直接退出路由池，而不是进入冷却后重试。
- 不会输出任何 token；写回时也只替换插件自己负责的字段。

## 许可证

MIT
