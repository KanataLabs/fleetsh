---
title: 快速开始
permalink: /zh/getting-started/
lang: zh-CN
locale: zh
page_key: 'getting-started/'
---

# 快速开始

fleetsh 已实现 v0.1 的核心流程，目前仍是开发版。
先参考 [安装与全局 PATH 注册](../installation/)，让命令在任意目录可用。

## 使用 npx 或 npm 运行

需要 Node.js 22 或更新版本：

```sh
npx fleetsh init
npx fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519
npx fleetsh ssh hk1
npx fleetsh alive
npx fleetsh stats
```

也可先 `npm install -g fleetsh` 全局安装，把所有示例的
`npx fleetsh` 前缀替换为 `fleetsh`。两种方式与直接运行二进制共用主机清单和系统凭据库。

启动器自动跟随 GitHub Release，优先正式版，没有正式版时才使用预览版。
启动器选项放在子命令前：`npx fleetsh --release bundled version` 使用原版，
`--release v0.1.0-alpha.1` 固定某个 Go 版本。
可运行 `npx fleetsh --launcher-help`，或阅读
[版本选择、缓存和 PATH](../installation/)。

## 帮助与离线指南

程序内置带示例的英文命令帮助和十一个离线指南。执行 `init` 前即可查看，
无需配置文件、网络连接或系统凭据库。在线文档继续按语言分开，默认英语。

位置参数缺失、过多或选项无效时，会在标准错误输出中显示具体错误及该命令的
语法、示例和选项，并返回退出码 2。`npx fleetsh exec` 和
`npx fleetsh exec whoami` 都会显示帮助；`exec` 需要同时提供目标主机或组，以及
一个带引号的远程命令，例如 `npx fleetsh exec '@all' "whoami"`。
程序内帮助链接使用可直接访问的 HTTPS 文档地址，避开 GitHub Pages 默认地址
继承的 HTTP 跳转。

```sh
fleetsh --help
fleetsh help exec
fleetsh add --help
fleetsh docs
fleetsh docs passwords
fleetsh docs groups
fleetsh docs patching
```

| 主题 | 内容 |
| --- | --- |
| `quickstart` | 首台主机、SSH 指纹信任和常用操作 |
| `config` | 各系统配置位置、默认值和备份 |
| `passwords` | 密码自动保存、更新、共享引用及 sudo |
| `groups` | 多分组、增删成员关系和标签 |
| `selectors` | 主机与分组的并集、`@all` 和标签筛选 |
| `exec` | 并发、引号、超时、sudo 和 JSON |
| `proxies` | 全局/单机 SOCKS5、HTTP/HTTPS CONNECT 和跳板 |
| `patching` | 更新预演、apt 等包管理器升级和重启 |
| `monitoring` | SSH 存活、CPU、内存、swap、用户和磁盘 |
| `forwarding` | 本地、远程和动态 TCP 转发 |
| `troubleshooting` | 退出码、信任、认证和系统凭据库 |

使用 `fleetsh docs TOPIC` 查看具体指南；离线正文采用英文纯文本。
`--json` 用于支持它的列表和批量操作结果，不改变文档格式。
查看帮助不会执行其中的示例，也不会自动打开文档链接。

## 初始化主机清单

清单保存在本机当前用户的配置目录：Windows 为 `%APPDATA%\fleetsh\config.toml`，
macOS 为 `~/Library/Application Support/fleetsh/config.toml`，
Linux 为 `$XDG_CONFIG_HOME/fleetsh/config.toml` 或 `~/.config/fleetsh/config.toml`。
详见 [存储位置、自定义清单与备份迁移](../configuration/#storage)。

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups asia,web
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth password --groups asia
fleetsh ls
fleetsh ls '@asia' --json
fleetsh show hk1
fleetsh edit hk1 --port 2222
```

密码输入不会显示。凭据值保存在操作系统凭据库中，清单仅保存引用名称。
密钥口令也可使用 `--credential` 引用；没有引用时会通过隐藏输入询问。

现在添加密码认证主机会直接询问两次密码并自动保存，无需手动建立 credential。
自动引用以 `fleetsh-ssh-` 开头，清单只保存名称。
可用 `fleetsh edit sg1 --save-password` 重新保存密码。
详见 [密码保存](../configuration/#passwords) 和 [后续修改分组](../configuration/#groups)。

所有命令支持 `--config PATH`，以 `~/` 开头的路径会展开。
清单会严格校验，未知字段和明文 `password` 字段会被拒绝。
`edit` 会重写 TOML，不保留注释。

## 确认主机身份

```sh
fleetsh ssh hk1 --connect-timeout 60s
fleetsh hostkey show hk1
```

首次连接显示 SHA256 指纹，需要交互输入 `y` 确认。
请通过独立来源核实指纹后再信任。确认过程共用连接超时；
需要更多时间时可调整 `--connect-timeout`。
主机密钥变化始终拒绝连接；无人值守命令会拒绝未知密钥。

确认确实发生合法密钥轮换后：

```sh
fleetsh hostkey reset hk1
fleetsh ssh hk1 --connect-timeout 60s
```

专用 `known_hosts` 文件位于所选清单旁边。
标准输入为终端时，交互 SSH 会使用 PTY 和终端原始模式，并传递窗口大小变化。
端口转发使用命名配置和 `fleetsh forward`；OpenSSH 参数透传仍在规划中。

## 一键远程监控

```sh
fleetsh alive
fleetsh stats
fleetsh alive '@web' --timeout 10s
fleetsh stats '@web' --parallel 5 --timeout 15s
fleetsh monitor hk1
fleetsh stats --json
```

`alive` 检查 SSH 认证及远程命令执行是否可用；`stats`（别名 `monitor`）
一次打印 Linux 的当前用户、CPU 一秒采样占用、内存及 swap 的 MiB/百分比占用、
以及 `df -h -P` 的磁盘空余。两个命令默认选择 `@all`，也支持选择器、标签、
并发和超时参数。仅控制台资产会跳过，JSON 使用标准批量报告，快照放在各主机的 `stdout` 字段。

无需安装远程组件或使用 sudo，只需 Linux `/proc` 和 `awk`、`sleep`、`whoami`、`df`。
内存占用按总量减可用量计算，未启用 swap 时显示零。CPU 是主机级计数，
不反映容器配额。每次调用采集一份快照；使用 `--sudo` 会改变显示的用户。
离线说明见 `fleetsh docs monitoring`。

端口转发用 `fleetsh forward HOST [NAME]` 启动，
详见 [本地、远程及动态转发配置](../configuration/#forwarding)。

## 执行命令

PowerShell 中请为分组选择器加引号，例如 `fleetsh ls '@asia'`，
避免把 `@` 前缀解析为变量展开语法。相同写法也适用于 Bash 和 Zsh。

```sh
fleetsh exec hk1 "uptime"
fleetsh exec '@asia' "df -h" --parallel 10
fleetsh exec hk1,sg1 "uptime" --serial --json
fleetsh exec all "docker ps" --tag web --timeout 60s
fleetsh exec hk1 "apt-get update" --sudo
```

主机没有 `sudo_credential` 时，`--sudo` 使用免密 `sudo -n`。
引用的 sudo 密码通过 SSH 通道标准输入传送，不会拼接到命令中。
命令执行不接收交互输入；交互程序请使用 `ssh`。

输出按主机组织，每台主机的标准输出和标准错误分别最多保留 4 MiB。
JSON 包含 `results` 和 `summary`，记录主机、退出状态、耗时、错误分类、
截断和跳过状态。仅控制台资产会明确标记为跳过。

退出码：0 表示所有可执行目标成功；1 表示命令失败、取消或执行超时；
2 表示本地用法、配置或凭据准备错误；3 表示连接、认证、代理或主机密钥失败，
包括连接超时。混合远程错误中命令失败优先。
交互远程 shell 非零退出返回 1，传输失败返回 3。

## 更新与重启 Linux VPS

```sh
fleetsh update '@asia' --dry-run --sudo
fleetsh update '@asia' --sudo
fleetsh update hk1 --dist --sudo --yes
fleetsh reboot '@asia' --dry-run --sudo
fleetsh reboot '@asia' --parallel 2 --sudo --wait-timeout 5m
```

更新预演会连接以识别操作系统，但不会执行软件包更新。
支持 Debian、Ubuntu、RHEL、Rocky、AlmaLinux、Fedora、CentOS、Arch 和 Manjaro。
任何目标的系统不受支持或预检查失败，都会阻止全部目标执行更新。
重启预演不会连接服务器。

实际更新和重启会显示所选别名并要求确认；自动化显式使用 `--yes`。
重启默认并发为 2。只有 Linux 启动标识变化且 uptime 健康探测成功，才算重启成功。
命令不会自动重试。

完整维护流程见 [给所有 VPS 打补丁](../cases/patch-all-vps/)。
