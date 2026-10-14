---
title: About fleetsh
permalink: /en/product/
lang: en
locale: en
page_key: 'product/'
---

# About fleetsh

fleetsh keeps your VPS connection settings in one local inventory. Use it to
connect over SSH, run a command on several machines, check resource usage,
update packages or transfer a file.

It runs on Windows, macOS and Linux as a Go binary. The managed machines need
SSH; they do not need fleetsh installed. You can also launch it through npm.

## Common tasks

| Task | Command |
| --- | --- |
| Add a machine | `fleetsh add hk1` |
| Change its settings or saved password | `fleetsh edit hk1` |
| Diagnose a failed login | `fleetsh doctor hk1` |
| Run a command on a group | `fleetsh exec '@web' "uptime"` |
| Watch resource usage | `fleetsh stats '@all' --watch 5s` |
| Run a saved command | `fleetsh run '@web' docker-update` |
| Check restart requirements | `fleetsh reboot-required '@all'` |
| Upload a file | `fleetsh upload hk1 ./script.sh /tmp/script.sh` |

`add` and `edit` use prompts in a terminal. For scripts, supply the required
flags. A host can belong to several groups. Group selectors start with `@`;
quote them in PowerShell.

See [getting started](../getting-started/) for your first host and
[daily operations](../operations/) for the other commands.

## Configuration and credentials

Connection settings are stored in a local TOML file. Passwords and passphrases
are stored in the operating system's credential store. The inventory contains
only their references.

SSH connections support keys, passwords, an SSH agent, proxies and jump hosts.
Local, remote and dynamic port forwards are configured by name.

See [configuration](../configuration/) for paths and examples, and
[security](../security/) for host-key verification and credential handling.

## Current limits

SFTP transfers handle one regular file at a time. OpenSSH import supports the
fields listed in its help and reports unsupported settings. Monitoring reads
Linux resource information. Commands are not retried automatically.

The [roadmap](../roadmap/) lists remaining work. The project is maintained in
[KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh) under GPL-3.0-only.
