---
title: 给所有 VPS 打补丁
description: 控制并发执行 apt update 和 apt upgrade，检查每台 VPS 的更新结果。
permalink: /zh/cases/patch-all-vps/
lang: zh-CN
locale: zh
page_key: 'cases/patch-all-vps/'
---

# 给所有 VPS 打补丁

你有多台 Debian 或 Ubuntu VPS，希望统一执行
`apt update && apt upgrade`。fleetsh 可以选择整组主机、预览更新、
控制同时执行的数量，并检查每台主机的结果。

## 1. 准备主机清单

先 [安装并注册 PATH](../../installation/)，再添加主机。以下示例使用密钥登录：

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups apt
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups apt
fleetsh ls '@all'
```

通过独立来源核实指纹后，逐台交互连接并建立信任：

```sh
fleetsh ssh hk1 --connect-timeout 60s
```

检查连接后退出远程 shell，继续处理其他别名。
无人值守命令会拒绝未知或变化的主机密钥。
PowerShell 中为 `@all` 等分组选择器加引号；同样写法适用于 Bash 和 Zsh。

下文假设 SSH 用户有 sudo 权限。没有 `sudo_credential` 时使用免密 sudo。
需要密码时，通过隐藏输入保存凭据并关联到主机：

```sh
fleetsh credential add hk1-sudo
fleetsh edit hk1 --sudo-credential hk1-sudo
```

其他主机按需配置。直接以 root 登录时，省略 `--sudo`。

## 2. 预览更新

```sh
fleetsh update '@all' --dry-run --sudo --parallel 5
```

预演会连接并识别每台主机的系统，展示计划命令，但不更新软件包。
Debian/Ubuntu 使用：

```sh
DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
```

这是刷新索引、升级已安装软件包的脚本形式。
`&&` 保证索引刷新成功后才执行升级。
它安装所有可用升级，包括安全更新和普通更新，并非只安装安全补丁。

混合系统清单中，`update` 会为各系统选择支持的软件包管理器。
任意预检查失败或系统不受支持，会阻止全部目标执行更新。
仅控制台资产会跳过。

只更新清单中的 apt 分组时，将后续命令里的 `'@all'` 替换为 `'@apt'`。

## 3. 应用补丁

```sh
fleetsh update '@all' --sudo --parallel 5 --timeout 30m
```

检查显示的主机别名后确认。最多同时更新五台。
改用 `--serial` 替代 `--parallel 5` 可逐台执行。
超时针对每台主机的更新命令。
fleetsh 不会自动重试失败命令，也不会在部分主机失败后回滚软件包。
软件包更新可能重启服务，请在选定的维护窗口执行。

### 直接执行 apt 命令

全部所选主机都是 Debian/Ubuntu 时，也可以直接执行 apt：

```sh
fleetsh exec '@all' "DEBIAN_FRONTEND=noninteractive apt update && DEBIAN_FRONTEND=noninteractive apt upgrade -y" --sudo --parallel 5 --timeout 30m
```

`--sudo` 覆盖整个 shell 命令，包括 `&&` 两侧。
批量执行没有交互标准输入，因此用 `-y` 确认软件包升级。
`exec` 会立即执行，不提供内置更新的系统预检查、预演和确认。
脚本中优先使用 apt-get，内置动作也采用它。
软件包特有的交互提示仍可能导致无人值守升级失败；
重新执行前请检查对应主机的标准错误。

## 4. 检查结果与自动化

终端显示每台主机的成功或失败、标准输出和标准错误，以及汇总。
退出码 0 表示全部可执行目标成功；1 表示远程命令失败、超时或取消；
2 表示本地配置或凭据准备错误；3 表示连接、认证、代理或主机密钥错误。
某台主机刷新索引失败后，不会在该主机上执行升级。

明确批准的自动化任务可用 `--yes --json`：

```sh
fleetsh update '@all' --sudo --parallel 5 --timeout 30m --yes --json > patch-results.json
```

JSON 包含 `results` 和 `summary`，记录逐台错误。
同时检查进程退出码。PowerShell 在命令后立即读取 `$LASTEXITCODE`；
Bash/Zsh 使用 `$?`。修复原因后，只重试失败的别名。

## 5. 检查是否需要重启

Debian/Ubuntu 可检查重启需求标记：

```sh
fleetsh exec '@all' "if test -f /var/run/reboot-required; then echo REBOOT_REQUIRED; else echo NO_REBOOT_MARKER; fi" --parallel 5
```

没有标记只表示发行版未通过该文件提示重启，
不能证明所有服务都已加载更新后的库。
选出需要重启的别名，然后预览并执行：

```sh
fleetsh reboot hk1,sg1 --dry-run --sudo
fleetsh reboot hk1,sg1 --sudo --parallel 2 --wait-timeout 5m
```

确认所选主机。成功必须看到新的 Linux 启动标识和 uptime 健康探测通过。
打补丁动作本身不会自动重启。

命令细节见 [快速开始](../../getting-started/)，
凭据和主机密钥规则见 [安全设计](../../security/)。
