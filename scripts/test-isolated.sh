#!/usr/bin/env bash
set -euo pipefail

repo_dir="${1:?repository directory is required}"
image="${2:-eceasy/cli-proxy-api:latest}"
work_dir="${3:-$(mktemp -d /tmp/codex-quota-guard-isolated.XXXXXX)}"
port="${4:-18319}"
container_name="codex-quota-guard-isolated-$$"
state_dir="$work_dir/plugins/state/codex-quota-guard"
state_file="$state_dir/state.json"

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
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-quota-guard:
      enabled: true
      dry_run: true
      initial_delay: 2s
      poll_interval: 10m
      keep_at_least_one: true
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
  grep -q "2.0.0" "$work_dir/status-$suffix.html"
  grep -q "fixture-codex.json" "$work_dir/status-$suffix.html"
}

wait_for_startup_state() {
  local previous_updated="${1:-}"
  for _ in $(seq 1 60); do
    if jq -e \
      --arg previous "$previous_updated" \
      '.last_sweep_cause == "startup" and
       .updated_at != $previous and
       .credentials["auth-persist"].quota_block.kind == "quota_5h"' \
      "$state_file" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "persistent state did not reach startup state" >&2
  jq . "$state_file" >&2 || true
  exit 1
}

docker run -d --name "$container_name" \
  --network bridge \
  -p "127.0.0.1:${port}:${port}" \
  -v "$work_dir/config.yaml:/CLIProxyAPI/config.yaml:ro" \
  -v "$work_dir/plugins:/CLIProxyAPI/plugins" \
  -v "$work_dir/auths:/CLIProxyAPI/auths" \
  -v "$work_dir/logs:/CLIProxyAPI/logs" \
  "$image" >/dev/null

wait_ready
wait_for_startup_state "fixture-before-restart"
check_status before

before_updated="$(jq -r '.updated_at' "$state_file")"
before_credentials="$(jq -cS '.credentials' "$state_file")"

docker stop "$container_name" >/dev/null
docker start "$container_name" >/dev/null
wait_ready
wait_for_startup_state "$before_updated"
check_status after

after_credentials="$(jq -cS '.credentials' "$state_file")"
if [ "$before_credentials" != "$after_credentials" ]; then
  echo "credential state changed across isolated restart" >&2
  exit 1
fi

if [ "$(stat -c '%a' "$state_dir")" != "700" ]; then
  echo "state directory mode is not 700" >&2
  exit 1
fi
if [ "$(stat -c '%a' "$state_file")" != "600" ]; then
  echo "state file mode is not 600" >&2
  exit 1
fi

if grep -Eiq 'access_token|refresh_token|id_token|management_key|api_key|authorization' "$state_file"; then
  echo "persistent state contains a forbidden secret field" >&2
  exit 1
fi

if docker logs "$container_name" 2>&1 | grep -Eiq "plugin load failed|panic|fatal error"; then
  docker logs "$container_name" 2>&1 | tail -n 120
  exit 1
fi

echo "isolated_plugin_ok=true"
echo "persistent_restart_ok=true"
echo "state_mode=$(stat -c '%a' "$state_file")"
echo "state_dir_mode=$(stat -c '%a' "$state_dir")"
echo "work_dir=$work_dir"
