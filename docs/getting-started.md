---
title: Getting started
permalink: /getting-started/
---

# Getting started

The foundation build is for contributors. No release binary is available yet.

## Build

Install Go 1.27 or newer and Git:

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go build -o dist/fleetsh ./cmd/fleetsh
```

On Windows, use `go build -o dist/fleetsh.exe ./cmd/fleetsh`.
Add the binary's directory to your PATH, or use `go run ./cmd/fleetsh`.

## Available commands

```sh
fleetsh help
fleetsh version
fleetsh init
fleetsh init --config ./config.toml
```

`init` writes a commented TOML example and refuses to overwrite any existing file.
It does not connect to servers, load credentials or validate the draft inventory schema.
Unix creates new directories with mode 0700 and the file with mode 0600.
Windows relies on the user directory's ACL.

## Configuration location

| OS | Default path |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\config.toml` |
| macOS | `~/Library/Application Support/fleetsh/config.toml` |
| Linux | `$XDG_CONFIG_HOME/fleetsh/config.toml`, or `~/.config/fleetsh/config.toml` |

Future SSH and execution commands are described in the [roadmap](../roadmap/).
