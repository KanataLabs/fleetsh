---
title: 架构
permalink: /zh/architecture/
lang: zh-CN
locale: zh
page_key: 'architecture/'
---

# 架构

命令行层负责参数和呈现。各包分别负责主机清单、凭据、传输、执行和内置动作。

| 路径 | 职责 |
| --- | --- |
| `cmd/fleetsh/` | 入口、版本、响应中断并取消 |
| `internal/cli/` | Cobra 命令、交互提示、JSON 和终端输出 |
| `internal/config/` | 独占创建初始配置 |
| `internal/inventory/` | 严格解析 TOML、校验、选择器、带锁原子保存 |
| `internal/credentials/` | 系统凭据库、隐藏输入、已知凭据脱敏 |
| `internal/transport/` | SSH 认证、Agent、跳板、SOCKS5/HTTP/HTTPS 代理、主机密钥、保活 |
| `internal/executor/` | 工作池、超时、输出限制、PTY、sudo |
| `internal/actions/` | Linux 资源快照、系统更新、重启验证 |
| `internal/forwarding/` | 本地/远程/动态隧道、连接数限制与取消清理 |
| `internal/testutil/` | 本地 SSH、Agent 和代理测试环境 |
| `docs/` | Jekyll 文档源码，通过 gh-pages 发布 |

## 连接生命周期

在并发工作前解析凭据和 Agent 密钥，避免交互提示交错。
每个目标拥有独立的 SSH 客户端和跳板链。
所有网络操作都支持取消，并受连接超时限制。
未知主机密钥需要确认；变化的密钥会被拒绝。
每 15 秒发送保活探测，无响应时关闭连接。

连接后启动独立的命令超时。取消时关闭 SSH 客户端，
保留标准输出、标准错误和远程退出状态。
每个输出流最多保留 4 MiB，并标记截断。
工作池限制同时进行的目标操作；排序后的清单选择结果保证输出顺序稳定。

## 状态修改

进程级协作文件锁保护遵守锁协议的清单写入。
校验后的 TOML 写入私有临时文件，同步后重命名。
命令行编辑会重写注释。主机密钥信任使用独立锁，
并在确认后重新校验身份，处理并发写入。

执行任何更新前，先探测全部可执行目标。
重启记录原启动标识，发送重启命令后重新连接，
直到确认启动标识变化且健康探测成功。

## 退出状态

- 0：全部可执行目标成功；仅控制台目标可能跳过。
- 1：远程命令失败、命令超时或取消。
- 2：本地用法、配置或凭据准备错误。
- 3：连接、认证、代理或主机密钥失败，包括连接超时。

混合远程结果中，1 优先于 3。交互 shell 非零退出返回 1。
不会自动重试命令。

## 依赖与验证

Cobra、go-toml/v2、x/crypto/ssh、x/net/proxy、x/term、
go-keyring、go-winio 和 flock 固定在 go.mod/go.sum 中。
核心 SSH 不调用 OpenSSH 可执行文件。
macOS Keychain 使用系统 `security` 工具，通过标准输入传送凭据；
Linux 使用 Secret Service 和 D-Bus。

测试覆盖本地 SSH 服务、加密密钥、Unix 套接字和 Windows Agent 管道、
严格信任、目标与跳板身份、认证代理、超时、sudo 输入、并发限制、
输出限制、更新确认和重启身份。

持续集成运行竞态检测，并构建六种不依赖 CGO 的目标。
原生凭据集成测试使用隔离系统凭据库中的临时测试条目。
