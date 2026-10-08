# fleetsh

[![CI](https://github.com/KanataLabs/fleetsh/actions/workflows/ci.yml/badge.svg)](https://github.com/KanataLabs/fleetsh/actions/workflows/ci.yml)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A lightweight, agentless VPS fleet manager for individual developers and small teams.
One Go binary for Windows, macOS and Linux, without a central server or remote agent.

[Documentation](https://kanatalabs.github.io/fleetsh/en/) · [Install / PATH](https://kanatalabs.github.io/fleetsh/en/installation/) · [中文](README.zh-CN.md) · [Roadmap](docs/en/roadmap.md)

> **Development build.** Core inventory, SSH, credentials, proxies, parallel exec,
> update and reboot are implemented. There is no stable release yet.

## Install

Build with Go 1.27 or newer:

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go build -trimpath -o dist/fleetsh ./cmd/fleetsh
```

Windows: `go build -trimpath -o dist/fleetsh.exe ./cmd/fleetsh`.
Or use `go install github.com/KanataLabs/fleetsh/cmd/fleetsh@latest`.
See [installation and global PATH registration](docs/en/installation.md) for
Windows user/system PATH and macOS/Linux user/system installation.

## Use

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups asia,web
fleetsh ssh hk1 --connect-timeout 60s
fleetsh exec '@web' "uptime" --parallel 10
fleetsh exec hk1 "df -h" --json
fleetsh update '@all' --dry-run --sudo
fleetsh reboot '@all' --parallel 2 --sudo
```

Verify the first-connection fingerprint independently before trusting it.
Batch automation rejects unknown/changed keys. Updates/reboots require confirmation;
`--yes` is the explicit automation override. A reboot is verified by a new boot ID
and health probe. Built-in update/reboot actions target Linux VPSs.

Commands: `init`, `version`, `add`, `edit`, `rm`, `ls`, `show`, `ssh`,
`exec`, `update`, `reboot`, `credential add/ls/rm`, `hostkey show/reset`.
All commands accept `--config PATH`. Run `fleetsh COMMAND --help` for flags.

VPS configuration is local to your OS user, in `%APPDATA%\fleetsh\config.toml`
(Windows), `~/Library/Application Support/fleetsh/config.toml` (macOS), or
`$XDG_CONFIG_HOME/fleetsh/config.toml` / `~/.config/fleetsh/config.toml` (Linux).
See [storage locations and backup](docs/en/configuration.md#storage) for custom
inventories, host trust files and migrating credentials.

## Use cases

[Patch every VPS](docs/en/cases/patch-all-vps.md): preview and apply Debian/Ubuntu
package updates across the fleet, inspect results and reboot selected hosts.

## Credentials and proxies

Passwords/passphrases use hidden input or Windows Credential Manager, macOS Keychain
and Linux Secret Service. Inventory holds only credential references.
Authenticated SOCKS5 stores `username:password` by reference; URL secrets are rejected.
SSH keys, agent sockets/pipes and ProxyJump work in process without OpenSSH.
Unavailable credential stores fail without a plaintext fallback.

See [configuration](docs/en/configuration.md), [usage](docs/en/getting-started.md) and
[security](docs/en/security.md). No playbooks, desired-state DSL, remote daemon or Web UI.

## Development

```sh
go test -race ./...
go vet ./...
go build ./cmd/fleetsh
```

Tests use localhost SSH/proxy fixtures, never real VPS credentials.
CI tests three OSs and six CGO-free build targets. Native keyring tests use
`FLEETSH_TEST_KEYRING=1` and temporary fixture entries in unlocked stores.
Documentation source is `docs/` on main. CI builds both languages; the organization App
publishes to gh-pages through configured CI credentials or scripts/publish-docs.ps1.
Read [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Copyright (C) 2026 KanataLabs contributors.
**GPL-3.0-only**; see [LICENSE](LICENSE) and [third-party notices](THIRD_PARTY_NOTICES.md).
