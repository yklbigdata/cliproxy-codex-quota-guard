# Codex Quota Guard

`codex-quota-guard` is a native [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) v8 plugin that keeps exhausted Codex credentials out of the routing pool and restores them after quota recovery.

Version 2.3 combines quota enforcement, 429 handling, scheduler filtering, scheduled warm-up, durable state, and an interactive management page in one plugin. It replaces `codex-429-autoban` and `codex-5h-quota-warmer` for this deployment.

## Features

- Disable a Codex credential when the 5-hour window has **1% or less remaining**.
- Re-enable it after the same window recovers to **5% or more**.
- Optionally apply the same policy to the 7-day window.
- Honour Codex 429 responses and their `x-codex-*` reset headers.
- Filter blocked credentials in the scheduler path, before a request is sent.
- Run optional daily warm-up requests for each eligible credential.
- Preserve block state, 429 state, ownership markers, and warm-up deduplication across restarts.
- Never re-enable a credential that a person disabled outside the plugin.
- Edit functional settings directly in the CPA plugin page, apply them immediately, run sweeps and warm-up, or reset one plugin-owned block.
- Keep credentials that an operator explicitly re-enables exempt from automatic quota and 429 disabling until the operator restores automatic policy.
- Show each credential once, even after the credential reference changes or persisted state is rebuilt.
- Switch the management page between Chinese and English without reconnecting or discarding unsaved fields.

## Requirements

- CLIProxyAPI v8 with native plugin, usage, scheduler, and management APIs.
- Linux `amd64` for the published shared object.
- Write access to the configured persistent state path.

The current release is built and tested against CLIProxyAPI `v8.0.51`.

## Policy flow

On each sweep the plugin:

1. lists credentials with `host.auth.list` and selects the configured providers;
2. reads each selected credential with `host.auth.get`;
3. reads Codex usage from the configured `usage_url`;
4. applies the 5-hour and optional 7-day hysteresis policy;
5. additionally considers 429 blocks recorded by the usage hook;
6. persists the decision state;
7. updates the credential only when its runtime state must change.

The scheduler plugin also applies persisted quota, 429, and manual-disable blocks. If every candidate is blocked, it rejects the pick instead of falling back to a blocked credential.

## Ownership and manual disables

A credential disabled by this plugin contains a managed `codex_quota_guard` marker. The plugin only automatically enables a credential when that marker proves ownership.

A credential with `disabled: true` and no managed marker has priority and is treated as manually disabled. The plugin does not probe its quota or enable it.

If an operator re-enables a credential that the plugin owns, the plugin records a persistent manual-enable protection. The protection survives restarts. It suppresses automatic quota and 429 disabling until the operator chooses **Resume policy**. A later manual disable takes priority over this protection.

## 429 handling

The usage hook reads the Codex response headers:

- `x-codex-primary-used-percent`
- `x-codex-primary-reset-at`
- `x-codex-primary-window-minutes`
- `x-codex-secondary-used-percent`
- `x-codex-secondary-reset-at`
- `x-codex-secondary-window-minutes`

The active window is selected from the reported usage and reset time. If no usable reset header exists, the configured `rate_limit_fallback` is used.

## Persistent state

The default container path is:

```text
/CLIProxyAPI/plugins/state/codex-quota-guard/state.json
```

With the usual CLIProxyAPI bind mount, the host path is:

```text
/opt/cliproxyapi/plugins/state/codex-quota-guard/state.json
```

State is written atomically with a temporary file, file sync, rename, and directory sync. Linux builds use an exclusive file lock. The directory is created with mode `0700` and the state file with mode `0600`.

The state contains non-secret scheduling metadata: credential identifiers, observed quota percentages, reset times, block reasons, and warm-up status. It does not contain access tokens, refresh tokens, ID tokens, API keys, Management Keys, or request bodies.

## Configuration

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
      keep_at_least_one: false
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

`resume_above_remaining_percent` must be greater than `disable_below_remaining_percent`. The same hysteresis rule applies to the 7-day policy.

`keep_at_least_one` defaults to `false`, so two credentials that independently hit a disable condition can both be disabled. Set it to `true` only when one credential must remain available. It protects the quota policy only; it does not override a 429 block or a manual disable.

An empty `warmup_schedules` list disables scheduled runs while leaving the warm-up subsystem enabled. Each credential runs at most once for the same schedule and local calendar date, including after a restart.

## Management page

The browser page is served at:

```text
/v0/resource/plugins/codex-quota-guard/status
```

It uses the global CPA Management Key described by `remote-management.secret-key`. This is the key that authenticates `/v0/management` and `/v8/management`; it is not an OpenAI key, a Codex OAuth credential, or an API key for proxy clients. The key is sent to the CPA Management API as an `Authorization: Bearer` header.

The page keeps the key in browser memory and `sessionStorage` by default. The optional remember setting stores it in `localStorage` for this browser only. The plugin does not write the key to its HTML, state file, logs, or repository.

Native CPA plugin configuration endpoints are used:

```text
GET   /v0/management/plugins/codex-quota-guard/config
PATCH /v0/management/plugins/codex-quota-guard/config
```

The page validates the functional settings before it sends a PATCH. PATCH performs a shallow merge, so `enabled`, `priority`, and unknown CPA-owned fields remain intact. The page then calls the plugin's `POST /v0/management/plugins/codex-quota-guard/apply-config` route to apply the patch to the running plugin immediately and starts one quota sweep. Deployment-only settings such as `providers`, `state_file`, `usage_url`, `user_agent`, `max_credentials`, and `http_timeout` are not exposed in the page.

The page can start one immediate quota sweep, start a warm-up pass, and edit daily warm-up schedules. It checks only fields that the operator changed for concurrent edits before saving. Block reset requires one credential and browser confirmation. It clears only plugin-owned quota and 429 blocks; credentials disabled outside the plugin remain disabled. A protected credential shows **Resume policy** so the operator can deliberately restore automatic enforcement.

The language selector switches the page between Chinese and English immediately. It keeps the Management Key, loaded status, unsaved policy and warm-up fields, and schedule rows intact. The choice is stored in this browser's `localStorage`.

## Build

```bash
make test
make vet
make build
```

Build output:

```text
dist/codex-quota-guard.so
```

## Install manually

```bash
install -d -m 0700 /opt/cliproxyapi/plugins/state/codex-quota-guard
install -m 0644 dist/codex-quota-guard.so \
  /opt/cliproxyapi/plugins/linux/amd64/codex-quota-guard.so
```

Add the configuration block, then restart CLIProxyAPI once.

The plugin ID comes from the shared-object filename. Keep the filename and configuration key exactly `codex-quota-guard`.

## Install from a plugin store

This repository publishes a CPA-compatible `registry.json`. Add its raw URL to `plugins.store-sources`:

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/yklbigdata/cliproxy-codex-quota-guard/main/registry.json"
```

Then install or update `codex-quota-guard` from the management UI.

## Verify

Open:

```text
http://127.0.0.1:8317/v0/resource/plugins/codex-quota-guard/status
```

Connect the page with the CPA Management Key to view live worker state, active policy, state persistence, warm-up results, and one row per credential. The static resource does not contain credential files or secret values.

Check persistence without exposing credentials:

```bash
sudo stat -c '%a %U:%G %s %n' \
  /opt/cliproxyapi/plugins/state/codex-quota-guard/state.json
sudo jq 'del(.credentials[].email)' \
  /opt/cliproxyapi/plugins/state/codex-quota-guard/state.json
```

Restart CLIProxyAPI and confirm that the status page restores the same block and warm-up state.

## Boundaries

- The plugin trusts the Codex usage endpoint. A failed usage read preserves existing blocks and does not trigger an enable.
- The plugin cannot enable a credential disabled manually.
- This is a native shared library and runs with the same privileges as CLIProxyAPI.
- Rotate credentials if you suspect a credential file has been exposed outside the host.

## License

MIT
