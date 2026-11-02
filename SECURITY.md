# Security

## What this plugin touches

`codex-quota-guard` runs inside the CLIProxyAPI process as a native shared library, so it has the same privileges as the host.

- It reads credential JSON through `host.auth.get` to obtain the access token and account id needed for the Codex usage probe.
- It calls the Codex usage endpoint through `host.http.do`, so the host proxy, transport policy, and request logging still apply.
- It observes Codex 429 response headers through the native usage-plugin API.
- It calls `host.auth.save` only when a managed credential must be disabled or enabled, changing `disabled` and the `codex_quota_guard` marker.
- It writes scheduler and warm-up state as a non-secret JSON document under the configured `state_file` path.

## What it never does

- It never logs tokens, credential JSON, or request bodies.
- It never re-enables a credential that was disabled outside the plugin; the `codex_quota_guard` marker is required ownership evidence.
- It never writes any credential secret into its persistent state.
- It never rewrites unrelated credential fields.

## Persistent state

The state file contains credential identifiers, quota percentages, reset times, block reasons, ownership flags, and warm-up results. Linux builds create the state directory with mode `0700`, create the state file with mode `0600`, use an exclusive lock, and replace the file atomically.

The state path must remain on a persistent volume. A container-local path without a bind mount does not survive replacement of the container.

## Reporting

Open a private security advisory on the repository, or contact the maintainer directly, rather than filing a public issue for anything that could expose credentials.
