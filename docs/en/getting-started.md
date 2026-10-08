---
title: Getting started
permalink: /en/getting-started/
lang: en
locale: en
page_key: 'getting-started/'
---

# Getting started

fleetsh now implements the core v0.1 workflow. It is a development build, not a
stable release. See [installation and PATH registration](../installation/) to make
the command available from any directory.

## Initialize an inventory

The inventory is stored locally for your OS user: `%APPDATA%\fleetsh\config.toml`
on Windows, `~/Library/Application Support/fleetsh/config.toml` on macOS, and
`$XDG_CONFIG_HOME/fleetsh/config.toml` or `~/.config/fleetsh/config.toml` on Linux.
See [storage locations, custom inventories and backup](../configuration/#storage).

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups asia,web
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth password --groups asia
fleetsh ls
fleetsh ls '@asia' --json
fleetsh show hk1
fleetsh edit hk1 --port 2222
```

The password prompt is hidden. Credential values are stored in the OS credential
store; inventory stores only reference names. Key passphrases can use the same
`--credential` field; without a reference they are prompted privately.

Adding a password-authenticated host now prompts twice and saves the password automatically.
The generated reference has a `fleetsh-ssh-` prefix; the inventory holds only its name.
Use `fleetsh edit sg1 --save-password` to update the password without managing references.
See [password storage](../configuration/#passwords) and [changing groups](../configuration/#groups).

Every command accepts `--config PATH`. Paths beginning with `~/` are expanded.
The inventory file is validated strictly; unknown fields and plaintext
`password` fields are rejected. `edit` rewrites TOML and does not preserve comments.

## Establish host identity

```sh
fleetsh ssh hk1 --connect-timeout 60s
fleetsh hostkey show hk1
```

First connections display a SHA256 fingerprint and require an interactive `y`.
Verify it through an independent source before trusting. This prompt shares the
connection deadline; allow time with `--connect-timeout`.
Changed host keys are always rejected. Unattended commands reject unknown keys.

After independently verifying a legitimate key rotation:

```sh
fleetsh hostkey reset hk1
fleetsh ssh hk1 --connect-timeout 60s
```

The dedicated `known_hosts` file sits next to the selected inventory file.
SSH interactive sessions use PTY/raw terminal mode when stdin is a terminal,
and forward terminal size changes. OpenSSH argument passthrough/forwarding is deferred.

## Execute

Quote group selectors in PowerShell: "fleetsh ls '@asia'". The quotes prevent
PowerShell from interpreting the @ prefix as variable splatting. Quoted selectors
also work in Bash and Zsh.

```sh
fleetsh exec hk1 "uptime"
fleetsh exec '@asia' "df -h" --parallel 10
fleetsh exec hk1,sg1 "uptime" --serial --json
fleetsh exec all "docker ps" --tag web --timeout 60s
fleetsh exec hk1 "apt-get update" --sudo
```

`--sudo` uses passwordless `sudo -n` unless the host has `sudo_credential`.
A referenced sudo password is sent over channel stdin, never inserted in the command.
Commands receive no interactive stdin; use `ssh` for interactive programs.

Output is grouped per host and bounded to 4 MiB per stdout/stderr stream.
JSON contains `results` and `summary`, with host, exit status, timing, error category,
truncation and skipping fields. Console-only assets are explicitly skipped.

Exit codes: 0 all executable targets succeeded; 1 a command failed or execution was
canceled/timed out; 2 local usage/configuration/credential preparation error;
3 connection/authentication/proxy/host-key failure, including connection timeout.
A command failure takes precedence in mixed remote failures. Interactive shell
nonzero remote exit status returns 1; transport failure returns 3.

## Update and reboot Linux VPSs

```sh
fleetsh update '@asia' --dry-run --sudo
fleetsh update '@asia' --sudo
fleetsh update hk1 --dist --sudo --yes
fleetsh reboot '@asia' --dry-run --sudo
fleetsh reboot '@asia' --parallel 2 --sudo --wait-timeout 5m
```

Update dry runs connect to identify the OS but never dispatch package updates.
Supported IDs: Debian, Ubuntu, RHEL, Rocky, AlmaLinux, Fedora, CentOS, Arch and Manjaro.
An unsupported OS or failed preflight prevents updates on every target.
Reboot dry runs do not connect.

Actual updates and reboots show selected aliases and require confirmation; use
`--yes` for explicit automation approval. Reboot concurrency defaults to 2.
A reboot succeeds only after a changed Linux boot ID and successful uptime probe.
No commands are automatically retried.

For a complete maintenance workflow, see [Patch every VPS](../cases/patch-all-vps/).
