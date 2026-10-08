---
title: 概览
permalink: /zh/
lang: zh-CN
locale: zh
page_key: ''
---

# 一个命令，管理你的 VPS。

fleetsh 是面向个人开发者和小团队的轻量 VPS 管理工具，使用 Go 编写。
一个二进制文件即可在 Windows、macOS 和 Linux 上运行，通过 SSH 连接，
并使用操作系统的凭据库保存密码。

> **当前为开发版。** 已实现主机管理、SSH、凭据、代理、并发命令、更新和重启，
> 并通过本地测试环境验证；尚未发布稳定版。

[安装与 PATH 注册](installation/) · [快速开始](getting-started/) ·
[产品设计](product/) · [路线图](roadmap/)

[使用案例：给所有 VPS 打补丁](cases/patch-all-vps/)

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec '@web' "uptime" --parallel 10
fleetsh update '@all' --dry-run --sudo
fleetsh reboot '@all' --parallel 2 --sudo
```

已有 Node.js 22+ 时，可使用 [npm 包](https://www.npmjs.com/package/fleetsh)：

```sh
npx fleetsh@alpha version
npx fleetsh@alpha stats
npm install -g fleetsh@alpha
```

启动器自动跟随 GitHub Release。命令前加 `--release bundled` 使用原版，
或 `--release vVERSION` 固定 Go 版本。详见 [npx/npm 运行和版本选择](installation/)。

## 设计原则

- 单个二进制文件，无需语言运行时或远程代理。
- 无需中心服务器或基础设施自动化配置语言。
- 使用可迁移的 TOML 主机清单，不保存明文密码。
- 严格验证主机密钥，包括跳板主机。
- 限制并发、设置超时，并为每台主机返回结构化结果。
- 更新和重启需要明确确认，重启后验证启动标识。

## 开源开发

项目位于 [KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh)，
采用 GPL-3.0-only 许可证。了解 [架构](architecture/) 和 [安全设计](security/)。
