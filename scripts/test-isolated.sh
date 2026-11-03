#!/usr/bin/env bash
set -euo pipefail

repo_dir="${1:?repository directory is required}"
image="${2:-eceasy/cli-proxy-api:latest}"
work_dir="${3:-$(mktemp -d /tmp/codex-quota-guard-isolated.XXXXXX)}"
port="${4:-18319}"
container_name="codex-quota-guard-isolated-$$"
state_dir="$work_dir/plugins/state/codex-quota-guard"
state_file="$state_dir/state.json"
management_key="isolated-management-key-3f071e4f"

mkdir -p "$work_dir/plugins/linux/amd64" "$work_dir/auths" "$work_dir/logs" "$state_dir"
chmod 0700 "$state_dir"
cp "$repo_dir/dist/codex-quota-guard.so" "$work_dir/plugins/linux/amd64/codex-quota-guard.so"

cat >"$state_file" <<'JSON'
{
  "schema_version": 1,
  "updated_at": "2026-11-02T00:00:00Z",
  "last_sweep_at": "2026-11-02T00:00:00Z",
  "last_sweep_cause": "fixture-before-restart",
  "warmup": {
    "last_run_at": "2026-11-02T00:00:00Z",
    "last_source": "fixture",
    "results": 1,
    "success": 1
  },
  "credentials": {
    "auth-persist": {
      "key": "auth-persist",
      "auth_id": "auth-persist",
      "name": "fixture-codex.json",
      "provider": "codex",
      "disabled_by_plugin": true,
      "quota_block": {
        "kind": "quota_5h",
        "reason": "persistence fixture",
        "window": "5h",
        "status": "active",
        "blocked_at": "2026-11-02T00:00:00Z",
        "reset_at": "2099-11-02T00:00:00Z"
      },
      "observed_quota": {
        "primary_remaining_percent": 1,
        "primary_reset_at": "2099-11-02T00:00:00Z"
      },
      "warmup": {
        "last_run_at": "2026-11-02T00:00:00Z",
        "last_run_key": "schedule:daily:2026-11-02",
        "last_success": true,
        "last_status": 200
      }
    }
  }
}
JSON
chmod 0600 "$state_file"

cat >"$work_dir/config.yaml" <<YAML
host: 0.0.0.0
port: $port
auth-dir: /CLIProxyAPI/auths
debug: true
remote-management:
  allow-remote: true
  secret-key: $management_key
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-quota-guard:
      enabled: true
      dry_run: true
      initial_delay: 2s
      poll_interval: 10m
      keep_at_least_one: false
      warmup_enabled: false
      state_file: /CLIProxyAPI/plugins/state/codex-quota-guard/state.json
YAML

cleanup() {
  docker rm -f "$container_name" >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_ready() {
  local ready=false
  for _ in $(seq 1 180); do
    if curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null 2>&1; then
      ready=true
      break
    fi
    sleep 0.5
  done
  if [ "$ready" != "true" ]; then
    echo "isolated CPA did not become ready" >&2
    docker logs "$container_name" 2>&1 | tail -n 120
    exit 1
  fi
}

check_status() {
  local suffix="$1"
  local status_code
  status_code="$(curl -sS -o "$work_dir/status-$suffix.html" -w '%{http_code}' \
    "http://127.0.0.1:${port}/v0/resource/plugins/codex-quota-guard/status" || true)"
  echo "status_http_$suffix=$status_code"
  if [ "$status_code" != "200" ]; then
    docker logs "$container_name" 2>&1 | tail -n 120
    exit 1
  fi
  grep -q "Codex Quota Guard" "$work_dir/status-$suffix.html"
  grep -q "2.3.0" "$work_dir/status-$suffix.html"
  grep -q "Quota policy" "$work_dir/status-$suffix.html"
  grep -q "Scheduled warm-up" "$work_dir/status-$suffix.html"
  grep -q 'id="language"' "$work_dir/status-$suffix.html"
  grep -q 'codexQuotaGuardLanguage' "$work_dir/status-$suffix.html"
  if grep -Fq "$management_key" "$work_dir/status-$suffix.html"; then
    echo "status resource contains the management key" >&2
    exit 1
  fi
}

management_curl() {
  curl -sS -H "Authorization: Bearer $management_key" "$@"
}

read_state_file() {
  docker exec "$container_name" cat /CLIProxyAPI/plugins/state/codex-quota-guard/state.json
}

check_management_api() {
  local status_code
  local config_before
  local config_after
  local api_status

  status_code="$(curl -sS -o "$work_dir/management-unauthorized.json" -w '%{http_code}' \
    "http://127.0.0.1:${port}/v0/management/plugins/codex-quota-guard/config" || true)"
  echo "management_unauthorized_http=$status_code"
  if [ "$status_code" != "401" ]; then
    docker logs "$container_name" 2>&1 | tail -n 120
    exit 1
  fi

  config_before="$(management_curl \
    "http://127.0.0.1:${port}/v0/management/plugins/codex-quota-guard/config")"
  if ! jq -e '.enabled == true and .dry_run == true and .state_file != null' <<<"$config_before" >/dev/null; then
    echo "unexpected plugin config before patch: $config_before" >&2
    docker logs "$container_name" 2>&1 | tail -n 120 >&2
    exit 1
  fi

  if ! jq -e '.keep_at_least_one == false' <<<"$config_before" >/dev/null; then
    echo "keep_at_least_one did not default to false: $config_before" >&2
    exit 1
  fi

  management_curl -X POST -H "Content-Type: application/json" \
    --data '{"disable_below_remaining_percent":2.5}' \
    "http://127.0.0.1:${port}/v0/management/plugins/codex-quota-guard/validate-config" \
    >"$work_dir/validate-config.json"
  if ! jq -e '.status == "valid" and .settings.disable_below_remaining_percent == 2.5' \
    "$work_dir/validate-config.json" >/dev/null; then
    echo "validate-config response is invalid" >&2
    exit 1
  fi

  management_curl -X PATCH -H "Content-Type: application/json" \
    --data '{"disable_below_remaining_percent":2.5}' \
    "http://127.0.0.1:${port}/v0/management/plugins/codex-quota-guard/config" \
    >"$work_dir/patch-config.json"
  if ! jq -e '.status == "ok"' "$work_dir/patch-config.json" >/dev/null; then
    echo "plugin config PATCH failed: $(cat "$work_dir/patch-config.json")" >&2
    docker logs "$container_name" 2>&1 | tail -n 120 >&2
    exit 1
  fi

  management_curl -X POST -H "Content-Type: application/json" \
    --data '{"disable_below_remaining_percent":2.5}' \
    "http://127.0.0.1:${port}/v0/management/plugins/codex-quota-guard/apply-config" \
    >"$work_dir/apply-config.json"
  if ! jq -e '.status == "applied" and .settings.disable_below_remaining_percent == 2.5' \
    "$work_dir/apply-config.json" >/dev/null; then
    echo "apply-config response is invalid" >&2
    cat "$work_dir/apply-config.json" >&2
    exit 1
  fi

  for _ in $(seq 1 40); do
    api_status="$(management_curl \
      "http://127.0.0.1:${port}/v0/management/plugins/codex-quota-guard/status")"
    if jq -e '.settings.disable_below_remaining_percent == 2.5' <<<"$api_status" >/dev/null 2>&1; then
      break
    fi
    sleep 0.5
  done
  if ! jq -e '.settings.disable_below_remaining_percent == 2.5' <<<"$api_status" >/dev/null 2>&1; then
    echo "running plugin did not apply the PATCHed setting" >&2
    echo "$api_status" >&2
    exit 1
  fi

  config_after="$(management_curl \
    "http://127.0.0.1:${port}/v0/management/plugins/codex-quota-guard/config")"
  if ! jq -e '.enabled == true and .dry_run == true and .state_file != null and .disable_below_remaining_percent == 2.5' \
    <<<"$config_after" >/dev/null; then
    echo "plugin config PATCH lost host-owned or functional fields: $config_after" >&2
    exit 1
  fi
  echo "management_config_patch_ok=true"
  echo "management_config_apply_ok=true"
}

wait_for_startup_state() {
  local previous_updated="${1:-}"
  local current_state
  for _ in $(seq 1 60); do
    current_state="$(read_state_file 2>/dev/null || true)"
    if [ -n "$current_state" ] && jq -e \
      --arg previous "$previous_updated" \
      '.last_sweep_cause == "startup" and
       .updated_at != $previous and
       .credentials["auth-persist"].quota_block.kind == "quota_5h"' \
      <<<"$current_state" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "persistent state did not reach startup state" >&2
  read_state_file | jq . >&2 || true
  exit 1
}

docker run -d --name "$container_name" \
  --network bridge \
  -p "127.0.0.1:${port}:${port}" \
  -v "$work_dir/config.yaml:/CLIProxyAPI/config.yaml" \
  -v "$work_dir/plugins:/CLIProxyAPI/plugins" \
  -v "$work_dir/auths:/CLIProxyAPI/auths" \
  -v "$work_dir/logs:/CLIProxyAPI/logs" \
  "$image" >/dev/null

wait_ready
wait_for_startup_state "fixture-before-restart"
check_status before
check_management_api

before_updated="$(read_state_file | jq -r '.updated_at')"
before_credentials="$(read_state_file | jq -cS '.credentials')"

docker stop "$container_name" >/dev/null
docker start "$container_name" >/dev/null
wait_ready
wait_for_startup_state "$before_updated"
check_status after

after_credentials="$(read_state_file | jq -cS '.credentials')"
if [ "$before_credentials" != "$after_credentials" ]; then
  echo "credential state changed across isolated restart" >&2
  exit 1
fi

if [ "$(docker exec "$container_name" stat -c '%a' /CLIProxyAPI/plugins/state/codex-quota-guard)" != "700" ]; then
  echo "state directory mode is not 700" >&2
  exit 1
fi
if [ "$(docker exec "$container_name" stat -c '%a' /CLIProxyAPI/plugins/state/codex-quota-guard/state.json)" != "600" ]; then
  echo "state file mode is not 600" >&2
  exit 1
fi

if read_state_file | grep -Eiq 'access_token|refresh_token|id_token|management_key|api_key|authorization'; then
  echo "persistent state contains a forbidden secret field" >&2
  exit 1
fi

if { read_state_file; cat "$work_dir"/status-*.html; } | grep -Fq "$management_key"; then
  echo "isolated state or HTML contains the management key" >&2
  exit 1
fi

if docker logs "$container_name" 2>&1 | grep -Fq "$management_key"; then
  echo "isolated CPA logs contain the management key" >&2
  exit 1
fi

if docker logs "$container_name" 2>&1 | grep -Eiq "plugin load failed|panic|fatal error"; then
  docker logs "$container_name" 2>&1 | tail -n 120
  exit 1
fi

echo "isolated_plugin_ok=true"
echo "persistent_restart_ok=true"
echo "state_mode=$(docker exec "$container_name" stat -c '%a' /CLIProxyAPI/plugins/state/codex-quota-guard/state.json)"
echo "state_dir_mode=$(docker exec "$container_name" stat -c '%a' /CLIProxyAPI/plugins/state/codex-quota-guard)"
echo "work_dir=$work_dir"
