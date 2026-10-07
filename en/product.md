---
title: Product brief
permalink: /en/product/
lang: en
locale: en
page_key: 'product/'
---

# Lightweight VPS fleet manager

> This is the product design brief. It describes both implemented and proposed
> features. The [roadmap](../roadmap/) defines the current implementation scope;
> the brief retains differences in the original priorities.
> Proposed commands and configuration below are design examples, not a supported
> configuration reference. Use [configuration](../configuration/) and
> [getting started](../getting-started/) for the current CLI.

## 1. Product positioning

A lightweight VPS manager for individual developers and small teams.

Core goals:

- One binary, without Python, Node.js or Java runtimes.
- No remote agent or required central server.
- Runs on Windows, macOS and Linux.
- Manages connection information and credential references for multiple VPSs.
- Supports SSH passwords, keys and agents.
- Supports proxies and jump hosts.
- Executes commands on one host or a group.
- Provides a small set of common VPS maintenance actions.
- Keeps configuration simple, portable and easy to back up.
- Uses secure defaults and never stores passwords in plaintext.

Full configuration management and replacing Ansible are outside the scope.

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec hk1 "uptime"
fleetsh exec all "uptime"
fleetsh exec '@web' "docker ps"
fleetsh update hk1
fleetsh update '@all'
fleetsh reboot hk1
```

## 2. Core design principles

### 2.1 Single binary

Distribute `fleetsh.exe` on Windows and `fleetsh` on macOS/Linux.
Running the tool does not require Python, pip, Node.js, Docker, Java or Ruby.
Existing OS capabilities such as ssh-agent, Windows Credential Manager,
macOS Keychain and Linux Secret Service may be used.
Keep core functionality inside the binary wherever practical.

### 2.2 Agentless

Targets need an SSH server, with no resident fleetsh software:

```text
fleetsh → SSH → VPS
```

Machines without SSH can remain in inventory as
`connection = "console-only"`. They support asset and credential management,
but do not receive remote commands.

## 3. Host management

Store the alias, hostname/IP, port, username, authentication type,
credential reference, proxy, groups/tags and description.

```toml
[hosts.hk1]
host = "1.2.3.4"
port = 22
user = "root"
auth = "password"
credential = "hk1-root"
groups = ["asia", "web"]

[hosts.sg1]
host = "sg1.example.com"
user = "ubuntu"
auth = "key"
key = "~/.ssh/id_ed25519"
groups = ["asia"]

[hosts.cn1]
host = "10.0.0.10"
user = "root"
auth = "password"
credential = "cn1-root"
proxy_jump = "jump-hk"
groups = ["china"]
```

Provide `add`, `edit`, `rm`, `show` and `ls`.
Support noninteractive creation:

```sh
fleetsh add hk1 --host 1.2.3.4 --user root --auth password
```

## 4. Groups and tags

Groups overlap: hk1 may belong to both asia and web, while sg1 belongs to asia
and a hosting-panel group named bt. Support explicit host lists and tag filters.

```sh
fleetsh ls '@asia'
fleetsh exec '@asia' "uptime"
fleetsh exec '@bt' "bt update"
fleetsh exec hk1,sg1,jp1 "uptime"
fleetsh exec all "docker ps" --tag web
```

Quote selectors beginning with `@` in PowerShell; the same syntax works in Bash/Zsh.

## 5. Authentication

Support private key files, encrypted private keys, ssh-agent and passwords.

```toml
auth = "key"
key = "~/.ssh/id_ed25519"
```

Agent authentication uses `auth = "agent"`. Password authentication uses
`auth = "password"` and a reference such as `credential = "hk1-root"`.
Never put a `password` value in configuration.

## 6. Credential manager

Use Windows Credential Manager, macOS Keychain and Linux Secret Service
(for example, GNOME Keyring or a compatible KWallet provider).

The original design considered an encrypted local database fallback using
AES-256-GCM and a user-provided or securely derived master key.
That fallback is not implemented; the current version fails when the OS store
is unavailable, without a plaintext fallback.

```sh
fleetsh credential add hk1-root
fleetsh credential ls
fleetsh credential rm hk1-root
```

Password entry and confirmation must be hidden and must not enter shell history.

## 7. Proxies

The original brief called proxy support P0, including three types.

### Jump hosts

```text
Local → jump-hk → cn1
```

Define a jump host in inventory, then use `proxy_jump = "jump-hk"` on the target.
`fleetsh ssh cn1` follows the configured path automatically.

### SOCKS5

Use a URL such as `socks5://127.0.0.1:7890`, including local Clash or sing-box
proxies. Support both unauthenticated and username/password SOCKS5.
The current implementation stores authenticated proxy values through
`proxy_credential`, without URL userinfo.

### HTTP CONNECT

Proposed URLs such as `http://127.0.0.1:8080` establish a CONNECT tunnel.
HTTP CONNECT is scheduled for v0.2.

## 8. SSH

`fleetsh ssh hk1` should behave like logging in with a normal SSH client.
Support PTY, interactive shell, password/public-key/agent authentication,
custom ports, host-key verification, keepalive and timeout.

The original brief proposed forwarding extra OpenSSH arguments, such as
`fleetsh ssh hk1 -- -L 8080:localhost:80`. Argument passthrough and forwarding
are deferred; core SSH runs in process.

## 9. Remote execution

```sh
fleetsh exec hk1 "uptime"
fleetsh exec '@all' "uptime"
```

Default to parallel execution. Show each host's result and elapsed time,
followed by its stdout/stderr. Keep host output distinguishable.

## 10. Concurrency control

```sh
fleetsh exec '@all' "uptime" --parallel 10
fleetsh exec '@all' "uptime" --serial
```

Default parallelism is 10. Serial execution sets concurrency to 1.
Concurrency control matters especially for updates and reboots.

## 11. Exit codes and errors

Preserve stdout, stderr, remote exit status, connection failures, timeouts,
authentication failures and proxy failures. Report success, failure and skip counts.

The current exit codes are:

- 0: all executable targets succeeded.
- 1: remote command failed, timed out or was canceled.
- 2: local usage, configuration or credential preparation error.
- 3: connection, authentication, proxy or host-key failure.

Code 1 takes precedence over 3 in mixed remote failures.
See [architecture](../architecture/) for the detailed contract.

## 12. sudo

```sh
fleetsh exec hk1 "apt-get update" --sudo
```

Use a credential reference such as `sudo_credential = "hk1-root"` when a password
is required. Transmit the password over channel stdin, without exposing it in
logs or process arguments through constructions such as `echo password | sudo -S`.

## 13. Built-in actions

Start with a few frequent actions, without an Ansible-style YAML system.

### status (planned)

`fleetsh status '@all'` would report SSH reachability, uptime, load, disk,
memory, OS and kernel information.

### update

Detect the distribution: Debian/Ubuntu use apt, RHEL/Rocky/AlmaLinux use dnf,
older CentOS uses yum, and Arch uses pacman.
For Debian/Ubuntu, the default is package-list refresh followed by upgrade.
`--dist` selects a full distribution upgrade.

### reboot-required (planned)

`fleetsh reboot-required '@all'` would check whether a restart is needed,
including `/var/run/reboot-required` on Debian/Ubuntu.

### reboot

Send reboot, tolerate the expected SSH disconnect, wait for reconnection,
then run a health probe. The implementation also requires a changed Linux
boot ID before declaring success. Limit batch concurrency by default:

```sh
fleetsh reboot '@all' --parallel 2
```

## 14. Custom commands and aliases (planned)

Store simple reusable commands in TOML:

```toml
[commands.bt-update]
command = "bt update"

[commands.docker-update]
command = "docker compose pull && docker compose up -d"

[commands.disk]
command = "df -h"
```

Proposed usage: `fleetsh run hk1 bt-update` and `fleetsh run '@bt' bt-update`.
This feature is scheduled for v0.2.

## 15. Command profiles (planned)

Select different commands by detected OS:

```toml
[commands.patch]
debian = "apt-get update && apt-get upgrade -y"
ubuntu = "apt-get update && apt-get upgrade -y"
rocky = "dnf upgrade -y"
```

Proposed usage: `fleetsh run '@all' patch`. OS-aware profiles are a later
priority, scheduled for v0.3.

## 16. Host-key security

Strict verification is enabled by default.
First connections show a SHA256 fingerprint and require explicit trust.
Changed fingerprints reject the connection.

```sh
fleetsh hostkey show hk1
fleetsh hostkey reset hk1
```

Do not disable host-key verification by default.

## 17. Timeouts and retries

```toml
[defaults]
connect_timeout = "10s"
command_timeout = "30m"
parallel = 10
```

Override command deadlines with `--timeout 60s`.
The original design also proposed `retries = 1`; retries are deferred.
The current version rejects that unsupported field and never retries commands automatically.

## 18. Output modes

Default to human-readable output; provide `--json` for scripts:

```sh
fleetsh exec '@all' "uptime" --json
```

The original JSON example was illustrative. Current JSON uses `results`
and `summary`, including per-host stdout/stderr, exit status, timing, errors,
truncation and skipping. JSON Lines and quiet mode remain future options.

## 19. Dry run

`fleetsh update '@all' --dry-run` shows the intended package update commands.
It connects only to identify the OS and performs no package updates.
Reboot dry runs do not connect. Dry run is chiefly for built-in actions;
arbitrary shell commands cannot be meaningfully simulated.

## 20. Confirmation

Updates and reboots list the selected aliases and require confirmation.
Use `--yes` for explicit approval in automation.

## 21. Configuration locations

| OS | Directory |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\` |
| Linux | `~/.config/fleetsh/` (or XDG configuration directory) |
| macOS | `~/Library/Application Support/fleetsh/` |

Current files are `config.toml` and adjacent `known_hosts`.
The original design also considered a separate `hosts.toml`.
Secret values do not belong in these files.

## 22. OpenSSH configuration import (planned)

Proposed `fleetsh import ~/.ssh/config` would convert Host, HostName, User,
Port, IdentityFile and ProxyJump entries into inventory hosts.
The original brief also considered `use_openssh_config = true`.
Neither interface is implemented yet. Import reduces migration effort.

## 23. Export (planned)

Provide ordinary TOML/JSON export and an OpenSSH config format, with proposed
commands `fleetsh export` and `fleetsh export ssh-config`.
Keep host information portable and avoid data lock-in.

## 24. Secret redaction

Hide known passwords, proxy passwords and sudo passwords from diagnostics.
The design also calls for protecting private keys and tokens wherever handled.
Current buffered execution output redacts resolved secrets; arbitrary application
secrets cannot be recognized. Interactive terminal output passes through unchanged.

## 25. Non-goals for the first version

Complex YAML playbooks, desired-state configuration, template languages/Jinja,
roles, collections, package ecosystems, remote agents, central daemons, Web UI,
scheduling, Terraform replacement, Kubernetes management, service discovery and CMDB.

Keep the main workflow focused on connecting to hosts and running commands.

## 26. Recommended MVP

Inventory add/edit/rm/ls/show, SSH, remote execution, groups/tags,
password/key/agent authentication, OS credential stores, ProxyJump/SOCKS5,
parallel execution, deadlines, strict host keys, JSON, update and reboot.

The original MVP list included `run`; the roadmap places it in v0.2.

## 27. v0.2

HTTP CONNECT, reboot-required, status, custom aliases, OpenSSH import,
configuration export, connection retries and rolling execution.
sudo credential references and serial execution were proposed here but are
already implemented with the core executor.

## 28. v0.3

File upload/download through SFTP, port forwarding, OS-aware command profiles,
optional history and logs, PowerShell/Bash/Zsh completion and an optional TUI.

## 29. Recommended technology

Go fits the project's I/O, SSH, CLI, configuration, credential-store, proxy
and concurrency requirements. Use Cobra or urfave/cli, x/crypto/ssh,
x/net/proxy, TOML, an OS keyring abstraction and goroutines.

```text
CLI
├── Inventory
├── Credential Store
├── SSH Connection Builder
│   ├── Direct
│   ├── ProxyJump
│   ├── SOCKS5
│   └── HTTP CONNECT (planned)
├── Executor: single / parallel
├── Actions: exec / update / reboot / status (planned)
└── Output: terminal / JSON
```

## 30. Intended user experience

```sh
# List hosts
fleetsh ls
# Log in
fleetsh ssh hk1
# Execute on one host
fleetsh exec hk1 "uptime"
# Execute on every host
fleetsh exec '@all' "df -h"
# Update hosting-panel software
fleetsh exec '@bt' "bt update"
# Update the OS
fleetsh update '@all'
# Check restart requirements (planned)
fleetsh reboot-required '@all'
# Limit simultaneous reboots
fleetsh reboot '@all' --parallel 2
# Use the configured SOCKS proxy
fleetsh ssh special1
# Structured status output (planned)
fleetsh status '@all' --json
```

Users should need to understand their hosts and the commands they want to run,
without learning an inventory DSL, playbooks, roles, collections, modules,
facts, templates or a Python environment.
