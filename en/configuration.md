---
title: Configuration
permalink: /en/configuration/
lang: en
locale: en
page_key: 'configuration/'
---

# Configuration

`fleetsh init` writes a commented example. Add hosts through CLI or edit TOML.
Strict decoding rejects unknown fields and accidental plaintext secrets.
CLI mutations use a file lock and atomic replacement and rewrite comments.

```toml
[defaults]
connect_timeout = "10s"
command_timeout = "30m"
parallel = 10

[hosts.hk1]
host = "hk1.example.com"
port = 22
user = "ubuntu"
auth = "key"
key = "~/.ssh/id_ed25519"
groups = ["asia", "web"]
tags = ["production"]

[hosts.sg1]
host = "sg1.example.com"
user = "ubuntu"
auth = "password"
credential = "sg1-login"
sudo_credential = "sg1-sudo"
groups = ["asia"]

[hosts.jump-hk]
host = "jump.example.com"
user = "ubuntu"
auth = "agent"

[hosts.private1]
host = "10.0.0.10"
user = "ubuntu"
auth = "agent"
proxy_jump = "jump-hk"

[hosts.proxy1]
host = "target.example.com"
user = "ubuntu"
auth = "agent"
proxy = "socks5://127.0.0.1:1080"
proxy_credential = "local-proxy"

[hosts.console1]
host = "console.example.com"
connection = "console-only"
```

Host defaults are port 22, connection `ssh`, auth `agent`.
SSH hosts require a username; key auth requires a key path.
Key paths resolve relative to the current directory, or expand `~/`.
Passwords and key passphrases can be prompted interactively when no reference is set.

Aliases, groups, tags and references use letters/digits/dot/underscore/hyphen,
start with a letter or digit and are at most 128 characters. Alias `all` is reserved.
Parallelism ranges from 1 to 256; durations must be positive.

## Selectors

| Selector | Meaning |
| --- | --- |
| `hk1` | One host |
| `hk1,sg1` | Explicit hosts, deduplicated and sorted |
| `all` / `@all` | Every inventory host |
| `@asia` | Hosts in group asia |
| `--tag production` | Filter the selected hosts by tag |

Unknown hosts/groups and empty remote selections are local errors.
Console-only assets appear in lists and are skipped for remote operations.

## Credential store

`credential`, `proxy_credential` and `sudo_credential` hold names only.
Use `fleetsh credential add REFERENCE`, `ls` and `rm REFERENCE`.
Replacing an existing secret requires `add --replace`.

Credential values use Windows Credential Manager, macOS Keychain or Linux Secret
Service. Linux needs a running user D-Bus session and an unlocked Secret Service
provider such as GNOME Keyring. Unavailable stores fail without a plaintext fallback.

`credential ls` lists references tracked by this inventory or used by its hosts;
it does not enumerate unrelated OS-store items. A top-level `credentials` array
records names added through this inventory. Removing an entry from the store leaves
host references intact; update those hosts as needed.

## Proxies and SSH agent

Use `proxy_jump` for an inventory jump host, or `proxy` for SOCKS5, not both.
Jump cycles and missing jump hosts are rejected. Both target and jump host keys
are checked. The target hostname is resolved by SOCKS5.
For authenticated SOCKS5, store `username:password` with `credential add` and set
`proxy_credential`; URL userinfo is forbidden.

Unix agents use `SSH_AUTH_SOCK`. Windows uses
`\\.\pipe\openssh-ssh-agent` by default, or the socket/pipe in `SSH_AUTH_SOCK`.
Agent connections require at least one loaded key.

HTTP CONNECT, custom aliases, OpenSSH import/export and forwarding remain on the
[roadmap](../roadmap/).

## Files and permissions

| OS | Default inventory |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\config.toml` |
| macOS | `~/Library/Application Support/fleetsh/config.toml` |
| Linux | `$XDG_CONFIG_HOME/fleetsh/config.toml`, or `~/.config/fleetsh/config.toml` |

The adjacent `known_hosts` file holds trusted public keys.
Unix files are created with mode 0600 and new directories with 0700.
Windows uses the user directory's ACL. Inventory and known_hosts must be regular
files; symlinks are rejected. Secret values remain in the OS store.
