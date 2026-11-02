# Security

## What this plugin touches

`codex-quota-guard` runs inside the CLIProxyAPI process as a native shared library, so it has the same privileges as the host.

- It reads credential JSON through `host.auth.get` to obtain the access token and account id needed for the Codex usage probe.
- It calls the Codex usage endpoint through `host.http.do`, so the host proxy, transport policy, and request logging still apply.
- It rewrites the credential file it owns the quota decision for, replacing only `disabled` and the `codex_quota_guard` marker.

## What it never does

- It never logs tokens, credential JSON, or request bodies.
- It never re-enables a credential that was disabled outside the plugin; the `codex_quota_guard` marker is required ownership evidence.
- It never writes to a credential that has no backing file.

## Reporting

Open a private security advisory on the repository, or contact the maintainer directly, rather than filing a public issue for anything that could expose credentials.
