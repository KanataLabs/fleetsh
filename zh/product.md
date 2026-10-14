---
title: 关于 fleetsh
permalink: /zh/product/
lang: zh-CN
locale: zh
page_key: 'product/'
---

# 关于 fleetsh

fleetsh 把 VPS 的连接信息保存在本地。你可以用它登录 SSH、在多台机器上运行命令、
查看资源占用、更新软件包或传输文件。

程序使用 Go 编写，可在 Windows、macOS 和 Linux 上运行。被管理的机器只需提供 SSH，
不需要安装 fleetsh。也可以通过 npm 启动。

## 常用操作

| 操作 | 命令 |
| --- | --- |
| 添加主机 | `fleetsh add hk1` |
| 修改设置或本地保存的密码 | `fleetsh edit hk1` |
| 排查登录失败 | `fleetsh doctor hk1` |
| 对分组运行命令 | `fleetsh exec '@web' "uptime"` |
| 持续查看资源占用 | `fleetsh stats '@all' --watch 5s` |
| 执行保存的命令 | `fleetsh run '@web' docker-update` |
| 检查是否需要重启 | `fleetsh reboot-required '@all'` |
| 上传文件 | `fleetsh upload hk1 ./script.sh /tmp/script.sh` |

终端中的 `add`、`edit` 会显示输入提示；脚本中需完整指定参数。
一台主机可以加入多个分组。分组选择器以 `@` 开头，在 PowerShell 中要加引号。

第一次添加主机请看[快速开始](../getting-started/)，其他操作见[日常操作](../operations/)。

## 配置与密码

连接设置保存在本地 TOML 文件中。密码和密钥口令保存在操作系统凭据库，
配置文件只保存引用。

SSH 支持密钥、密码、agent、代理和跳板。本地、远程、动态端口转发使用命名配置。
路径及示例见[配置](../configuration/)，主机密钥和凭据处理见[安全说明](../security/)。

## 当前限制

SFTP 每次传输一个普通文件。OpenSSH 导入只支持帮助中列出的字段，
会提示不支持的设置。资源监控读取 Linux 数据。命令不会自动重试。

剩余工作见[路线图](../roadmap/)。项目位于
[KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh)，采用 GPL-3.0-only 许可证。
