---
title: Overview
permalink: /
---

# Your VPS fleet, one shell away.

fleetsh is a lightweight VPS manager built in Go for individual developers and
small teams. One binary, agentless SSH and operating-system credential stores on
Windows, macOS and Linux.

> **Development build.** Core inventory, SSH, credentials, proxies, parallel exec,
> update and reboot are implemented and tested with local fixtures.
> There is no stable release yet.

[Install and register PATH](installation/) · [Get started](getting-started/) ·
[Read the product brief](product/) · [Roadmap](roadmap/)

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec '@web' "uptime" --parallel 10
fleetsh update '@all' --dry-run --sudo
fleetsh reboot '@all' --parallel 2 --sudo
```

## Principles

- A single binary, without a language runtime or remote agent.
- No central server or infrastructure automation DSL.
- Portable TOML inventory, with no plaintext passwords.
- Strict host key verification, including jump hosts.
- Bounded concurrency, deadlines and structured per-host results.
- Explicit consent for updates/reboots and boot identity verification.

## Open development

Maintained at [KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh), under
GPL-3.0-only. Read the [architecture](architecture/) and [security design](security/).
