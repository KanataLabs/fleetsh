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

`fleetsh add` defaults to port 22, connection `ssh`, and auth `password`, with automatic password saving.
Existing hosts retain their saved mode. For compatibility, TOML hosts with no `auth` field still use `agent`.
Select `--auth agent` explicitly for agent login, or `--auth key --key PATH` for a private key.
If an existing host reports `SSH agent unavailable` and you use password login,
run `fleetsh edit HOST --auth password`, then `fleetsh ssh HOST`.
SSH hosts require a username; key auth requires a key path.
Key paths resolve relative to the current directory, or expand `~/`.
Passwords and key passphrases can be prompted interactively when no reference is set.

Aliases, groups, tags and references use letters/digits/dot/underscore/hyphen,
start with a letter or digit and are at most 128 characters. Alias `all` is reserved.
Parallelism ranges from 1 to 256; durations must be positive.

<a id="groups"></a>

## Change group memberships

One host can belong to multiple groups. Groups are derived from host memberships;
there is no separate group registry to create first. You can update them after adding a host.

```sh
fleetsh edit hk1 --add-groups production,monitoring
fleetsh edit hk1 --remove-groups asia
fleetsh edit hk1 --groups web,production
fleetsh edit hk1 --groups ""
fleetsh ls '@production'
```

`--add-groups` preserves existing groups and ignores duplicate additions.
`--remove-groups` removes only the named memberships; absent names are harmless.
`--groups` replaces the entire list, and an empty value clears it.
You can combine additions and removals for different names, but cannot combine
them with `--groups` or add and remove the same name. Editing groups does not prompt
for or change the host's password.

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

<a id="passwords"></a>

## Save a host password

Adding an SSH host with `--auth password` and no `--credential` prompts twice using
hidden terminal input. fleetsh creates a unique `fleetsh-ssh-ALIAS-RANDOM` reference,
saves the secret in the OS store and associates the reference with the host.
You do not need to run `credential add` first.

```sh
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth password --groups asia,web
fleetsh edit sg1 --save-password
fleetsh show sg1
```

`edit --save-password` stores a new password under a new reference. It leaves the
previous credential intact so other hosts sharing it keep working. Switching to
password authentication without an explicit reference also prompts and saves.
Inspect references with `show` or `credential ls`; remove an unused old reference
with `credential rm REFERENCE` when you no longer need it.
Removing a host does not automatically delete its credentials.

Use `--no-save-password` to prompt at each connection instead. On edit this detaches
the host's reference and keeps the saved OS entry. For noninteractive creation,
supply an existing `--credential REFERENCE` or explicitly use `--no-save-password`.
Do not combine a nonempty reference with password-saving options.

Host settings are validated before prompting. Failed input or storage leaves the
inventory unchanged; a newly saved secret is removed if inventory persistence fails.

### Recognize fleetsh credentials

| Platform | Visible identification |
| --- | --- |
| Windows Credential Manager | Target `fleetsh:REFERENCE`; comment `Created by fleetsh (KanataLabs). Managed VPS credential.` |
| macOS Keychain | Service `fleetsh`, label `fleetsh:REFERENCE`, and the same creator comment |
| Linux Secret Service | Service attribute `fleetsh`; label starts with `fleetsh:REFERENCE` and includes the creator note |

Both automatic saving and manual `credential add` apply this identification to new
or replaced entries. Existing references and the `fleetsh` service namespace remain
compatible. Older entries gain the note when replaced.

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

<a id="proxies"></a>

## Global and per-host SSH proxies

Edit the existing `[defaults]` table to route SSH through a global proxy:

```toml
[defaults]
proxy = "socks5://127.0.0.1:1080"
# proxy_credential = "local-proxy"
```

Supported schemes are `socks5://`, `http://` and `https://`. HTTP/HTTPS uses CONNECT;
HTTPS checks the proxy certificate. Default ports are 1080, 80 and 443 respectively.
URLs must not contain userinfo, paths, queries or fragments. Save authentication
as `username:password` through `fleetsh credential add local-proxy`, then set
`proxy_credential`. No secret values belong in TOML.

A host with no explicit proxy inherits the global URL. A per-host URL overrides
it and uses only that host’s `proxy_credential`, without inheriting credentials
for a different endpoint. A host inheriting the global URL may override just the
credential reference. `proxy = "direct"` bypasses the global route.

```sh
fleetsh edit hk1 --proxy http://127.0.0.1:7890
fleetsh edit hk1 --proxy socks5://127.0.0.1:1080 --proxy-credential local-proxy
fleetsh edit hk1 --proxy direct --proxy-credential ""
fleetsh edit hk1 --proxy "" --proxy-credential ""
```

The last command restores inheritance. Proxies apply to `ssh`, `exec`, `alive`,
`stats`, `update`, `reboot` and `forward`. Settings are resolved per connection;
CLI edits preserve the distinction between inherited and explicit routes.

`proxy_jump` selects an inventory SSH gateway instead of the global proxy for
the destination. The gateway itself inherits the global proxy unless overridden.
A host cannot combine `proxy` and `proxy_jump`; jump cycles and missing gateways
are rejected. Target and gateway host keys are verified independently.

Unix agents use `SSH_AUTH_SOCK`. Windows uses `\\.\pipe\openssh-ssh-agent`
by default, or the socket/pipe in `SSH_AUTH_SOCK`. Load at least one usable key.
Run `fleetsh docs proxies` for the offline guide.

<a id="forwarding"></a>

## Three kinds of SSH port forwarding

Add profiles to an existing SSH host:

```toml
[[hosts.hk1.forwards]]
name = "web"
type = "local"
listen = "127.0.0.1:8080"
destination = "127.0.0.1:80"

[[hosts.hk1.forwards]]
name = "reverse"
type = "remote"
listen = "127.0.0.1:9000"
destination = "127.0.0.1:3000"

[[hosts.hk1.forwards]]
name = "socks"
type = "dynamic"
listen = "127.0.0.1:1080"
```

| Type | Listener | Destination connects from |
| --- | --- | --- |
| `local` | This computer | The VPS |
| `remote` | The VPS | This computer |
| `dynamic` | This computer’s SOCKS5 endpoint | The VPS; requested by the SOCKS client |

```sh
fleetsh forward hk1 --dry-run
fleetsh forward hk1 web
fleetsh forward hk1 socks
fleetsh forward hk1
fleetsh forward hk1 --dry-run --json
```

An optional name selects one profile; otherwise all profiles start together.
The dry run reads config without SSH, passwords or listeners. Actual forwarding
runs in the foreground until Ctrl+C or disconnect, then releases listeners and
active streams. Failed startup rolls back all profiles. Other commands do not
start profiles automatically. Profiles survive ordinary `edit` operations.

Names must be valid and unique per host. Listen hosts must be explicit IPs or
`localhost`, with ports 0–65535 (0 allocates a free port and prints its address).
Destination ports are 1–65535. Bracket IPv6 addresses, for example `[::1]:1080`.
`--connect-timeout` bounds connection, setup and destination dialing; existing
streams have no artificial lifetime limit. A session permits 128 active connections.

Use loopback binds for private access. Explicit `0.0.0.0`/`::` exposes listeners
to other machines. Dynamic forwarding is unauthenticated SOCKS5 TCP CONNECT;
BIND and UDP are unsupported, and domain names resolve through the VPS.
Remote listening depends on sshd `AllowTcpForwarding`/`GatewayPorts` policy;
fleetsh does not modify server settings. Run `fleetsh docs forwarding` for examples.

Custom command aliases and OpenSSH import/export remain on the [roadmap](../roadmap/).

## Files and permissions

Inventory and host trust file locations are listed [above](#storage).
Unix files are created with mode 0600 and new directories with 0700.
Windows uses the user directory's ACL. Inventory and known_hosts must be regular
files; symlinks are rejected. Secret values remain in the OS store.
