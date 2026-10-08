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
OpenSSH 参数透传和端口转发尚未实现。

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
