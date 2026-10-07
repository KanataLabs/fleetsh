# fleetsh

[![CI](https://github.com/KanataLabs/fleetsh/actions/workflows/ci.yml/badge.svg)](https://github.com/KanataLabs/fleetsh/actions/workflows/ci.yml)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A lightweight, agentless VPS fleet manager for individual developers and small teams.
One Go binary for Windows, macOS and Linux, with no central server or remote agent.

[Documentation](https://kanatalabs.github.io/fleetsh/) · [中文介绍](README.zh-CN.md) · [Roadmap](docs/roadmap.md) · [Product design](docs/product.md)

> **Project status: foundation stage.** Available commands are `help`, `version` and
> `init`. SSH, credential storage and remote execution are under design and are
> not implemented yet. There is no v0.1 release.

## Build and try

Install Go 1.27 or newer, then:

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go build -o dist/fleetsh ./cmd/fleetsh
go run ./cmd/fleetsh help
go run ./cmd/fleetsh version
go run ./cmd/fleetsh init
```

On Windows, build with `go build -o dist/fleetsh.exe ./cmd/fleetsh`.
`init` creates a commented example configuration in the OS user config directory
and refuses to overwrite existing files. Override it with
`fleetsh init --config ./config.toml`.

## Planned workflow

These examples describe the target UX; they do not work in the foundation build:

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec @web "uptime" --parallel 10
fleetsh exec hk1,sg1 "df -h" --json
fleetsh update @all --dry-run
fleetsh reboot @all --parallel 2
```

Planned v0.1 features include TOML inventory, groups and tags, SSH password/key/agent
authentication, OS credential stores, ProxyJump and SOCKS5, strict host key
verification, bounded parallel execution, JSON output, update and reboot.
[See the versioned scope](docs/roadmap.md).

## Boundaries

fleetsh executes commands over SSH. It does not introduce playbooks, desired state,
a remote agent, a central daemon, a Web UI or an infrastructure automation DSL.

Passwords, private keys, sudo secrets and proxy secrets must never be stored in
inventory files or command-line arguments. See the [security design](docs/security.md).

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/fleetsh
```

CI tests Linux, macOS and Windows and cross-compiles all three platforms for amd64
and arm64. Documentation source lives in `docs/` on `main`; an Actions workflow
syncs it to `gh-pages`, which GitHub Pages builds with Jekyll.

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

## License

Copyright (C) 2026 KanataLabs contributors.
Licensed under **GNU GPL version 3 only** (`GPL-3.0-only`); see [LICENSE](LICENSE).
