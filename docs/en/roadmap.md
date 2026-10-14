---
title: Roadmap
permalink: /en/roadmap/
lang: en
locale: en
page_key: 'roadmap/'
---

# Roadmap

The current release is **v0.1.0-beta.1**.
[Release notes](https://github.com/KanataLabs/fleetsh/releases/tag/v0.1.0-beta.1)

## Available now

- Host inventories, overlapping groups, tags and terminal-guided add/edit.
- SSH with passwords, keys, an agent, proxies or jump hosts; saved passwords use the OS credential store.
- Connection diagnosis, liveness checks and Linux resource snapshots with table/watch output.
- Batch commands, saved commands, package updates, restart checks and reboot batches with health verification.
- Local, remote and dynamic port forwards; single-file SFTP upload/download.
- Restricted OpenSSH import, TOML/JSON/OpenSSH export and shell completion.

See [daily operations](../operations/) for examples and limits.

## Next

- More coverage of real terminal/PTY behavior and long-running SSH sessions.
- OS-aware saved commands.
- Optional command history with redaction and retention settings.
- Connection retry rules that cannot repeat a remote command unintentionally.

## Release checks

Before a release, run the Go unit and local SSH/SFTP tests, `go vet` and the npm
launcher tests. CI also checks race conditions, native credential stores and
builds on Windows, macOS and Linux. Release packages contain their licenses and
SHA-256 checksums.

The current test fixtures do not modify your inventory or contact your VPSs.
