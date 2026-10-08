---
title: 安装与 PATH
permalink: /zh/installation/
lang: zh-CN
locale: zh
page_key: 'installation/'
---

# 安装与全局 PATH 注册

将二进制文件放进固定目录，再把**目录**加入 PATH，就可以在任意工作目录运行
`fleetsh`。运行时不需要 Go、Python 或 Node.js；目前可从源码构建开发版，
还没有发布稳定版。

## 使用 npx 或通过 npm 安装

安装 Node.js 22 或更新版本后，只需使用一个 npm 包：

```sh
npx fleetsh@alpha version
npx fleetsh@alpha init
npx fleetsh@alpha stats
npx fleetsh@alpha ssh hk1
```

全局安装后即可直接运行：

```sh
npm install -g fleetsh@alpha
fleetsh version
```

npm 会在全局可执行文件目录建立命令入口。Windows 对应 `npm config get prefix`
的输出目录；macOS/Linux 对应该目录下的 `bin`。若 Node.js 安装程序没有将其
加入 PATH，请补充这个目录；已在 PATH 中时无需手动复制二进制或注册路径。

当前 npm 版本为 `0.1.0-alpha.1`，`alpha` 是预览通道。
用 `npx fleetsh@0.1.0-alpha.1 version` 可以固定版本。首次**运行**时，
启动器从 GitHub Releases 下载对应版本的 Go 二进制，校验包内固定的归档和
可执行文件 SHA256，再缓存到本机。没有安装钩子、运行时 npm 依赖或平台子包。
首次运行需要访问 GitHub 并有可写缓存目录；之后启动器复用已校验的二进制。
npx 解析包版本时仍可能访问 npm registry。

| 系统 | 二进制缓存 |
| --- | --- |
| Windows | `%LOCALAPPDATA%\fleetsh\Cache\npm` |
| macOS | `~/Library/Caches/fleetsh/npm` |
| Linux | `$XDG_CACHE_HOME/fleetsh/npm` 或 `~/.cache/fleetsh/npm` |

可通过 `FLEETSH_NPM_CACHE` 指定其他缓存目录。下载过程不会使用主机清单的
SSH 代理配置。支持 Windows、macOS、Linux 的 x64 和 ARM64 架构。
参数、交互输入输出和退出码原样交给 Go 程序，主机清单和凭据位置保持一致。
下载提示写入 stderr，因此不会混入 stdout 的 JSON。

若提示缓存校验失败，只删除错误中指出的缓存目录，再重试。
没有 Node.js 或无法访问 GitHub 的环境，可以在其他机器下载
[发行归档](https://github.com/KanataLabs/fleetsh/releases)，再按下文配置二进制及 PATH。

## 从源码构建

构建需要 Go 1.27 或更新版本：

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go build -trimpath -o dist/fleetsh ./cmd/fleetsh
```

Windows PowerShell 使用：

```powershell
git clone https://github.com/KanataLabs/fleetsh.git
Set-Location fleetsh
go build -trimpath -o dist/fleetsh.exe ./cmd/fleetsh
```

也可以用 `go install github.com/KanataLabs/fleetsh/cmd/fleetsh@latest`。
它将文件安装到 `go env GOBIN` 指定的目录；GOBIN 为空时使用
`go env GOPATH` 的 `bin` 子目录。该目录也需要加入 PATH。

## Windows：用户级全局注册（推荐，无需管理员）

在刚才构建项目的目录执行：

```powershell
$binDir = Join-Path $env:LOCALAPPDATA 'fleetsh\bin'
New-Item -ItemType Directory -Path $binDir -Force | Out-Null
Copy-Item -LiteralPath '.\dist\fleetsh.exe' -Destination (Join-Path $binDir 'fleetsh.exe') -Force

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$entries = @($userPath -split ';' | Where-Object { $_ })
if ($entries -notcontains $binDir) {
    [Environment]::SetEnvironmentVariable('Path', (($entries + $binDir) -join ';'), 'User')
}

# 让当前 PowerShell 立即生效
if (($env:Path -split ';') -notcontains $binDir) {
    $env:Path = $env:Path + ';' + $binDir
}

Get-Command fleetsh
fleetsh version
```

这保留已有用户 PATH，不覆盖系统 PATH。其他终端需要关闭后重新启动；
Windows Terminal 已运行时，可能需要退出整个应用再打开。

也可使用图形界面：搜索“编辑账户的环境变量” → 用户变量 Path → 编辑 → 新建 →
填写 `%LOCALAPPDATA%\fleetsh\bin`，然后重启终端。

### Windows：所有用户可用（需要管理员）

在管理员 PowerShell 中复制到系统目录并更新 Machine PATH：

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

安装目录可以共享，但每个用户仍使用自己的配置目录和系统凭据库。

## macOS：用户级注册（默认 Zsh）

在项目目录执行：

```sh
mkdir -p "$HOME/.local/bin"
install -m 0755 dist/fleetsh "$HOME/.local/bin/fleetsh"

# 只需要注册一次；保留原有 .zprofile 内容
touch "$HOME/.zprofile"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.zprofile" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zprofile"

export PATH="$HOME/.local/bin:$PATH"
command -v fleetsh
fleetsh version
```

Zsh 登录 shell 会读取 `.zprofile`。如果你的终端使用非登录 shell，将同一行
`export PATH="$HOME/.local/bin:$PATH"` 添加到 `.zshrc`。

## Linux：用户级注册（Bash）

```sh
mkdir -p "$HOME/.local/bin"
install -m 0755 dist/fleetsh "$HOME/.local/bin/fleetsh"

touch "$HOME/.bashrc"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.bashrc" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"

# 登录 shell 的环境也从 .profile 设置
touch "$HOME/.profile"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.profile" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.profile"

export PATH="$HOME/.local/bin:$PATH"
command -v fleetsh
fleetsh version
```

使用 Zsh 时采用上面的 `.zshrc`/`.zprofile` 方案。存在自定义
`.bash_profile` 时，确认它会读取 `.profile` 或在其中加入相同 PATH 设置。
Fish 可使用 `fish_add_path "$HOME/.local/bin"`。

### macOS / Linux：系统级安装

```sh
sudo mkdir -p /usr/local/bin
sudo install -m 0755 dist/fleetsh /usr/local/bin/fleetsh
command -v fleetsh
fleetsh version
```

多数系统已将 `/usr/local/bin` 加入 PATH。如果 `command -v` 没有找到它，
在所用 shell 的配置文件中加入 `export PATH="/usr/local/bin:$PATH"`。

## 更新、验证与卸载

更新后，将新二进制复制到同一个安装目录，无需重复注册 PATH。
Windows 更新前关闭正在运行的 fleetsh；Unix shell 找到旧路径时可执行
`hash -r`，Zsh 使用 `rehash`。

验证时运行 `fleetsh version`；Windows 用 `Get-Command fleetsh -All` 检查同名文件，
Unix 用 `type -a fleetsh`。PATH 中的先后顺序决定实际执行哪个文件。

卸载时删除安装目录中的 fleetsh 二进制，然后从 PATH 或 shell 配置中移除对应目录行。
配置与凭据不会随二进制删除；需要时先运行 `fleetsh credential rm REFERENCE`。
