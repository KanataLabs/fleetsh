---
title: 概览
permalink: /zh/
lang: zh-CN
locale: zh
page_key: ''
---

# 管理 VPS 主机

fleetsh 用于保存 VPS 连接信息、登录 SSH 和批量运行命令。
支持 Windows、macOS、Linux，密码保存在系统凭据库。

> 当前版本：**v0.1.0-beta.1**。[发行说明](https://github.com/KanataLabs/fleetsh/releases/tag/v0.1.0-beta.1)。

[安装与 PATH 注册](installation/) · [快速开始](getting-started/) ·
[关于 fleetsh](product/) · [路线图](roadmap/)

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
npx fleetsh version
npx fleetsh stats
npm install -g fleetsh
```

启动器自动跟随 GitHub Release。命令前加 `--release bundled` 使用原版，
或 `--release vVERSION` 固定 Go 版本。详见 [npx/npm 运行和版本选择](installation/)。

## 开源开发

项目位于 [KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh)，
采用 GPL-3.0-only 许可证。了解 [架构](architecture/) 和 [安全设计](security/)。
