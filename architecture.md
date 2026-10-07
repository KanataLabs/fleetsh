---
title: Architecture
permalink: /architecture/
---

# Architecture

## Boundaries

The CLI owns argument parsing and presentation. Inventory, secrets, connection
construction, execution and built-in actions are separate packages.
No package may silently disable host key verification or persist plaintext secrets.

```text
CLI
 ├─ Inventory: TOML, host validation, groups/tags, selectors
 ├─ Credentials: OS store, prompted input, credential references
 ├─ Transport: direct SSH, ProxyJump, SOCKS5, later HTTP CONNECT
 ├─ Executor: deadlines, bounded concurrency, ordered per-host results
 ├─ Actions: exec, update, reboot, later status
 └─ Output: terminal and structured JSON
```

## Current repository

| Path | Responsibility |
| --- | --- |
| `cmd/fleetsh/` | Process entry point and build version |
| `internal/cli/` | Foundation commands and exit codes |
| `internal/config/` | Embedded TOML example and exclusive file creation |
| `docs/` | Documentation source |
| `.github/workflows/` | CI and documentation publishing |

Additional packages should be added when their functionality is implemented,
rather than as empty scaffolding.

## Planned implementation choices

Use Cobra for the full command hierarchy, a maintained TOML parser,
`golang.org/x/crypto/ssh` for in-process SSH and
`golang.org/x/net/proxy` for SOCKS5. Evaluate native OS credential adapters for
Windows Credential Manager, macOS Keychain and Linux Secret Service.
Do not require the OpenSSH executable for the core SSH implementation.

All remote work receives a context with cancellation and separate connection and
command deadlines. Parallelism defaults to 10; reboot defaults to 2.
Resources, agent sockets and jump-host chains must close on every failure path.

## Results and exit codes (proposed)

Each result contains host ID, stdout, stderr, remote exit status, duration and an
error category. Console-only hosts are explicitly skipped.

- 0: every executable target succeeded.
- 1: at least one remote command failed (takes precedence in mixed failures).
- 2: invalid local arguments, inventory or configuration.
- 3: connection/authentication/proxy/host-key failure without a command failure.

Only 0 and 2 are used by the current foundation CLI.
Never retry a command automatically; future retries apply only before command
dispatch to avoid duplicating side effects.

## Validation

Use local SSH fixtures to verify auth, host keys, jump hosts, proxy negotiation,
PTY behavior, timeouts and cancellation. Test credential adapters per OS.
Cross-compile with CGO disabled and verify that releases run without language runtimes.
