---
title: Security design
permalink: /en/security/
lang: en
locale: en
page_key: 'security/'
---

# Security design

fleetsh is a development build. The following implemented controls and remaining
release checks guide contributors.

## Secrets

Inventory stores credential references, never secret values. Strict decoding rejects
plaintext password fields and unsupported fields without echoing source lines in errors.
Passwords/passphrases use hidden terminal input; echoed/piped secret input is rejected.
Credential replacement requires an explicit `--replace`.

Windows Credential Manager, macOS Keychain and Linux Secret Service hold values.
Store errors fail without plaintext fallback. macOS uses `security -i` and stdin,
so secrets never enter process arguments. Secret size is limited to 2560 bytes.
Agent sockets and private keys stay local.

sudo secrets are transmitted through SSH channel stdin. Commands are never constructed
with `echo password` or credential values. Known secrets are redacted from buffered
execution output and diagnostics; arbitrary application secrets cannot be recognized.
Interactive shell output is passed through unchanged to preserve terminal behavior.
There is no persisted execution log/history.

## Host identity

Strict known_hosts validation applies to targets and every jump host.
First connections show fingerprints and require an interactive `y`; automation fails
on unknown keys. Changed keys are rejected without a new-trust prompt.
Trust writes are locked and revalidated after confirmation.

`hostkey reset` requires confirmation or `--yes`. Shared/wildcard/marked records
require manual editing so resetting one host cannot silently remove broader trust.

## Actions and resources

Execution uses bounded concurrency, connection/command deadlines, cancellation,
keepalive and output caps. User-supplied shell commands execute intentionally on the
remote shell. Generated action commands use fixed templates and safe shell quoting.

Update dry run probes OS only. Any preflight failure prevents all updates.
Actual update/reboot requires confirmation showing selected aliases, or explicit
`--yes` automation approval. Reboot concurrency defaults to 2 and success requires
a new boot identity plus a successful uptime probe.

## Files and platform checks

Inventory updates use advisory locks, private temporary files and atomic replacement.
Initialization never overwrites an existing file. Inventory and known_hosts symlinks
are refused. New Unix directories/files use 0700/0600; Windows uses directory ACLs.
Credential values are not exported in inventory backups and are not recoverable
from reference names alone.

Tests use localhost fixtures and temporary OS-store entries. No real VPS is needed.
The roadmap distinguishes completed test coverage from the remaining release work.

## Reporting

Use [private vulnerability reporting](https://github.com/KanataLabs/fleetsh/security/advisories/new).
See [SECURITY.md](https://github.com/KanataLabs/fleetsh/blob/main/SECURITY.md).
