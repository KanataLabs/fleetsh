---
title: Configuration
permalink: /configuration/
---

# Configuration

The TOML schema is a design draft. The current CLI can create an example file but
does not parse inventory or establish SSH connections.

```toml
[defaults]
connect_timeout = "10s"
command_timeout = "30m"
parallel = 10

[hosts.hk1]
host = "hk1.example.com"
port = 22
user = "ubuntu"
connection = "ssh"
auth = "key"
key = "~/.ssh/id_ed25519"
groups = ["asia", "web"]
tags = ["production"]

[hosts.sg1]
host = "sg1.example.com"
user = "ubuntu"
auth = "password"
credential = "sg1-login"
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
```

## Selectors (planned)

| Selector | Meaning |
| --- | --- |
| `hk1` | One host alias |
| `hk1,sg1` | Explicit aliases, deduplicated |
| `all` or `@all` | All inventory hosts |
| `@asia` | Hosts in the asia group |
| `--tag production` | Hosts with a matching tag |

Unknown hosts or groups must be local configuration errors. Console-only assets
remain visible in inventory and are reported as skipped for remote execution.

## Credentials and proxies (planned)

`credential`, `proxy_credential` and `sudo_credential` reference secret store entries;
they never contain secret values. Private keys remain in their original files.

Use `proxy_jump = "jump-hk"` for a jump host, or
`proxy = "socks5://127.0.0.1:1080"` for SOCKS5.
Authenticated proxy secrets must use a credential reference, never URL userinfo.
HTTP CONNECT is scheduled for v0.2.

Command aliases are scheduled for v0.2:

```toml
[commands.disk]
command = "df -h"
```

The planned invocation is `fleetsh run @web disk`.
