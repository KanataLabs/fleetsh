---
title: 日常操作
permalink: /zh/operations/
lang: zh-CN
locale: zh
page_key: 'operations/'
---

# 日常操作

以下功能已在开发版实现。CLI 帮助和离线指南默认英语。
使用自定义清单时，每条命令都需指定相同的 `--config PATH`。

<a id="interactive"></a>

## 交互式添加与编辑

在终端中运行 `npx fleetsh add hk1`，程序会询问缺少的地址、用户或密钥路径，
再进行隐藏的密码输入。新增主机默认使用 SSH 密码认证，也可以完整指定参数。

`npx fleetsh edit hk1` 打开菜单，可修改地址、用户、本地保存的密码、分组、
标签、备注或认证方式。取消不会保存。分组和标签输入会替换列表，空输入会清空。
脚本和 `--json` 模式必须提供修改参数，不会弹出菜单。

```sh
npx fleetsh edit hk1
npx fleetsh edit hk1 --save-password
npx fleetsh edit hk1 --add-groups production
npx fleetsh show hk1
```

修改密码只替换本地保存值，不会修改服务器密码。保存成功会明确显示主机名。

<a id="diagnostics"></a>

## 排查连接问题

```sh
npx fleetsh doctor hk1
npx fleetsh doctor hk1 --connect-timeout 30s --json
```

报告显示清单路径、实际代理／跳板路线、本地认证准备、网络连接、主机密钥、
SSH 认证及远程健康检查。诊断不会修改配置或自动信任新密钥。
遇到未知密钥，应独立核对指纹，再用 `fleetsh ssh hk1` 明确接受。

错误会说明问题并给出下一步命令。JSON 保留结构化检查项和现有退出码含义。
保存错密码时，执行 `fleetsh edit hk1 --save-password`。

<a id="monitoring"></a>

## 持续查看资源占用

```sh
npx fleetsh stats '@all' --table
npx fleetsh stats '@all' --watch 5s
npx fleetsh stats '@web' --watch 5s --json
```

表格显示主机状态、CPU、内存、swap、当前用户及根文件系统空余。
每轮显示时间戳；其他磁盘仍可通过普通 `stats` 输出查看。一台失败不会隐藏其他主机。

间隔至少一秒，从上一轮结束后开始计算，探测不会重叠。Ctrl+C 停止。
JSON 持续输出每行一份报告，包含 `timestamp`、`results`、`summary`。
配置和凭据在启动时加载，修改后需重启监控命令。无需安装远程代理。

<a id="saved-commands"></a>

## 保存常用命令

在清单中加入：

```toml
[commands.disk]
command = "df -h"
timeout = "30s"

[commands.docker-update]
command = "docker compose pull && docker compose up -d"
sudo = true
timeout = "10m"
```

```sh
fleetsh run '@web' docker-update --dry-run
fleetsh run '@web' docker-update --parallel 2
fleetsh run hk1 disk --json
```

选择器与命令名必填。`--sudo`、`--timeout` 可覆盖保存的默认值。
预览不会连接服务器，命令不会自动重试。不要在命令定义中保存密码或 token。

<a id="completion"></a>

## 开启自动补全

先安装二进制，或运行 `npm install -g fleetsh`，使 `fleetsh` 位于 PATH。
支持 `bash`、`zsh`、`fish`、`powershell`。

```powershell
fleetsh completion powershell > fleetsh-completion.ps1
. ./fleetsh-completion.ps1
```

需要每次启动生效时，在 PowerShell profile 中加入使用绝对路径的点加载语句。
Bash 可加载生成的脚本；Zsh 将 `_fleetsh` 放入 `fpath` 并初始化 `compinit`；
Fish 将脚本放入 `~/.config/fish/completions/fleetsh.fish`。

补全只读取本地主机、分组及命令名，不读取秘密值或连接服务器。
受支持入口是 `fleetsh` 命令；`npx fleetsh` 的补全取决于 shell。

<a id="maintenance"></a>

## 检查重启需求与分批重启

```sh
fleetsh update '@all' --sudo --check-reboot
fleetsh reboot-required '@all'
fleetsh reboot '@web' --sudo --batch-size 2 --wait-timeout 5m
```

Debian／Ubuntu 检查 reboot-required 标记，其他系统在安装后可使用
`needs-restarting -r`。不支持或无法检查时明确显示“未知”。更新不会自动重启。

整批的新启动标识与健康检查通过后才开始下一批。上一批失败会停止后续重启，
未执行主机显示跳过。仍需确认，自动化可明确指定 `--yes`。
批大小与同时执行的 worker 数量分别控制。

<a id="import-export"></a>

## 导入 OpenSSH 配置与导出主机信息

```sh
fleetsh import ~/.ssh/config --dry-run
fleetsh import ~/.ssh/config --user ubuntu
fleetsh export --format toml --output inventory.toml
fleetsh export --format json
fleetsh export --format ssh-config --output ssh-config
```

导入支持明确的别名、通配符默认值、`HostName`、`User`、`Port`、
单个 `IdentityFile` 与单个跳板别名 `ProxyJump`，遵循 OpenSSH 首个匹配值优先。
没有用户设置时需指定 `--user USER`。同名主机会停止导入，已有主机不会被覆盖。

`Include`、`Match` 会停止导入；依赖 `ProxyCommand`、多跳／地址形式的 ProxyJump
或不支持的展开变量的主机会跳过并解释。其他不支持的设置会列出，不执行本地命令。
导入主机使用密钥或 agent 认证，需要密码认证时用 `edit HOST` 修改。

导出不包含系统凭据值、凭据引用或常用命令定义，但会保留用户填写的备注与连接设置。
OpenSSH 输出会提示无法表达的代理 URL 与转发配置。
输出文件采用 UTF-8，已有文件需明确指定 `--overwrite`。

<a id="transfers"></a>

## SFTP 上传与下载

```sh
fleetsh upload '@web' ./script.sh /tmp/script.sh --dry-run
fleetsh upload '@web' ./script.sh /tmp/script.sh
fleetsh download hk1 /var/log/syslog ./syslog.txt
fleetsh download '@web' /var/log/syslog ./logs
```

复用已有 SSH 登录、代理和主机密钥设置。只传输一个普通文件，不支持递归目录或
`--sudo`；上传目标目录必须存在且对 SSH 用户可写。

单机下载可指定本地文件或已有目录，多机下载使用各自的别名目录并附短哈希，
避免文件系统名称冲突。预览会显示确切路径，不创建文件。
已有目标文件需明确指定 `--overwrite`。中断可能留下未完成文件，重试前请检查。
