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

## Run with npx or install with npm

With Node.js 22 or newer, run the single [npm package](https://www.npmjs.com/package/fleetsh):

```sh
npx fleetsh@alpha version
npx fleetsh@alpha init
npx fleetsh@alpha alive
npx fleetsh@alpha stats
npx fleetsh@alpha ssh hk1
```

Or install once and use the command directly:

```sh
npm install -g fleetsh@alpha
fleetsh version
fleetsh stats
```

npm puts the command in its global executable directory. Windows uses the output
of `npm config get prefix`; macOS/Linux use that prefix's `bin` directory.
If necessary, add that directory to PATH. You do not need to copy the Go binary
manually when npm's executable directory is already on PATH.

### Automatic Go releases and command-line version selection

Starting with npm launcher `0.1.0-alpha.2`, the npm version and the Go version are
independent. The default `latest` selector prefers the newest stable GitHub Release;
if no stable release exists, it selects the most recently published preview.
New Go releases need no new npm publication. Only launcher changes require npm updates.

Place launcher options **before** the Go command:

```sh
npx fleetsh@alpha --release latest stats
npx fleetsh@alpha --release preview stats
npx fleetsh@alpha --release bundled version
npx fleetsh@alpha --release v0.1.0-alpha.1 stats
npx fleetsh@alpha --refresh version
npx fleetsh@alpha --offline stats
npx fleetsh@alpha --launcher-help
```

| Selector or option | Behavior |
| --- | --- |
| `--release latest` | Stable first; preview only while no stable release exists (default) |
| `--release preview` | Most recently published release, including previews |
| `--release bundled` | Original binary pinned in this launcher: `v0.1.0-alpha.1` |
| `--release vVERSION` | Exact Go release, regardless of the npm launcher version |
| `--refresh` | Check GitHub immediately, bypassing the one-hour metadata cache |
| `--offline` | Use cached metadata/binary; make no GitHub requests |
| `--launcher-help` | Show launcher help without downloading or starting Go |

The same options work after global installation:
`fleetsh --release bundled version` or `fleetsh --release v0.1.0-alpha.1 stats`.
Environment alternatives are `FLEETSH_RELEASE`, `FLEETSH_REFRESH=1` and
`FLEETSH_OFFLINE=1`; command-line options take precedence.

`fleetsh version` prints the selected **Go** version. `npm ls -g fleetsh --depth=0`
shows the installed **launcher** version. Pinning `npx fleetsh@0.1.0-alpha.2`
pins the launcher; add `--release vVERSION` to pin the Go program too.
The original npm launcher `0.1.0-alpha.1` always selects Go `v0.1.0-alpha.1` and
does not support these options. Update it once with `npm install -g fleetsh@alpha`.

### Downloads, cache and offline use

Release information is checked at most once per hour in normal online use.
New selected versions are downloaded automatically and cached separately.
GitHub publishes a manifest with archive and executable SHA256 hashes; the launcher
verifies it against GitHub's asset digest, then verifies the archive, executable
and every cached execution. The initial release uses its embedded verified manifest.

Network outages or rate limiting reuse previously verified release metadata, or
the bundled manifest on first use. Invalid hashes or release formats fail explicitly.
Offline use requires the selected binary to have been downloaded earlier; npx
may still contact npm to resolve/install its launcher. Global installation avoids
that npx resolution step. `--offline` controls the launcher's GitHub traffic.

| OS | Binary and release-metadata cache |
| --- | --- |
| Windows | `%LOCALAPPDATA%\fleetsh\Cache\npm` |
| macOS | `~/Library/Caches/fleetsh/npm` |
| Linux | `$XDG_CACHE_HOME/fleetsh/npm` or `~/.cache/fleetsh/npm` |

Use `FLEETSH_NPM_CACHE` for another writable cache directory. Downloads do not use
the inventory's SSH proxy. Windows, macOS and Linux each support x64 and ARM64.
There are no runtime npm dependencies, platform packages or install hooks.
Arguments, interactive input/output and exit codes pass through to Go.
Inventory, host trust and credential locations are the same as with the direct binary.
Download/status messages use stderr, preserving JSON stdout.

For a cached checksum mismatch, remove only the cache directory named in the
error and retry online. Without Node.js or GitHub access, download a
[release archive](https://github.com/KanataLabs/fleetsh/releases) elsewhere and
use the binary/PATH instructions below.

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
