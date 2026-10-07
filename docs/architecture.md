---
title: Architecture
permalink: /architecture/
---

# Architecture

The CLI owns arguments and presentation. Packages separate inventory, credentials,
transport, execution and built-in actions.

| Path | Responsibility |
| --- | --- |
| `cmd/fleetsh/` | Entrypoint, version, interrupt cancellation |
| `internal/cli/` | Cobra commands, prompting, JSON/terminal output |
| `internal/config/` | Exclusive initial configuration creation |
| `internal/inventory/` | Strict TOML, validation, selectors, locked atomic saves |
| `internal/credentials/` | OS keyring, hidden terminal input, known-secret redaction |
| `internal/transport/` | SSH auth, Agent, jump/SOCKS5, known_hosts, keepalive |
| `internal/executor/` | Worker pool, deadlines, capped output, PTY, sudo |
| `internal/actions/` | OS-aware update and verified Linux reboot |
| `internal/testutil/` | Local-only SSH/Agent/proxy fixtures |
| `docs/` | Jekyll source; published through gh-pages |

## Connection lifecycle

Credentials and agent keys are resolved before parallel work to prevent prompt
interleaving. Each target owns its SSH client and jump chain.
All network work receives cancellation and a connection deadline.
Unknown host keys require confirmation; changed keys are rejected.
Keepalive probes run every 15 seconds and close unresponsive connections.

Execution starts a separate command deadline after connecting, closes the SSH
client on cancellation and preserves stdout/stderr/remote exit status.
Output is capped at 4 MiB per stream and reports truncation.
The worker pool limits simultaneous target operations; sorted inventory selectors
produce stable output order.

## State changes

A process-level advisory file lock protects cooperating inventory writers.
Validated TOML is written to a private temporary file, synchronized and renamed.
CLI edits rewrite comments. Known-host trust uses a separate lock and rechecks
identity after confirmation to handle concurrent writers.

Updates probe every selected executable host before dispatching any update.
Reboots capture the original boot ID, dispatch the command and reconnect until
both a changed boot ID and health probe are verified.

## Exit status

- 0: all executable targets succeeded; console-only targets may be skipped.
- 1: remote command failure, command timeout or cancellation.
- 2: local usage/configuration or credential preparation error.
- 3: connection/authentication/proxy/host-key failure, including connection timeout.

In a mixed remote result, code 1 takes precedence over 3.
Interactive nonzero shell exit returns 1. There are no automatic command retries.

## Dependencies and validation

Cobra, go-toml/v2, x/crypto/ssh, x/net/proxy, x/term, go-keyring,
go-winio and flock are pinned in go.mod/go.sum.
Core SSH does not invoke an OpenSSH executable. macOS Keychain uses its standard
`security` utility with secret input through stdin; Linux uses Secret Service/D-Bus.

Tests exercise local SSH servers, encrypted keys, Unix sockets/Windows Agent pipes,
strict trust, target/jump identity, authenticated proxies, deadlines, sudo input,
worker bounds, output caps, update confirmation and reboot identity.
CI runs race tests and six CGO-free cross-builds; native store tests use temporary
fixture entries in isolated OS credential stores.
