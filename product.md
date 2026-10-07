---
title: Product brief
permalink: /product/
lang: zh-CN
---

> 这是原始产品方案，已统一项目及命令名为 fleetsh。功能均为规划；当前实现范围见路线图。
> 版本范围以 [Roadmap](../roadmap/) 为准，本文保留原始讨论中的优先级差异。

# Lightweight fleetsh Fleet Manager

## 1. 产品定位

一个面向个人开发者、小团队的轻量 fleetsh 管理工具。

核心目标：

- Single binary
- 无 Python / Node.js / Java runtime
- 无服务端 Agent
- 不要求安装中心服务器
- Windows / macOS / Linux 可运行
- 管理多台 fleetsh 的连接信息和凭据
- 支持 SSH Password / SSH Key / SSH Agent
- 支持 Proxy / Jump Host
- 支持单机和批量执行命令
- 提供少量常用 fleetsh 管理动作
- 配置简单，可导出、备份、迁移
- 默认安全，不明文保存密码

它不是 Ansible 的替代品，不负责完整 Configuration Management。

典型使用方式：

```bash
fleetsh ls
fleetsh ssh hk1

fleetsh exec hk1 "uptime"
fleetsh exec all "uptime"
fleetsh exec @web "docker ps"

fleetsh update hk1
fleetsh update @all

fleetsh reboot hk1
```

---

# 2. 核心设计原则

## 2.1 Single Binary

最终发行物：

```text
fleetsh.exe
fleetsh
```

不要求：

```text
Python
pip
Node.js
Docker
Java
Ruby
```

允许调用操作系统已有能力，例如：

```text
ssh-agent
Windows Credential Manager
macOS Keychain
Linux Secret Service
```

但核心功能应尽可能内置。

---

## 2.2 Agentless

目标 fleetsh 不安装任何常驻程序。

通信方式：

```text
fleetsh
  ↓
SSH
  ↓
fleetsh
```

只要服务器能够 SSH 登录即可执行管理操作。

对于无法 SSH 的 fleetsh，可以记录在 inventory 中，但标记为：

```text
connection = "console-only"
```

这种机器只做资产/凭据管理，不执行远程命令。

---

# 3. Host 管理

最低限度需要保存：

```text
ID / Alias
Hostname / IP
Port
Username
Authentication type
Credential reference
Proxy
Tags / Groups
Description
```

例如：

```toml
[hosts.hk1]
host = "1.2.3.4"
port = 22
user = "root"
auth = "password"
credential = "hk1-root"
groups = ["all", "asia", "web"]

[hosts.sg1]
host = "sg1.example.com"
user = "ubuntu"
auth = "key"
key = "~/.ssh/id_ed25519"
groups = ["all", "asia"]

[hosts.cn1]
host = "10.0.0.10"
user = "root"
auth = "password"
credential = "cn1-root"
proxy = "proxy-hk"
groups = ["all", "china"]
```

CLI：

```bash
fleetsh add
fleetsh edit hk1
fleetsh rm hk1
fleetsh show hk1
fleetsh ls
```

支持非交互式：

```bash
fleetsh add hk1 \
  --host 1.2.3.4 \
  --user root \
  --auth password
```

---

# 4. Group / Tag

这是批量管理最重要的功能之一。

例如：

```text
all

asia
├── hk1
├── sg1
└── jp1

web
├── hk1
└── us1

bt
├── hk1
├── sg1
└── us1
```

CLI：

```bash
fleetsh ls @asia
fleetsh exec @asia "uptime"
fleetsh exec @bt "bt update"
```

建议同时支持：

```bash
fleetsh exec hk1,sg1,jp1 "uptime"
```

以及：

```bash
fleetsh exec --tag web "docker ps"
```

---

# 5. Authentication

必须支持：

## SSH Key

```text
private key file
encrypted private key
ssh-agent
```

例如：

```toml
auth = "key"
key = "~/.ssh/id_ed25519"
```

或者：

```toml
auth = "agent"
```

---

## Password

例如：

```toml
auth = "password"
credential = "hk1-root"
```

**config 中不得保存密码。**

不要：

```toml
password = "123456"
```

---

# 6. Credential Manager

这是这个工具与普通 parallel-ssh 工具相比很重要的一部分。

推荐：

### Windows

```text
Windows Credential Manager
```

### macOS

```text
Keychain
```

### Linux

优先：

```text
Secret Service
```

例如 GNOME Keyring / KWallet。

无法使用 OS credential store 时，可以提供 fallback：

```text
encrypted local credential database
```

例如：

```text
credentials.db
```

使用：

```text
AES-256-GCM
```

master key 由用户提供或通过安全 KDF 得到。

CLI：

```bash
fleetsh credential add hk1-root
fleetsh credential ls
fleetsh credential rm hk1-root
```

输入密码时：

```text
Password:
Confirm:
```

不得显示，也不得写入 shell history。

---

# 7. Proxy

这是 P0 功能。

至少支持三类：

## ProxyJump

```text
Local
 ↓
Jump Host
 ↓
Target
```

配置：

```toml
[hosts.jump-hk]
host = "1.2.3.4"
user = "root"

[hosts.cn1]
host = "10.0.0.2"
user = "root"
proxy_jump = "jump-hk"
```

CLI：

```bash
fleetsh ssh cn1
```

自动走：

```text
local → jump-hk → cn1
```

---

## SOCKS5

例如：

```toml
proxy = "socks5://127.0.0.1:7890"
```

典型：

```text
fleetsh
  ↓
Clash / sing-box
  ↓
SOCKS5
  ↓
fleetsh
```

必须支持：

```text
SOCKS5 without auth
SOCKS5 username/password
```

---

## HTTP CONNECT Proxy

例如：

```toml
proxy = "http://127.0.0.1:8080"
```

支持 CONNECT tunnel。

---

# 8. SSH

基础 SSH 功能：

```bash
fleetsh ssh hk1
```

行为应尽可能接近：

```bash
ssh root@1.2.3.4
```

必须支持：

```text
PTY
interactive shell
password authentication
public key authentication
ssh-agent
custom port
host key verification
keepalive
timeout
```

建议支持：

```bash
fleetsh ssh hk1 -- -L 8080:localhost:80
```

即额外 SSH 参数透传。

---

# 9. Remote Exec

核心命令：

```bash
fleetsh exec hk1 "uptime"
```

输出：

```text
hk1
---
21:04:32 up 39 days, 2 users, load average: ...
```

批量：

```bash
fleetsh exec @all "uptime"
```

默认并发。

例如：

```text
hk1    ✓  0.8s
sg1    ✓  1.1s
jp1    ✓  1.3s
us1    ✗  timeout
```

然后输出各主机 stdout/stderr。

---

# 10. 并发控制

必须支持：

```bash
fleetsh exec @all "uptime" --parallel 10
```

默认：

```text
parallel = 10
```

也支持：

```bash
fleetsh exec @all "..." --serial
```

等于：

```text
parallel = 1
```

这是 reboot 和 update 时尤其重要。

---

# 11. Exit Code / Error

必须正确记录：

```text
stdout
stderr
exit code
connection error
timeout
authentication error
proxy error
```

汇总：

```text
SUMMARY

Success: 12
Failed:   2
Skipped:  0
```

程序自身 exit code：

```text
0 = 全部成功
1 = 至少一台执行失败
2 = 本地配置错误
3 = connection/authentication error
```

具体码可以后续定义。

---

# 12. sudo

必须支持：

```bash
fleetsh exec hk1 --sudo "apt update"
```

需要 sudo password 时：

```text
sudo password:
```

或者引用 Credential Store：

```toml
sudo_credential = "hk1-root"
```

不要把：

```text
echo password | sudo -S
```

暴露在日志或进程参数里。

---

# 13. Built-in Actions

不要一开始做 Ansible-style YAML。

先提供极少量、高频操作。

## status

```bash
fleetsh status @all
```

可以检测：

```text
SSH reachable
uptime
load
disk
memory
OS
kernel
```

例如：

```text
NAME  STATUS  UPTIME  DISK  MEMORY
hk1   UP      39d     53%   41%
sg1   UP      12d     67%   28%
jp1   DOWN    -       -     -
```

---

## update

```bash
fleetsh update hk1
```

自动检测发行版：

```text
Debian / Ubuntu
→ apt

RHEL / Rocky / Alma
→ dnf

CentOS old
→ yum

Arch
→ pacman
```

Ubuntu/Debian 默认：

```bash
apt-get update
apt-get upgrade -y
```

可以：

```bash
fleetsh update hk1 --dist
```

对应：

```text
dist-upgrade / full-upgrade
```

---

## reboot-required

```bash
fleetsh reboot-required @all
```

Ubuntu/Debian：

```text
/var/run/reboot-required
```

输出：

```text
hk1 YES
sg1 NO
jp1 YES
```

---

## reboot

```bash
fleetsh reboot hk1
```

不能把 SSH disconnect 当作失败。

正确流程：

```text
send reboot
↓
SSH disconnect
↓
host unavailable
↓
wait
↓
SSH becomes available
↓
execute health probe
↓
SUCCESS
```

输出：

```text
hk1 rebooting...
hk1 offline
hk1 waiting...
hk1 online
✓ reboot completed in 28s
```

批量 reboot 默认建议限制并发：

```bash
fleetsh reboot @all --parallel 2
```

---

# 14. Custom Commands / Aliases

这对 `bt update` 特别适合。

配置：

```toml
[commands.bt-update]
command = "bt update"

[commands.docker-update]
command = "docker compose pull && docker compose up -d"

[commands.disk]
command = "df -h"
```

运行：

```bash
fleetsh run hk1 bt-update
```

批量：

```bash
fleetsh run @bt bt-update
```

这样不需要 YAML。

---

# 15. Command Profiles

可以进一步允许：

```toml
[commands.patch]
debian = "apt-get update && apt-get upgrade -y"
ubuntu = "apt-get update && apt-get upgrade -y"
rocky = "dnf upgrade -y"
```

那么：

```bash
fleetsh run @all patch
```

不同系统自动选择不同 command。

但这应该是 P1，不是第一版必须做。

---

# 16. Host Key Security

默认必须：

```text
Strict Host Key Verification = ON
```

第一次连接：

```text
The authenticity of host 'hk1 (1.2.3.4)' cannot be established.

ED25519 fingerprint:
SHA256:xxxx

Trust this host? [y/N]
```

以后指纹变化：

```text
WARNING: HOST KEY CHANGED
```

默认拒绝连接。

提供：

```bash
fleetsh hostkey show hk1
fleetsh hostkey reset hk1
```

不要默认做：

```text
StrictHostKeyChecking=no
```

---

# 17. Timeout / Retry

配置：

```toml
[defaults]
connect_timeout = "10s"
command_timeout = "30m"
retries = 1
parallel = 10
```

CLI 可覆盖：

```bash
fleetsh exec @all "..." --timeout 60s
```

---

# 18. 输出模式

默认人类可读。

同时提供：

```bash
fleetsh exec @all "uptime" --json
```

用于脚本。

例如：

```json
{
  "hk1": {
    "success": true,
    "exitCode": 0,
    "stdout": "...",
    "stderr": "",
    "duration": 0.82
  }
}
```

以后可以支持：

```bash
--jsonl
--quiet
```

---

# 19. Dry Run

对高风险 built-in action：

```bash
fleetsh update @all --dry-run
```

显示：

```text
hk1
Would execute:
apt-get update && apt-get upgrade -y

sg1
Would execute:
apt-get update && apt-get upgrade -y
```

对任意 shell 命令 dry-run 意义有限，因此主要针对 built-in action。

---

# 20. Confirmation

危险命令：

```bash
fleetsh reboot @all
```

应该显示：

```text
This will reboot 23 hosts.

Continue? [y/N]
```

自动化可：

```bash
--yes
```

---

# 21. 配置位置

跨平台：

Windows：

```text
%APPDATA%\fleetsh\
```

Linux：

```text
~/.config/fleetsh/
```

macOS：

```text
~/Library/Application Support/fleetsh/
```

例如：

```text
config.toml
hosts.toml
known_hosts
```

Secret 不放里面。

---

# 22. Import OpenSSH Config

非常值得做。

例如：

```bash
fleetsh import ~/.ssh/config
```

把：

```sshconfig
Host hk1
    HostName 1.2.3.4
    User root
    Port 2222
    IdentityFile ~/.ssh/hk
    ProxyJump jump1
```

转换成本工具里的 host。

也可以允许：

```toml
use_openssh_config = true
```

让工具读取现有 SSH config。

这是减少迁移成本的重要功能。

---

# 23. Export

必须方便用户逃离你的工具。

```bash
fleetsh export
```

输出普通 TOML/JSON。

最好还能：

```bash
fleetsh export ssh-config
```

生成：

```sshconfig
Host hk1
    HostName 1.2.3.4
    User root
```

开源工具不要做数据锁定。

---

# 24. Secret Redaction

日志中必须自动遮蔽：

```text
password
proxy password
sudo password
private key
token
```

例如：

```text
proxy=socks5://foo:******@127.0.0.1:1080
```

---

# 25. 第一版明确“不做”

为了避免最后重新造出 Ansible，MVP 不做：

```text
复杂 YAML playbook
desired state configuration
template language
Jinja
role
collection
package ecosystem
remote agent
central daemon
Web UI
scheduler
Terraform replacement
Kubernetes management
service discovery
CMDB
```

原则：

> 用户想执行命令，就执行命令。

而不是：

> 建立一个完整 infrastructure automation DSL。

---

# 26. 推荐 MVP

第一版只做：

```text
fleetsh add
fleetsh edit
fleetsh rm
fleetsh ls
fleetsh show

fleetsh ssh

fleetsh exec
fleetsh run

group/tag

password
SSH key
ssh-agent

OS credential store

ProxyJump
SOCKS5

parallel execution

timeout

host-key verification

JSON output
```

再加两个 built-in：

```text
fleetsh update
fleetsh reboot
```

这已经是一个非常完整的 v0.1。

---

# 27. v0.2

再增加：

```text
HTTP CONNECT proxy
reboot-required
status dashboard
custom command aliases
sudo credential
OpenSSH config import
config export
retry
serial / rolling execution
```

---

# 28. v0.3

再考虑：

```text
file upload
file download

scp / sftp

port forwarding

OS-aware command profiles

command history

execution logs

shell completion

PowerShell completion
Bash completion
Zsh completion

TUI
```

---

# 29. 推荐技术栈

如果目标是：

```text
single binary
cross platform
SSH
proxy
credential management
```

我会选：

```text
Go
```

而不是 Rust。

原因不是 Rust 做不了，而是这个项目主要是：

```text
I/O
SSH
CLI
config
credential store
network proxy
concurrency
```

Go 非常合适，而且开发成本更低。

核心大致：

```text
Go
├── cobra / urfave/cli
├── golang.org/x/crypto/ssh
├── golang.org/x/net/proxy
├── TOML
├── OS keyring abstraction
└── goroutines
```

架构：

```text
CLI
 │
 ├── Inventory
 │
 ├── Credential Store
 │
 ├── SSH Connection Builder
 │      ├── direct
 │      ├── ProxyJump
 │      ├── SOCKS5
 │      └── HTTP CONNECT
 │
 ├── Executor
 │      ├── single
 │      └── parallel
 │
 ├── Actions
 │      ├── exec
 │      ├── update
 │      ├── reboot
 │      └── status
 │
 └── Output
        ├── terminal
        └── JSON
```

---

# 30. 最终 UX

我认为项目做到下面这个程度就已经非常有价值：

```bash
# 查看服务器
fleetsh ls

# 登录
fleetsh ssh hk1

# 执行
fleetsh exec hk1 "uptime"

# 所有服务器
fleetsh exec @all "df -h"

# 宝塔服务器
fleetsh exec @bt "bt update"

# 系统更新
fleetsh update @all

# 哪些需要重启
fleetsh reboot-required @all

# 分批重启
fleetsh reboot @all --parallel 2

# 走 SOCKS 的服务器
fleetsh ssh special1

# JSON
fleetsh status @all --json
```

用户不需要知道：

```text
inventory DSL
playbook
role
collection
Python environment
module
facts
Jinja
```

只需要知道：

> 我有一堆服务器，我要连它们、管它们、在它们上执行东西。

这应该是整个项目最重要的产品边界。