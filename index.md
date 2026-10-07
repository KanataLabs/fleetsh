---
title: Overview
permalink: /
---

# Your VPS fleet, one shell away.

fleetsh is a lightweight VPS manager built in Go for individual developers and small
teams. The goal is one binary, agentless SSH connections and secure credentials on
Windows, macOS and Linux.

> **Foundation stage.** Only `help`, `version` and `init` are implemented.
> The remote management examples below describe planned behavior.

## Start here

[Build the current CLI](getting-started/) · [Read the product brief](product/) ·
[Follow the roadmap](roadmap/)

## Target experience

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec @web "uptime" --parallel 10
fleetsh update @all --dry-run
fleetsh reboot @all --parallel 2
```

## Principles

- A single binary, with no Python, Node.js or Java runtime.
- No remote agent and no central server.
- Ordinary TOML inventory that users can back up and migrate.
- Secrets held by the operating system's credential store.
- Strict host key verification, explicit consent for disruptive actions.
- Direct command execution, without a configuration management DSL.

## Open development

The project is maintained at [KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh)
and licensed under GPL-3.0-only. The [architecture](architecture/) and
[security requirements](security/) guide the first implementation.
