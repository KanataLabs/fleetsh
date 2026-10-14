---
title: Overview
permalink: /en/
lang: en
locale: en
page_key: ''
---

# Manage your VPS hosts

fleetsh stores VPS connection settings locally and runs commands over SSH.
It works on Windows, macOS and Linux. Saved passwords use the operating system's
credential store.

> Current release: **v0.1.0-beta.1**. [Release notes](https://github.com/KanataLabs/fleetsh/releases/tag/v0.1.0-beta.1).

[Install and register PATH](installation/) · [Get started](getting-started/) ·
[About fleetsh](product/) · [Roadmap](roadmap/)

[Use cases: patch every VPS](cases/patch-all-vps/)

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec '@web' "uptime" --parallel 10
fleetsh update '@all' --dry-run --sudo
fleetsh reboot '@all' --parallel 2 --sudo
```

With Node.js 22+, run through [npm](https://www.npmjs.com/package/fleetsh):

```sh
npx fleetsh version
npx fleetsh stats
npm install -g fleetsh
```

The launcher follows GitHub Releases automatically. Use `--release bundled`
for the original binary or `--release vVERSION` for a fixed Go version.
See [npx/npm usage and version selection](installation/).

## Open development

Maintained at [KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh), under
GPL-3.0-only. Read the [architecture](architecture/) and [security design](security/).
