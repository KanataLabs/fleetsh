---
title: Patch every VPS
description: Run apt update and apt upgrade across a VPS fleet with controlled concurrency and per-host results.
permalink: /en/cases/patch-all-vps/
lang: en
locale: en
page_key: 'cases/patch-all-vps/'
---

# Patch every VPS

You have several Debian or Ubuntu VPSs and want to run the equivalent of
`apt update && apt upgrade` on all of them. fleetsh lets you select the fleet,
preview updates, limit simultaneous operations and inspect each host's result.

## 1. Prepare the fleet

Install fleetsh and [register it in PATH](../../installation/), then add your
hosts to the inventory. Here is an example with two key-authenticated hosts:

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups apt
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups apt
fleetsh ls '@all'
```

For each host, establish trust interactively after independently verifying its
fingerprint. Repeat for the other aliases:

```sh
fleetsh ssh hk1 --connect-timeout 60s
```

Exit the remote shell after checking the connection. Unattended commands reject
unknown or changed host keys. Quote `@all` and other group selectors in
PowerShell; the same quoting works in Bash and Zsh.

The following commands assume the SSH account can use sudo. fleetsh uses
passwordless sudo unless the host has a `sudo_credential` reference.
For password-based sudo, save a credential with hidden input and associate it:

```sh
fleetsh credential add hk1-sudo
fleetsh edit hk1 --sudo-credential hk1-sudo
```

Repeat for other accounts as needed. When connecting as root, omit `--sudo`.

## 2. Preview the updates

```sh
fleetsh update '@all' --dry-run --sudo --parallel 5
```

This connects to identify each OS and displays the intended command without
updating packages. Debian/Ubuntu hosts use:

```sh
DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
```

This is the scripted form of refreshing package lists and upgrading installed
packages. The `&&` runs the upgrade only when the refresh succeeds.
It updates all installed packages with available upgrades, including security
and ordinary updates. It does not limit updates to security patches.

For a mixed fleet, `update` selects the appropriate supported package manager
for each OS. Any failed preflight or unsupported distribution prevents updates
from being dispatched to every target. Console-only assets are skipped.

To select only an apt group in a larger inventory, replace `'@all'` with
`'@apt'` in the commands below.

## 3. Apply the patches

```sh
fleetsh update '@all' --sudo --parallel 5 --timeout 30m
```

Review the listed aliases and confirm. At most five hosts are updated at a time.
Use `--serial` in place of `--parallel 5` to process one host at a time.
The timeout applies to each remote update command. fleetsh does not automatically
retry failed commands or roll back packages after a partial fleet failure.
Updates can restart services; run during your chosen maintenance window.

### Run the literal apt command

When every selected host is Debian/Ubuntu, you can also run apt directly:

```sh
fleetsh exec '@all' "DEBIAN_FRONTEND=noninteractive apt update && DEBIAN_FRONTEND=noninteractive apt upgrade -y" --sudo --parallel 5 --timeout 30m
```

`--sudo` applies to the whole shell command, including both sides of `&&`.
`-y` supplies the package upgrade confirmation because batch exec has no
interactive stdin. `exec` runs immediately, without the built-in update's OS
preflight, dry run or confirmation. apt-get is preferred for scripted operation;
the built-in action uses it. Package-specific prompts may still cause an
unattended upgrade to fail; inspect that host's stderr before rerunning.

## 4. Inspect results and automate

Interactive output shows each host's success/failure, stdout/stderr and a summary.
Exit code 0 means all executable targets succeeded; 1 means a remote command
failed, timed out or was canceled; 2 means a local/configuration/credential
preparation error; 3 means a connection/authentication/proxy/host-key error.
A failed refresh never runs the upgrade on that host.

For explicitly approved automation, use `--yes --json`:

```sh
fleetsh update '@all' --sudo --parallel 5 --timeout 30m --yes --json > patch-results.json
```

The JSON report has `results` and `summary`, including per-host errors.
Check the process exit code as well as the report.
In PowerShell, inspect `$LASTEXITCODE` immediately after fleetsh;
in Bash/Zsh, use `$?`. Rerun only the failed aliases after fixing their causes.

## 5. Check whether a reboot is needed

For Debian/Ubuntu, check the reboot-required file:

```sh
fleetsh exec '@all' "if test -f /var/run/reboot-required; then echo REBOOT_REQUIRED; else echo NO_REBOOT_MARKER; fi" --parallel 5
```

A missing marker means the distribution has not signaled a reboot through that
file; it does not prove that every service has loaded updated libraries.
Choose the aliases that need a reboot, then preview and reboot them:

```sh
fleetsh reboot hk1,sg1 --dry-run --sudo
fleetsh reboot hk1,sg1 --sudo --parallel 2 --wait-timeout 5m
```

Confirm the selected hosts. Success requires a new Linux boot ID and a passing
uptime probe. The patch action itself never reboots automatically.

See [getting started](../../getting-started/) for command details and
[security design](../../security/) for credential and host-key behavior.
