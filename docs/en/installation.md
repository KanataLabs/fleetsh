---
title: Installation and PATH
permalink: /en/installation/
lang: en
locale: en
page_key: 'installation/'
---

# Installation and global PATH registration

Place the binary in a permanent directory and add that **directory** to PATH
to run `fleetsh` from any working directory. No Go, Python or Node.js runtime
is required to run the binary. You can currently build the development version
from source; no stable release is available yet.

## Build from source

Building requires Go 1.27 or newer:

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go build -trimpath -o dist/fleetsh ./cmd/fleetsh
```

On Windows PowerShell:

```powershell
git clone https://github.com/KanataLabs/fleetsh.git
Set-Location fleetsh
go build -trimpath -o dist/fleetsh.exe ./cmd/fleetsh
```

Alternatively, run `go install github.com/KanataLabs/fleetsh/cmd/fleetsh@latest`.
This installs into the directory reported by `go env GOBIN`; when GOBIN is empty,
it uses the `bin` subdirectory of `go env GOPATH`. Add that directory to PATH too.

## Windows: install for your user (recommended, no administrator required)

Run these commands from the project directory where you built the binary:

```powershell
$binDir = Join-Path $env:LOCALAPPDATA 'fleetsh\bin'
New-Item -ItemType Directory -Path $binDir -Force | Out-Null
Copy-Item -LiteralPath '.\dist\fleetsh.exe' -Destination (Join-Path $binDir 'fleetsh.exe') -Force

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$entries = @($userPath -split ';' | Where-Object { $_ })
if ($entries -notcontains $binDir) {
    [Environment]::SetEnvironmentVariable('Path', (($entries + $binDir) -join ';'), 'User')
}

# Apply to the current PowerShell session
if (($env:Path -split ';') -notcontains $binDir) {
    $env:Path = $env:Path + ';' + $binDir
}

Get-Command fleetsh
fleetsh version
```

This preserves existing user PATH entries and leaves the system PATH intact.
Close and reopen other terminals. If Windows Terminal is already running,
you may need to quit the entire application before reopening it.

You can also use the graphical editor: search for “Edit environment variables
for your account”, select the user Path variable, choose Edit, then New, and add
`%LOCALAPPDATA%\fleetsh\bin`. Restart your terminal.

### Windows: install for all users (administrator required)

Copy into the system directory and update Machine PATH in an administrator PowerShell:

```powershell
$binDir = Join-Path $env:ProgramFiles 'fleetsh\bin'
New-Item -ItemType Directory -Path $binDir -Force | Out-Null
Copy-Item -LiteralPath '.\dist\fleetsh.exe' -Destination (Join-Path $binDir 'fleetsh.exe') -Force

$machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
$entries = @($machinePath -split ';' | Where-Object { $_ })
if ($entries -notcontains $binDir) {
    [Environment]::SetEnvironmentVariable('Path', (($entries + $binDir) -join ';'), 'Machine')
}
$env:Path = $env:Path + ';' + $binDir
fleetsh version
```

The installation directory can be shared; each user still has their own
configuration directory and OS credential store.

## macOS: install for your user (default Zsh)

Run from the project directory:

```sh
mkdir -p "$HOME/.local/bin"
install -m 0755 dist/fleetsh "$HOME/.local/bin/fleetsh"

# Register once; preserve existing .zprofile contents
touch "$HOME/.zprofile"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.zprofile" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zprofile"

export PATH="$HOME/.local/bin:$PATH"
command -v fleetsh
fleetsh version
```

Zsh login shells read `.zprofile`. For a non-login terminal shell, add the same
`export PATH="$HOME/.local/bin:$PATH"` line to `.zshrc`.

## Linux: install for your user (Bash)

```sh
mkdir -p "$HOME/.local/bin"
install -m 0755 dist/fleetsh "$HOME/.local/bin/fleetsh"

touch "$HOME/.bashrc"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.bashrc" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"

# Also configure login shells through .profile
touch "$HOME/.profile"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.profile" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.profile"

export PATH="$HOME/.local/bin:$PATH"
command -v fleetsh
fleetsh version
```

For Zsh, use the `.zshrc` / `.zprofile` approach above.
If you have a custom `.bash_profile`, make sure it reads `.profile` or add
the same PATH setting there. Fish users can run `fish_add_path "$HOME/.local/bin"`.

### macOS / Linux: system installation

```sh
sudo mkdir -p /usr/local/bin
sudo install -m 0755 dist/fleetsh /usr/local/bin/fleetsh
command -v fleetsh
fleetsh version
```

Most systems already include `/usr/local/bin` in PATH. If `command -v` cannot
find the binary, add `export PATH="/usr/local/bin:$PATH"` to your shell's configuration.

## Update, verify and uninstall

To update, copy the new binary into the same installation directory.
You do not need to register PATH again. Close running fleetsh processes before
updating on Windows. If a Unix shell caches an old location, run `hash -r`;
Zsh uses `rehash`.

Verify with `fleetsh version`. Use `Get-Command fleetsh -All` on Windows
or `type -a fleetsh` on Unix to check competing binaries.
PATH order determines which binary runs.

To uninstall, delete the installed fleetsh binary and remove its directory
entry from PATH or the relevant shell configuration.
Configuration and credentials remain; use `fleetsh credential rm REFERENCE`
first if you also want to remove stored credentials.
