---
title: Daily operations
permalink: /en/operations/
lang: en
locale: en
page_key: 'operations/'
---

# Daily operations

Use these commands to edit hosts, diagnose connections and run everyday tasks.
CLI help is in English. If you use a custom inventory, pass the same
`--config PATH` on every command.

<a id="interactive"></a>

## Add and edit interactively

In a terminal, start with `npx fleetsh add hk1`. Missing address, username and key
settings are requested before hidden password input. New hosts default to SSH
password authentication. You can also supply every setting explicitly.

`npx fleetsh edit hk1` opens a menu for address, username, saved password, groups,
tags, description or authentication. Canceling saves nothing. Group and tag input
replaces the list; empty input clears it. Scripted calls and `--json` need explicit
change flags and never open a menu.

```sh
npx fleetsh edit hk1
npx fleetsh edit hk1 --save-password
npx fleetsh edit hk1 --add-groups production
npx fleetsh show hk1
```

Password replacement updates the local saved copy, not the server password.
Successful changes identify the updated host.

<a id="diagnostics"></a>

## Diagnose a login

```sh
npx fleetsh doctor hk1
npx fleetsh doctor hk1 --connect-timeout 30s --json
```

The report shows the inventory location, actual proxy/jump route, local login
preparation, network connection, verified host key, SSH authentication and a small
remote health check. Diagnosis never edits configuration or trusts new keys.
For an unknown key, verify its fingerprint independently and connect with
`fleetsh ssh hk1` to accept it explicitly.

Human-readable failures explain the problem and suggest a next command.
JSON retains machine-readable checks and existing exit-code meanings.
For a wrong saved password, use `fleetsh edit hk1 --save-password`.

<a id="monitoring"></a>

## Watch resource usage

```sh
npx fleetsh stats '@all' --table
npx fleetsh stats '@all' --watch 5s
npx fleetsh stats '@web' --watch 5s --json
```

The table shows host status, CPU, memory, swap, current user and free space on
the root filesystem. Each snapshot has a timestamp. Other filesystem details
remain available in ordinary `stats` output. A failed host does not hide others.

Refresh intervals must be at least one second and start after the previous
snapshot finishes, so probes do not overlap. Ctrl+C stops monitoring.
JSON watch output is one compact report per line, with `timestamp`, `results`
and `summary`. Host settings and credentials are loaded at startup; restart the
watch command after editing them. No remote agent is installed.

<a id="saved-commands"></a>

## Save common commands

Add sections to your inventory:

```toml
[commands.disk]
command = "df -h"
timeout = "30s"

[commands.docker-update]
command = "docker compose pull && docker compose up -d"
sudo = true
timeout = "10m"
```

```sh
fleetsh run '@web' docker-update --dry-run
fleetsh run '@web' docker-update --parallel 2
fleetsh run hk1 disk --json
```

Both selector and command name are required. Explicit `--sudo` or `--timeout`
override saved defaults. Dry runs make no connection. Commands are not retried
automatically. Do not put passwords or tokens in command definitions.

<a id="completion"></a>

## Enable shell completion

Install the native binary or `npm install -g fleetsh` so `fleetsh` is on PATH.
Generate a script for `bash`, `zsh`, `fish` or `powershell`.

```powershell
fleetsh completion powershell > fleetsh-completion.ps1
. ./fleetsh-completion.ps1
```

Add the dot-source line with an absolute path to your PowerShell profile for
future sessions. For Bash, source the generated Bash script. For Zsh, put
`_fleetsh` in a directory on `fpath` and initialize `compinit`. For Fish, place
the generated script in `~/.config/fish/completions/fleetsh.fish`.

Completion reads aliases, groups and saved command names locally. It never
reads secrets or connects to a server. Completion for `npx fleetsh` depends
on your shell; the supported completion entry point is the `fleetsh` command.

<a id="maintenance"></a>

## Check restart requirements and reboot in batches

```sh
fleetsh update '@all' --sudo --check-reboot
fleetsh reboot-required '@all'
fleetsh reboot '@web' --sudo --batch-size 2 --wait-timeout 5m
```

Debian/Ubuntu use the reboot-required marker. Other distributions can use
`needs-restarting -r` when installed. Unsupported checks explicitly report
**unknown**. Updates never reboot machines automatically.

A batch must finish its new-boot-ID and health checks before the next starts.
A failed batch stops later reboots; unstarted hosts are reported as skipped.
Confirmation is still required unless you supply `--yes`.
Batch size and simultaneous worker count are independent.

<a id="import-export"></a>

## Import OpenSSH settings and export host data

```sh
fleetsh import ~/.ssh/config --dry-run
fleetsh import ~/.ssh/config --user ubuntu
fleetsh export --format toml --output inventory.toml
fleetsh export --format json
fleetsh export --format ssh-config --output ssh-config
```

Import supports literal aliases, wildcard defaults, `HostName`, `User`, `Port`,
one `IdentityFile` and one `ProxyJump` alias, using OpenSSH's first-value-wins
ordering. A missing username needs `--user USER`. Duplicate aliases stop import;
existing hosts are never overwritten.

`Include` and `Match` stop import because their full semantics are unsupported.
Hosts relying on `ProxyCommand`, multi-hop/address-style ProxyJump or unsupported
expansion tokens are skipped with notes. Other unsupported settings are listed.
No local command is executed. Imported hosts use key or agent authentication;
use `edit HOST` to choose password authentication.

Exports omit saved OS credential values, credential references and saved command
definitions. User-entered descriptions and connection settings are retained.
OpenSSH output reports proxy URLs and forwarding profiles it cannot represent.
An output file is written as UTF-8; existing files require `--overwrite`.

<a id="transfers"></a>

## Upload and download with SFTP

```sh
fleetsh upload '@web' ./script.sh /tmp/script.sh --dry-run
fleetsh upload '@web' ./script.sh /tmp/script.sh
fleetsh download hk1 /var/log/syslog ./syslog.txt
fleetsh download '@web' /var/log/syslog ./logs
```

Transfers use existing SSH login, proxy and host-key settings. Select one regular
file; recursive directories and `--sudo` are not supported. Upload destination
directories must exist and be writable by the SSH user.

Single-host downloads accept a local filename or an existing directory.
Group downloads create a subdirectory per alias, with a short hash to avoid
filesystem name collisions. Dry runs show exact paths and create no files.
Existing destination files require an explicit `--overwrite`.
Interrupted transfers may leave a partial destination; review it before retrying.
