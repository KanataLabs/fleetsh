---
title: 配置
permalink: /zh/configuration/
lang: zh-CN
locale: zh
page_key: 'configuration/'
---

# 配置

`fleetsh init` 会写入带注释的示例。通过命令行添加主机，或直接编辑 TOML。
严格解析会拒绝未知字段和误写的明文密码。
命令行修改使用文件锁和原子替换，并会重写注释。

```toml
[defaults]
connect_timeout = "10s"
command_timeout = "30m"
parallel = 10

[hosts.hk1]
host = "hk1.example.com"
port = 22
user = "ubuntu"
auth = "key"
key = "~/.ssh/id_ed25519"
groups = ["asia", "web"]
tags = ["production"]

[hosts.sg1]
host = "sg1.example.com"
user = "ubuntu"
auth = "password"
credential = "sg1-login"
sudo_credential = "sg1-sudo"
groups = ["asia"]

[hosts.jump-hk]
host = "jump.example.com"
user = "ubuntu"
auth = "agent"

[hosts.private1]
host = "10.0.0.10"
user = "ubuntu"
auth = "agent"
proxy_jump = "jump-hk"

[hosts.proxy1]
host = "target.example.com"
user = "ubuntu"
auth = "agent"
proxy = "socks5://127.0.0.1:1080"
proxy_credential = "local-proxy"

[hosts.console1]
host = "console.example.com"
connection = "console-only"
```

主机默认端口为 22，连接类型为 `ssh`，认证类型为 `agent`。
SSH 主机必须指定用户名，密钥认证必须指定密钥路径。
密钥路径相对于当前工作目录解析，也支持展开 `~/`。
未指定凭据引用时，可交互输入密码和密钥口令。

别名、分组、标签和引用名称允许字母、数字、点、下划线和连字符，
必须以字母或数字开头，最多 128 个字符。别名 `all` 为保留名称。
并发数范围为 1 至 256；超时长度必须为正数。

## 选择器

| 选择器 | 含义 |
| --- | --- |
| `hk1` | 一台主机 |
| `hk1,sg1` | 指定多台主机，去重并排序 |
| `all` / `@all` | 清单中的全部主机 |
| `@asia` | asia 分组中的主机 |
| `--tag production` | 按标签筛选已选主机 |

未知主机、未知分组以及空的远程目标集合会返回本地错误。
仅控制台资产会出现在列表中，远程操作时跳过。

## 凭据库

`credential`、`proxy_credential` 和 `sudo_credential` 仅保存名称。
使用 `fleetsh credential add REFERENCE`、`ls` 和 `rm REFERENCE` 管理。
替换已有凭据必须指定 `add --replace`。

凭据值使用 Windows Credential Manager、macOS Keychain 或 Linux Secret Service。
Linux 需要运行中的用户 D-Bus 会话，以及已解锁的 Secret Service 实现，
例如 GNOME Keyring。凭据库不可用时会报错，不会回退到明文存储。

`credential ls` 仅列出当前清单跟踪或主机引用的凭据，不枚举系统中的其他凭据。
顶层 `credentials` 数组记录通过当前清单添加的名称。
从凭据库删除条目不会删除主机引用，请按需修改相关主机。

## 代理与 SSH Agent

使用 `proxy_jump` 指向清单中的跳板主机，或使用 `proxy` 指定 SOCKS5，
二者不能同时设置。循环跳板和不存在的跳板主机会被拒绝。
目标和跳板主机都要验证主机密钥。SOCKS5 会在代理端解析目标域名。

需要 SOCKS5 认证时，通过 `credential add` 保存 `username:password`，
并设置 `proxy_credential`；代理 URL 不允许包含用户名和密码。

Unix 使用 `SSH_AUTH_SOCK`。Windows 默认使用
`\\.\pipe\openssh-ssh-agent`，也可通过 `SSH_AUTH_SOCK` 指定套接字或命名管道。
Agent 中至少需要加载一把密钥。

HTTP CONNECT、自定义命令别名、OpenSSH 导入导出和端口转发见 [路线图](../roadmap/)。

## 文件与权限

| 操作系统 | 默认清单 |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\config.toml` |
| macOS | `~/Library/Application Support/fleetsh/config.toml` |
| Linux | `$XDG_CONFIG_HOME/fleetsh/config.toml`，或 `~/.config/fleetsh/config.toml` |

相邻的 `known_hosts` 保存已信任的公钥。
Unix 创建文件使用 0600 权限，新建目录使用 0700。
Windows 使用用户目录的访问控制列表。
清单和 `known_hosts` 必须是普通文件，符号链接会被拒绝。
凭据值保留在操作系统凭据库。
