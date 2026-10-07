---
title: Roadmap
permalink: /en/roadmap/
lang: en
locale: en
page_key: 'roadmap/'
---

# Roadmap

## Current: v0.1 development implementation

- [x] Public Go repository in KanataLabs, GPL-3.0-only.
- [x] Inventory add/edit/rm/ls/show, groups/tags, console-only assets.
- [x] Keys (including encrypted keys), passwords and SSH agent.
- [x] Native OS credential adapters and private terminal input.
- [x] Direct SSH, ProxyJump and authenticated SOCKS5.
- [x] Interactive PTY/shell, host key trust/show/reset, keepalive and deadlines.
- [x] Single/batch exec, worker bounds, per-host results, JSON and output limits.
- [x] Linux update with OS detection, dry run and confirmation.
- [x] Reboot with reconnect, new boot ID and health probe.
- [x] Channel-based sudo credential input and serial execution.
- [x] Cross-platform CI, release packaging workflow, main-to-gh-pages publishing.
- [x] Windows/macOS/Linux user/system PATH installation instructions.
- [ ] First tagged release binaries and checksums.
- [ ] Manual real-VPS acceptance before declaring a stable release.

Implemented features are tested with local fixtures. This is a development build;
no stable version has been published.

## v0.2: convenience and interoperability

- [ ] HTTP CONNECT proxy.
- [ ] status and reboot-required.
- [ ] Custom command aliases and `run`.
- [ ] OpenSSH config import and TOML/JSON/OpenSSH export.
- [ ] Connection retry and rolling execution.

The initial brief placed HTTP CONNECT in both P0 and v0.2; the roadmap uses v0.2.
Aliases/`run` also remain v0.2. sudo credential references and serial execution
were implemented alongside the core executor.

## v0.3: transfers and interactive tooling

- [ ] SFTP upload/download and port forwarding.
- [ ] OS-aware command profiles.
- [ ] Opt-in history/logs with redaction and retention policy.
- [ ] Bash/Zsh/PowerShell completion.
- [ ] Optional TUI.

## Release gates

- [x] Target/jump key verification, unknown-key refusal and changed-key rejection.
- [x] Plaintext-secret field rejection and channel-based sudo.
- [x] Authenticated proxies and cancellation tested with local fixtures.
- [x] Worker limits, output truncation and mixed-failure exit codes tested.
- [x] Update confirmation/preflight and reboot identity verification tested.
- [x] CGO-free single binaries across OS/architectures.
- [x] Native credential integration CI passed on all supported platforms.
- [ ] Manual terminal/PTY acceptance and real-VPS action verification.
- [ ] Reviewed, tagged release.

## Non-goals

Complex YAML playbooks, desired-state configuration, template languages, roles,
collections, remote agents, a central daemon, Web UI, scheduling, Terraform
replacement, Kubernetes management, service discovery and CMDB.
