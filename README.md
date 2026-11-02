# Codex Quota Guard

`codex-quota-guard` is a [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) native plugin that enables and disables Codex credentials from the quota Codex itself reports, instead of inferring health from request outcomes.

When the 5h window of a Codex credential is nearly spent, the guard disables that credential so no further traffic is routed to it. Once the window rolls over and the credential has quota again, the guard re-enables it. No 429, retry counter, or error-rate heuristic is involved.

## How it decides

Every sweep the plugin:

1. lists credentials through `host.auth.list` and keeps the ones whose provider is listed in `providers` (default `codex`);
2. reads each credential JSON through `host.auth.get` to obtain its access token and account id;
3. asks the same endpoint the Codex client and the management UI use, `https://chatgpt.com/backend-api/wham/usage`, through `host.http.do`;
4. reads `rate_limit.primary_window.used_percent` (the 5h window) and `rate_limit.secondary_window.used_percent` (the 7d window);
5. disables or re-enables the credential by rewriting its credential JSON, then polls `host.auth.get_runtime` until the runtime record matches the new state.

Defaults: disable at **1% remaining** on the 5h window, re-enable once the 5h window is back to **5% remaining** or more.

## Why the guard writes the credential file itself

`host.auth.save` is the documented way for a plugin to persist credential JSON, and this plugin used it first. It does not work for this purpose on CLIProxyAPI v7.2.156 through v7.3.12: the host rebuilds the runtime record from the payload without applying `disabled`, and then persists that record back to disk, so the credential is enabled again immediately and the `disabled` flag never survives.

The guard therefore replaces the credential file atomically (temporary file plus rename, preserving the original mode) and lets the host's own auth directory watcher synthesize the record, which does honour `disabled`. Every write is confirmed by reading `host.auth.get_runtime`; if the runtime does not reflect the change within five seconds, the sweep reports an error instead of claiming success.

## Ownership

The guard only re-enables credentials it disabled itself. Every disable writes a marker next to the credential:

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

A credential that a person disabled by hand is skipped: the guard does not read its quota, does not touch it, and does not re-enable it. Re-enabling removes the marker and clears `disabled`.

## Configuration

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-quota-guard:
      enabled: true
      priority: 100

      # Providers handled by this plugin.
      providers: ["codex"]

      # Disable when the 5h window remaining quota reaches this percentage.
      disable_below_remaining_percent: 1
      # Re-enable when the 5h window remaining quota is back to this percentage.
      resume_above_remaining_percent: 5

      # Optional 7d window policy. Off by default.
      enforce_weekly_window: false
      weekly_disable_below_remaining_percent: 1
      weekly_resume_above_remaining_percent: 5

      # Sweep cadence. The interval floor is 15s.
      poll_interval: 60s
      initial_delay: 15s
      http_timeout: 25s

      # Never disable every credential of one provider; the one with the most
      # remaining quota stays enabled.
      keep_at_least_one: true

      # Report decisions without writing credential state.
      dry_run: false

      max_credentials: 50
      usage_url: "https://chatgpt.com/backend-api/wham/usage"
      user_agent: "codex-cli/0.154.0"
```

`resume_above_remaining_percent` must be greater than `disable_below_remaining_percent`; the reverse would make the credential flap on every sweep.

## Install

Build the shared library for the platform that runs CLIProxyAPI:

```bash
make build           # -> dist/codex-quota-guard.so (linux/amd64)
```

Then place it where the host discovers plugins and add the configuration block:

```bash
install -m 0644 dist/codex-quota-guard.so /opt/cliproxyapi/plugins/linux/amd64/codex-quota-guard.so
```

The plugin id comes from the file name, so the file must be `codex-quota-guard.so` and the configuration key must be `codex-quota-guard`. Restart CLIProxyAPI, or install through the plugin store, so the host loads the new library.

## Verify

The plugin registers one management resource page:

```
http://127.0.0.1:8317/v0/resource/plugins/codex-quota-guard/status
```

It shows the sweep counters, the active policy, and one row per credential with the 5h/7d usage, the runtime disabled flag, the guard ownership flag, and the last decision. The same state reaches the host log:

```
codex-quota-guard: credential disabled   credential=codex-...json reason="5h window remaining 0.4% <= 1%"
codex-quota-guard: credential re-enabled credential=codex-...json reason="5h window recovered to 12% remaining"
```

To exercise the guard without waiting for a real quota exhaustion, raise `disable_below_remaining_percent` above the current usage of one credential, restart, and confirm the credential flips to disabled. Lower it back and the next sweep re-enables it.

## Limits

- The plugin trusts the Codex usage endpoint. If that call fails, the credential keeps its current state; the guard never re-enables on a failed read.
- Disabling is a hard filter: an exhausted credential leaves the routing pool entirely instead of cooling down and retrying.
- Tokens are never logged. The guard writes only the credential JSON it read, with the fields it owns replaced.

## License

MIT
