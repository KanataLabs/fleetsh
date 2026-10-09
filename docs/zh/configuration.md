---
title: 配置
permalink: /zh/configuration/
lang: zh-CN
locale: zh
page_key: 'configuration/'
---

# 配置

<a id="storage"></a>

## VPS 配置信息存放在哪里

主机清单保存在运行 fleetsh 的电脑上，属于当前操作系统用户。
一个 `config.toml` 保存 VPS 地址、SSH 用户名和端口、分组、标签、代理设置、
默认参数及凭据引用名称；每台 VPS 对应一个 `[hosts.别名]` 条目。
把程序加入 PATH 不会改变配置位置，在不同工作目录运行时仍使用同一份默认清单。

| 操作系统 | 默认清单路径 |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\config.toml` |
| macOS | `~/Library/Application Support/fleetsh/config.toml` |
| Linux | 设置了 `XDG_CONFIG_HOME` 时为 `$XDG_CONFIG_HOME/fleetsh/config.toml`，否则为 `~/.config/fleetsh/config.toml` |

Windows 下通常展开为 `C:\Users\<用户名>\AppData\Roaming\fleetsh\config.toml`。
`fleetsh init` 创建文件时会输出路径；已有文件会保留。
初始化后，可以用下面的命令查找默认清单：

Windows PowerShell：

```powershell
$configFile = Join-Path $env:APPDATA "fleetsh\config.toml"
Write-Output $configFile
Get-Item -LiteralPath $configFile
notepad $configFile
```

macOS：

```sh
config_file="$HOME/Library/Application Support/fleetsh/config.toml"
printf '%s\n' "$config_file"
ls -l "$config_file"
```

Linux：

```sh
config_file="${XDG_CONFIG_HOME:-$HOME/.config}/fleetsh/config.toml"
printf '%s\n' "$config_file"
ls -l "$config_file"
```

这些命令显示默认位置。如果运行时指定了 `--config`，应查看该参数所选的文件。

### 指定其他清单

需要使用自定义清单时，每条相关命令都传入 `--config PATH`：

```sh
fleetsh --config "./inventories/production/config.toml" init
fleetsh --config "./inventories/production/config.toml" add web1 --host web1.example.com --user ubuntu --auth agent
fleetsh --config "./inventories/production/config.toml" ls
fleetsh --config "./inventories/production/config.toml" ssh web1
```

相对路径以当前工作目录为基准；`~/` 展开为当前用户的主目录。
包含空格的路径需要加引号。该选项只为本次执行选择一份文件，
不会修改默认位置，也不会合并其他清单。经常切换工作目录时，建议传入绝对路径。
分别放在 `inventories/production/` 和 `inventories/staging/` 等不同目录，
还可以分别保存各环境信任的主机公钥。

### 相关文件与凭据

| 数据 | 存储位置 |
| --- | --- |
| VPS 清单和凭据名称 | 默认 `config.toml`，或 `--config` 指定的文件 |
| 已信任的 SSH 主机公钥 | 所选清单目录中的 `known_hosts`，首次信任主机密钥时创建 |
| SSH 私钥 | 主机 `key` 字段指定的路径，fleetsh 从该处读取 |
| 已保存的密码、私钥口令和代理凭据 | Windows Credential Manager、macOS Keychain 或 Linux Secret Service |
| 文件锁 | 清单旁的 `<清单路径>.lock` 和信任文件旁的 `known_hosts.lock` |

fleetsh 使用清单旁的 `known_hosts`；同目录的清单共用该文件。
它不使用 `~/.ssh/known_hosts`。秘密值保存在当前用户的系统凭据库中，
服务名称为 `fleetsh`。即使清单位于不同目录，相同凭据引用仍指向同一个秘密值。
需要隔离时可分别命名为 `production-web1-login` 和 `staging-web1-login`。

### 备份与迁移到另一台电脑

1. 等待 fleetsh 命令执行完毕，将所选清单及其相邻的 `known_hosts` 复制到目标配置目录。锁文件用于协调写入，无需备份。
2. 单独迁移需要的 SSH 私钥，保留受限访问权限，并按需修改 `key` 路径。
3. 在目标电脑上用 `fleetsh credential add REFERENCE` 重新保存凭据；自定义清单需同时传入 `--config PATH`。复制 TOML 不会复制系统凭据库中的秘密值。
4. 连接前先用 `fleetsh ls` 检查所选清单。没有迁移 `known_hosts` 时，需要重新核实并信任主机指纹。

清单包含主机地址、用户名等基础设施信息，备份应保持私密。
文件权限说明见本文末尾。

## 清单格式

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

`fleetsh add` 默认端口为 22，连接类型为 `ssh`，认证类型为 `password`，并自动保存密码。
已有主机保留原认证方式；为兼容旧清单，TOML 中省略 `auth` 字段仍按 `agent` 处理。
Agent 认证需显式指定 `--auth agent`；私钥文件使用 `--auth key --key PATH`。
旧主机若提示 `SSH agent unavailable`，且你使用密码登录，先运行
`fleetsh edit HOST --auth password`，再运行 `fleetsh ssh HOST`。
SSH 主机必须指定用户名，密钥认证必须指定密钥路径。
密钥路径相对于当前工作目录解析，也支持展开 `~/`。
未指定凭据引用时，可交互输入密码和密钥口令。

别名、分组、标签和引用名称允许字母、数字、点、下划线和连字符，
必须以字母或数字开头，最多 128 个字符。别名 `all` 为保留名称。
并发数范围为 1 至 256；超时长度必须为正数。

<a id="groups"></a>

## 后续修改分组

一台主机可以同时属于多个分组。分组由主机的成员关系决定，无需预先建立独立的分组记录。
添加主机后也可以继续修改：

```sh
fleetsh edit hk1 --add-groups production,monitoring
fleetsh edit hk1 --remove-groups asia
fleetsh edit hk1 --groups web,production
fleetsh edit hk1 --groups ""
fleetsh ls '@production'
```

`--add-groups` 保留已有分组，重复添加会去重。
`--remove-groups` 只移除指定成员关系；主机不属于该组时会忽略。
`--groups` 整体替换分组列表，空值清空全部分组。
不同名称的追加与移除可以同时指定，但不能与 `--groups` 混用，也不能同时追加和移除同一名称。
只修改分组不会要求输入密码，也不会改变主机密码。

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

<a id="passwords"></a>

## 自动保存主机密码

使用 `--auth password` 添加 SSH 主机时，如果没有指定 `--credential`，会通过隐藏输入
询问两次密码。fleetsh 自动生成唯一的 `fleetsh-ssh-ALIAS-RANDOM` 引用，将秘密值存入
系统凭据库，并关联主机。无需先手动执行 `credential add`。

```sh
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth password --groups asia,web
fleetsh edit sg1 --save-password
fleetsh show sg1
```

`edit --save-password` 会使用新引用保存密码，保留原凭据，避免影响共用它的其他主机。
未显式指定引用、切换为密码认证时，也会询问并保存。
可用 `show` 或 `credential ls` 查看引用，不再需要的旧引用可通过 `credential rm REFERENCE` 删除。
删除主机不会自动删除其凭据。

希望每次连接时输入密码，可使用 `--no-save-password`。
编辑时该选项解除主机的凭据引用，但保留系统中已保存的条目。
非交互添加应传入已有的 `--credential REFERENCE`，或显式指定 `--no-save-password`。
非空凭据引用不能与密码保存选项同时指定。

主机设置会在输入密码前完成校验。输入或存储失败不会改变清单；
如果清单写入失败，会删除此次新保存的秘密值。

### 在系统中识别 fleetsh 创建的凭据

| 平台 | 可见标识 |
| --- | --- |
| Windows Credential Manager | 目标名 `fleetsh:REFERENCE`；备注 `Created by fleetsh (KanataLabs). Managed VPS credential.` |
| macOS Keychain | 服务名 `fleetsh`、标签 `fleetsh:REFERENCE`，以及相同的创建来源备注 |
| Linux Secret Service | 服务属性 `fleetsh`；标签以 `fleetsh:REFERENCE` 开头，并包含创建来源说明 |

自动保存和手动 `credential add` 都会为新建、替换的条目添加上述标识。
现有引用与 `fleetsh` 服务命名空间保持兼容，旧条目在替换时补上备注。

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

<a id="proxies"></a>

## 全局与单台主机的 SSH 代理

修改现有 `[defaults]` 段，为 SSH 设置全局代理：

```toml
[defaults]
proxy = "socks5://127.0.0.1:1080"
# proxy_credential = "local-proxy"
```

支持 `socks5://`、`http://`、`https://`，HTTP/HTTPS 通过 CONNECT 建立隧道，
HTTPS 会验证代理证书。三种协议的默认端口分别为 1080、80、443。
URL 不允许包含认证信息、路径、查询参数或片段。需要认证时，通过
`fleetsh credential add local-proxy` 隐藏输入 `username:password`，再设置
`proxy_credential` 引用；不要把密码写进 TOML。

单机未指定代理 URL 时继承全局设置。单机指定 URL 则覆盖全局，并只使用该主机的
`proxy_credential`，不会把全局凭据带到另一个代理地址。继承全局 URL 的主机可以单独
覆盖凭据引用。`proxy = "direct"` 表示绕过全局代理，直接连接 SSH。

```sh
fleetsh edit hk1 --proxy http://127.0.0.1:7890
fleetsh edit hk1 --proxy socks5://127.0.0.1:1080 --proxy-credential local-proxy
fleetsh edit hk1 --proxy direct --proxy-credential ""
fleetsh edit hk1 --proxy "" --proxy-credential ""
```

最后一条恢复继承。代理适用于 `ssh`、`exec`、`alive`、`stats`、`update`、
`reboot` 和 `forward`，在连接时解析；修改其他配置不会把继承值写成单机覆盖。

`proxy_jump` 让目标经过清单里的跳板主机，而不直接使用全局代理；跳板自己仍继承全局
代理，除非另行覆盖。单台主机不能同时设置 `proxy` 和 `proxy_jump`，
循环或不存在的跳板会被拒绝。目标和跳板的 SSH 指纹均需独立验证。

Unix Agent 使用 `SSH_AUTH_SOCK`；Windows 默认使用
`\\.\pipe\openssh-ssh-agent`，也可通过 `SSH_AUTH_SOCK` 指定其他套接字或管道。
Agent 至少需要加载一把可用密钥。离线说明见 `fleetsh docs proxies`。

<a id="forwarding"></a>

## 三种 SSH 端口转发

在已有 SSH 主机的配置下增加命名转发：

```toml
[[hosts.hk1.forwards]]
name = "web"
type = "local"
listen = "127.0.0.1:8080"
destination = "127.0.0.1:80"

[[hosts.hk1.forwards]]
name = "reverse"
type = "remote"
listen = "127.0.0.1:9000"
destination = "127.0.0.1:3000"

[[hosts.hk1.forwards]]
name = "socks"
type = "dynamic"
listen = "127.0.0.1:1080"
```

| 类型 | 在哪里监听 | 从哪里连接目标 |
| --- | --- | --- |
| `local` | 本机 | VPS |
| `remote` | VPS | 本机 |
| `dynamic` | 本机 SOCKS5 入口 | VPS；目标由 SOCKS 客户端请求 |

```sh
fleetsh forward hk1 --dry-run
fleetsh forward hk1 web
fleetsh forward hk1 socks
fleetsh forward hk1
fleetsh forward hk1 --dry-run --json
```

指定名称只启动一条，省略名称则一起启动全部配置。预演仅读取配置，不连接 SSH、
不读取密码、不监听端口。实际转发保持前台运行，Ctrl+C 或断线后关闭监听和连接，
启动失败会撤销全部已启动配置。其他命令不会自动开启转发，普通 `edit` 会保留转发配置。

名称须符合命名规则，且在同一主机内唯一。监听地址须明确写 IP 或 `localhost` 和
0–65535 的端口；0 会分配空闲端口并打印实际地址。目标端口须为 1–65535。
IPv6 使用 `[::1]:1080` 等带方括号的写法。`--connect-timeout` 限制 SSH 连接、
启动及目标拨号时间，已建立的连接没有固定寿命限制；每个会话最多 128 个同时连接。

私有访问使用回环地址监听。显式绑定 `0.0.0.0`/`::` 会让其他机器也能访问。
动态转发提供无认证 SOCKS5 TCP CONNECT，不支持 BIND 和 UDP，域名在 VPS 端解析。
远程监听受 sshd 的 `AllowTcpForwarding`/`GatewayPorts` 策略约束，fleetsh 不修改服务端配置。
离线示例见 `fleetsh docs forwarding`。

自定义命令别名和 OpenSSH 导入导出仍在 [路线图](../roadmap/) 中。

## 文件与权限

清单和主机信任文件的位置见 [上文](#storage)。
Unix 创建文件使用 0600 权限，新建目录使用 0700。
Windows 使用用户目录的访问控制列表。
清单和 `known_hosts` 必须是普通文件，符号链接会被拒绝。
凭据值保留在操作系统凭据库。
