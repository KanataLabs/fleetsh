---
title: Configuration
permalink: /en/configuration/
lang: en
locale: en
page_key: 'configuration/'
---

# Configuration

<a id="storage"></a>

## Where VPS configuration is stored

fleetsh stores the inventory locally on the computer running the CLI, under the
current OS user account. One `config.toml` holds host addresses, SSH users and
ports, groups, tags, proxy settings, defaults and credential reference names.
Each VPS is a `[hosts.ALIAS]` entry. Adding the binary to PATH does not change
the configuration location; commands use the same default inventory from any
working directory.

| OS | Default inventory |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\config.toml` |
| macOS | `~/Library/Application Support/fleetsh/config.toml` |
| Linux | `$XDG_CONFIG_HOME/fleetsh/config.toml` when `XDG_CONFIG_HOME` is set; otherwise `~/.config/fleetsh/config.toml` |

On Windows this is normally `C:\Users\<user>\AppData\Roaming\fleetsh\config.toml`.
`fleetsh init` creates the file and prints its path. It preserves existing files.
To locate the default inventory after initialization:

Windows PowerShell:

```powershell
$configFile = Join-Path $env:APPDATA "fleetsh\config.toml"
Write-Output $configFile
Get-Item -LiteralPath $configFile
notepad $configFile
```

macOS:

```sh
config_file="$HOME/Library/Application Support/fleetsh/config.toml"
printf '%s\n' "$config_file"
ls -l "$config_file"
```

Linux:

```sh
config_file="${XDG_CONFIG_HOME:-$HOME/.config}/fleetsh/config.toml"
printf '%s\n' "$config_file"
ls -l "$config_file"
```

These commands show the default path. If you supply `--config`, use that selected
path instead.

### Use a different inventory

Pass `--config PATH` on every command that should use a custom inventory:

```sh
fleetsh --config "./inventories/production/config.toml" init
fleetsh --config "./inventories/production/config.toml" add web1 --host web1.example.com --user ubuntu --auth agent
fleetsh --config "./inventories/production/config.toml" ls
fleetsh --config "./inventories/production/config.toml" ssh web1
```

Relative paths resolve from the current working directory; `~/` expands to the
current user's home directory. Quote paths containing spaces. The option selects
one file for that invocation; it does not save a new default or merge inventories.
Use an absolute path when invoking fleetsh from different directories. Separate
directories, such as `inventories/production/` and `inventories/staging/`, also
keep their host trust files separate.

### Related files and credentials

| Data | Location |
| --- | --- |
| VPS inventory and credential names | The selected `config.toml`, or the file passed through `--config` |
| Trusted SSH host public keys | `known_hosts` in the selected inventory's directory; created when a host key is trusted |
| SSH private keys | The path in each host's `key` field; fleetsh reads the file there |
| Saved passwords, key passphrases and proxy credentials | Windows Credential Manager, macOS Keychain or Linux Secret Service |
| File locks | `<inventory-path>.lock` and `known_hosts.lock` alongside their respective files |

fleetsh uses its adjacent `known_hosts` rather than `~/.ssh/known_hosts`. Inventories
in the same directory share that trust file. Secrets are stored under the
`fleetsh` service in the current user's OS credential store. Inventories using the
same credential reference share the same stored secret, even if their files are
in different directories. Use distinct names such as `production-web1-login`
and `staging-web1-login` when credentials should be independent.

### Back up or move to another computer

1. Finish running fleetsh commands, then copy the selected inventory and its adjacent `known_hosts` to the destination configuration directory. Lock files are coordination files and are not needed in the backup.
2. Transfer any SSH private keys separately, preserve restricted permissions and update `key` paths if necessary.
3. Re-create saved secrets on the destination with `fleetsh credential add REFERENCE`, supplying `--config PATH` if using a custom inventory. Copying TOML does not copy OS-store credentials.
4. Run `fleetsh ls` with the chosen inventory to check it before connecting. If you did not transfer `known_hosts`, verify and trust host fingerprints again.

The inventory contains infrastructure details such as host addresses and
usernames; keep backups private. See [files and permissions](#files-and-permissions).

## Inventory format

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

Inventory and host trust file locations are listed [above](#storage).
Unix files are created with mode 0600 and new directories with 0700.
Windows uses the user directory's ACL. Inventory and known_hosts must be regular
files; symlinks are rejected. Secret values remain in the OS store.
