---
title: Security design
permalink: /security/
---

# Security design

These are requirements for future remote management features, not claims that
those features already exist.

## Secrets

Passwords, key passphrases, proxy passwords, sudo passwords and tokens must never
be written to TOML, process arguments, shell history or diagnostic logs.
Read secrets with terminal echo disabled; require confirmation when storing them.
Store only credential references in inventory.

Use Windows Credential Manager, macOS Keychain or Linux Secret Service.
If a store is unavailable, fail clearly rather than falling back to plaintext.
An encrypted local fallback is deferred until its format, KDF, recovery and tests
receive a separate design review.

## Host identity

Host key verification is on by default. A first connection displays the fingerprint
and requires explicit trust. Changed keys are rejected, including on jump hosts.
Unattended connections to unknown hosts fail. Trust changes require explicit user
action and preserve an auditable fingerprint record.

## Remote operations

Use bounded concurrency and separate connection/command deadlines. Send sudo
passwords through the SSH channel, never by interpolating an echo command.
Quote generated action arguments; arbitrary user shell commands are intentionally
executed by the remote shell.

Updates and reboots support a reviewed plan/dry run and a confirmation showing all
targets; `--yes` is the explicit automation override.
A reboot is successful only after reconnecting and verifying changed boot identity
and a health probe, not merely after an SSH disconnect.

## Output and filesystem

Redact known secret values and sensitive URL userinfo in diagnostic output.
Arbitrary remote stdout/stderr may contain application secrets and cannot be
guaranteed secret-free; document this when execution output is implemented.

Create local configuration with restrictive permissions. Do not overwrite files
on initialization. Windows ACL validation needs dedicated testing before a local
secret database is introduced.

## Reporting

Use [private vulnerability reporting](https://github.com/KanataLabs/fleetsh/security/advisories/new).
See the repository's [security policy](https://github.com/KanataLabs/fleetsh/blob/main/SECURITY.md).
