# fleetsh

面向个人开发者和小团队的轻量 VPS 管理工具。Go 单个二进制文件，在 Windows、
macOS、Linux 上运行；无需远程 Agent 或中心服务器。

[文档站](https://kanatalabs.github.io/fleetsh/zh/) · [安装及全局 PATH 注册](docs/zh/installation.md) · [English](README.md) · [日本語](https://kanatalabs.github.io/fleetsh/ja/)

**当前为开发版。** 已实现主机管理、SSH、系统凭据、跳板/SOCKS5、并发命令、更新和
重启验证；尚未发布稳定版。

## 安装与 PATH

已有 Node.js 22 或更新版本时，可使用单个 npm 包：

```sh
npx fleetsh version
npx fleetsh stats
npm install -g fleetsh
```

npm 包作为长期使用的启动器自动跟随 GitHub Release，优先正式版，
没有正式版时使用预览版。以后 Go 更新无需再发 npm 包；仅启动器自身更新才发 npm。
没有运行时 npm 依赖、安装钩子或平台子包，npm 与 Go 程序的版本独立。

命令行可选择原版或固定某个 Go 版本：

```sh
npx fleetsh --release bundled version
npx fleetsh --release v0.1.0-alpha.1 stats
npx fleetsh --release preview stats
npx fleetsh --refresh version
npx fleetsh --offline stats
npx fleetsh --launcher-help
```

npm 的 `latest` 标签提供独立的启动器 `0.1.0`，它下载的 Go 程序仍可能是预览版。

旧 npm 启动器 `0.1.0-alpha.1` 为固定版本模式，先更新一次至
`0.1.0-alpha.2` 或更新版本。发行信息默认缓存一小时，详细策略见
[安装文档](docs/zh/installation.md)。直接运行二进制不需要 Node.js；
也可从[发行页](https://github.com/KanataLabs/fleetsh/releases)下载，或从源码构建：

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go build -trimpath -o dist/fleetsh ./cmd/fleetsh
```

需要 Go 1.27 或更新版本。Windows 构建输出使用 `dist/fleetsh.exe`。
也可 `go install github.com/KanataLabs/fleetsh/cmd/fleetsh@latest`。

[安装文档](docs/zh/installation.md) 提供可直接复制的方案：

- Windows：用户 PATH 注册，无需管理员；另附系统级 PATH 注册。
- macOS：`~/.local/bin` + Zsh 配置；系统级使用 `/usr/local/bin`。
- Linux：`~/.local/bin` + Bash/Zsh 配置；系统级使用 `/usr/local/bin`。

安装后运行 `fleetsh version` 验证，可在任意目录使用。

## 一键监控与 SSH 隧道

```sh
fleetsh alive
fleetsh stats
fleetsh stats '@web' --parallel 5 --timeout 15s
fleetsh forward hk1 --dry-run
fleetsh forward hk1 web
```

`alive` 检查 SSH 认证和执行是否可用；`stats`（别名 `monitor`）一次打印 Linux
CPU、内存、swap、当前用户和磁盘空余，默认选择所有主机，无需安装远程组件。
在 `[defaults]` 设置全局 SSH 代理；单机可覆盖，或用 `proxy = "direct"` 绕过。
支持 SOCKS5、HTTP/HTTPS CONNECT。`[[hosts.ALIAS.forwards]]` 配置本地、远程、
动态 TCP 转发，详见 [代理继承与端口转发](docs/zh/configuration.md#proxies)。

## 使用

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups asia,web
fleetsh ssh hk1 --connect-timeout 60s
fleetsh exec '@web' "uptime" --parallel 10
fleetsh exec hk1,sg1 "df -h" --json
fleetsh update '@all' --dry-run --sudo
fleetsh reboot '@all' --parallel 2 --sudo --wait-timeout 5m
```

首次连接需要核实并确认指纹，指纹变化始终拒绝；自动化不静默信任未知主机。
更新和重启需要确认，自动化显式使用 `--yes`。重启必须看到新的 boot ID 和健康探测成功。
内置更新/重启动作面向 Linux VPS。

`add --auth password` 未指定凭据引用时会直接询问并自动保存密码；
用 `edit HOST --save-password` 重新保存。系统条目带 fleetsh 命名空间和来源备注，
自动引用以 `fleetsh-ssh-` 开头。`edit HOST --add-groups web,production` 与
`--remove-groups GROUPS` 可在保留其他分组的同时增删成员关系。

密码、密钥口令和代理密码通过隐藏输入或系统凭据库读取，TOML 只保存引用。
配置及命令细节见 [快速开始](docs/zh/getting-started.md) 和 [配置](docs/zh/configuration.md)。
所有命令支持 `--config PATH`；`fleetsh COMMAND --help` 可查看行为说明、选项和示例。
`fleetsh docs` 列出十一个英文离线指南；`fleetsh docs config`、`fleetsh docs passwords`、
`fleetsh docs groups` 和 `fleetsh docs patching` 分别说明配置、密码、分组和批量补丁。
帮助和离线指南无需配置文件或网络；在线文档继续提供独立的中、英、日文版本。

VPS 配置保存在本机当前用户的配置目录：Windows 为 `%APPDATA%\fleetsh\config.toml`，
macOS 为 `~/Library/Application Support/fleetsh/config.toml`，
Linux 为 `$XDG_CONFIG_HOME/fleetsh/config.toml` 或 `~/.config/fleetsh/config.toml`。
[存储位置与备份迁移](docs/zh/configuration.md#storage) 说明自定义清单、主机信任文件及凭据迁移。

文档源文件位于 main 的 `docs/`，由持续集成构建，再通过组织 App 发布到 `gh-pages`。
项目采用 **GPL-3.0-only**，许可证全文见 [LICENSE](LICENSE)。

使用案例：[给所有 VPS 打补丁](docs/zh/cases/patch-all-vps.md)，涵盖预演、apt 批量更新、结果检查和按需重启。
