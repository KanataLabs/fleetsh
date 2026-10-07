---
title: Roadmap
permalink: /roadmap/
---

# Roadmap

## Current: foundation

- [x] Public Go repository in KanataLabs, GPL-3.0-only.
- [x] `help`, `version`, `init` with exclusive configuration creation.
- [x] English and Chinese introduction, product brief, architecture and security design.
- [x] CI for Windows/macOS/Linux, amd64/arm64 cross-compilation.
- [x] Documentation source on main and publication through gh-pages.

There is no production-ready VPS management release yet.

## v0.1: usable fleet management

- [ ] TOML inventory: add/edit/rm/ls/show, groups/tags, console-only assets.
- [ ] SSH keys (including encrypted keys), passwords and SSH agent.
- [ ] Native OS credential stores and private prompted input.
- [ ] Direct SSH, ProxyJump and SOCKS5, including authenticated proxies.
- [ ] Interactive shell/PTY, host key trust/reset, keepalive and deadlines.
- [ ] Single/batch exec, bounded concurrency, per-host results and JSON output.
- [ ] Update with OS detection, dry run and confirmation.
- [ ] Reboot with reconnect, boot identity verification and health probe.
- [ ] Tagged release binaries and checksums for supported OS/architectures.

Remote execution must not be released before the security gates below pass.

## v0.2: convenience and interoperability

- [ ] HTTP CONNECT proxy.
- [ ] status and reboot-required.
- [ ] Custom command aliases and `run`.
- [ ] sudo credential references.
- [ ] OpenSSH config import and TOML/JSON/OpenSSH export.
- [ ] Connection retry, serial/rolling execution.

The original brief calls HTTP CONNECT P0 but also schedules it for v0.2.
This roadmap adopts v0.2 to keep the first transport release focused.
It also places aliases/`run` in v0.2 to deliver them together.

## v0.3: transfers and interactive tooling

- [ ] SFTP upload/download and port forwarding.
- [ ] OS-aware command profiles.
- [ ] Opt-in history/logs with redaction and retention policy.
- [ ] Bash/Zsh/PowerShell completion.
- [ ] Optional TUI.

## Release gates

- [ ] Strict target and jump-host key verification; changed keys rejected.
- [ ] No plaintext secrets in inventory, argv, history or diagnostics.
- [ ] Native credential integration verified on all supported platforms.
- [ ] Authenticated proxies and cancellation tested with local fixtures.
- [ ] Mixed failures produce documented exit codes and structured output.
- [ ] Destructive actions require consent and verify the resulting host state.
- [ ] Single binary works without Python, Node.js, Java or a remote agent.

## Explicit non-goals

Complex YAML playbooks, desired-state configuration, template languages, roles,
collections, a package ecosystem, remote agents, a central daemon, Web UI,
scheduling, Terraform replacement, Kubernetes management, service discovery and CMDB.
