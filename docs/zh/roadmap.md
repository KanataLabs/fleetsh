---
title: 路线图
permalink: /zh/roadmap/
lang: zh-CN
locale: zh
page_key: 'roadmap/'
---

# 路线图

当前版本为 **v0.1.0-beta.1**。
[发行说明](https://github.com/KanataLabs/fleetsh/releases/tag/v0.1.0-beta.1)

## 已有功能

- 主机清单、多分组、标签，以及终端中的添加和编辑菜单。
- SSH 密码、密钥、agent、代理和跳板连接；保存的密码使用系统凭据库。
- 连接诊断、存活检查、Linux 资源快照和持续监控。
- 批量命令、常用命令、软件包更新、重启需求检查和带健康验证的分批重启。
- 本地、远程和动态端口转发，以及单文件 SFTP 上传、下载。
- 有限字段的 OpenSSH 导入、TOML／JSON／OpenSSH 导出，以及 shell 补全。

用法与限制见[日常操作](../operations/)。

## 后续工作

- 增加真实终端／PTY 和长时间 SSH 会话的验证。
- 根据操作系统选择保存的命令。
- 可选的命令历史，支持脱敏和保留时间。
- 明确连接重试规则，避免意外重复执行远程命令。

## 发布检查

发布前运行 Go 单元测试、本地 SSH／SFTP 回归测试、`go vet` 和 npm 启动器测试。
CI 还会在 Windows、macOS、Linux 上检查竞态、系统凭据库和编译。
发行包附带许可证和 SHA-256 校验值。

测试使用本地数据，不修改你的主机清单，也不连接你的 VPS。
