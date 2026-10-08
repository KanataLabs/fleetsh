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
还没有发布 Go 稳定版。

## 使用 npx 或通过 npm 安装

已有 Node.js 22 或更新版本时，可直接运行单个 [npm 包](https://www.npmjs.com/package/fleetsh)。
npm 的 `latest` 标签提供独立的启动器 `0.1.0`；下载的 Go 程序仍可能是预览版。

```sh
npx fleetsh version
npx fleetsh init
npx fleetsh alive
npx fleetsh stats
npx fleetsh ssh hk1
```

也可以全局安装一次，随后直接运行：

```sh
npm install -g fleetsh
fleetsh version
fleetsh stats
```

npm 会在全局可执行文件目录建立命令入口。Windows 使用 `npm config get prefix`
输出的目录，macOS/Linux 使用该目录下的 `bin`。若还没有加入 PATH，请补上对应目录。
全局 npm 命令目录已在 PATH 时，无需手动复制 Go 二进制。

### 自动跟随 GitHub Release，或命令行选择版本

从 npm 启动器 `0.1.0-alpha.2` 起，npm 包版本与 Go 程序版本分离。
默认 `latest` 优先使用最新正式 GitHub Release；尚无正式版时，使用最近发布的预览版。
以后更新 Go 程序只需要发布 GitHub Release，启动器本身变化时才需要再发 npm 包。

启动器选项放在 Go 子命令**前面**：

```sh
npx fleetsh --release latest stats
npx fleetsh --release preview stats
npx fleetsh --release bundled version
npx fleetsh --release v0.1.0-alpha.1 stats
npx fleetsh --refresh version
npx fleetsh --offline stats
npx fleetsh --launcher-help
```

| 选项 | 行为 |
| --- | --- |
| `--release latest` | 优先正式版；没有正式版时才选预览版，默认采用此模式 |
| `--release preview` | 最近发布的版本，包含预览版 |
| `--release bundled` | 启动器内固定的原版，目前为 `v0.1.0-alpha.1` |
| `--release vVERSION` | 固定 Go 程序版本，不随 npm 启动器版本变化 |
| `--refresh` | 立即检查 GitHub，跳过一小时的发行信息缓存 |
| `--offline` | 只使用已有缓存，不向 GitHub 请求信息或二进制 |
| `--launcher-help` | 查看启动器帮助，不下载或启动 Go 程序 |

全局安装后同样可以运行 `fleetsh --release bundled version` 或
`fleetsh --release v0.1.0-alpha.1 stats`。
也可设置 `FLEETSH_RELEASE`、`FLEETSH_REFRESH=1`、`FLEETSH_OFFLINE=1`；
命令行选项优先于环境变量。

`fleetsh version` 显示实际运行的 **Go 程序**版本；
`npm ls -g fleetsh --depth=0` 显示 **npm 启动器**版本。
`npx fleetsh@0.1.0` 只固定启动器；同时加上 `--release vVERSION` 才固定 Go 程序。
旧 npm 启动器 `0.1.0-alpha.1` 一直使用 Go `v0.1.0-alpha.1`，不支持这些选项。
已全局安装旧版时，先运行一次 `npm install -g fleetsh` 更新。

### 下载、缓存和离线运行

正常在线使用时，每小时最多检查一次 GitHub 发行信息；选中新版本后自动下载，
不同版本分开缓存。发行包包含归档和二进制 SHA256 清单，启动器先使用 GitHub
资产摘要校验清单，再校验下载归档和二进制，每次缓存运行也检查二进制摘要。
最初的发行版兼容启动器内已有的固定校验清单。

网络故障或限流时，复用之前校验过的发行信息；首次使用时可退回固定原版清单。
格式或摘要不匹配会明确报错。离线运行要求选定二进制已下载过。
npx 解析或安装启动器时仍可能访问 npm registry；全局安装后可避免这个步骤。
`--offline` 控制启动器对 GitHub 的访问。

| 系统 | 二进制及发行信息缓存 |
| --- | --- |
| Windows | `%LOCALAPPDATA%\fleetsh\Cache\npm` |
| macOS | `~/Library/Caches/fleetsh/npm` |
| Linux | `$XDG_CACHE_HOME/fleetsh/npm` 或 `~/.cache/fleetsh/npm` |

可用 `FLEETSH_NPM_CACHE` 指定其他可写缓存目录。下载不使用主机清单中的 SSH 代理。
支持 Windows、macOS、Linux 的 x64 和 ARM64。没有运行时 npm 依赖、平台子包或安装钩子。
参数、交互输入输出和退出码交给 Go 程序；主机清单、信任记录和凭据位置与直接运行一致。
下载与状态提示写入 stderr，不会混入 stdout 的 JSON。

缓存摘要错误时，只删除错误中指出的缓存目录，再联网重试。
没有 Node.js 或 GitHub 访问条件时，可在其他机器下载
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
