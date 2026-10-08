# fleetsh

面向个人开发者和小团队的轻量 VPS 管理工具。Go 单个二进制文件，在 Windows、
macOS、Linux 上运行；无需远程 Agent 或中心服务器。

[文档站](https://kanatalabs.github.io/fleetsh/zh/) · [安装及全局 PATH 注册](docs/zh/installation.md) · [English](README.md)

**当前为开发版。** 已实现主机管理、SSH、系统凭据、跳板/SOCKS5、并发命令、更新和
重启验证；尚未发布稳定版。

## 安装与 PATH

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

密码、密钥口令和代理密码通过隐藏输入或系统凭据库读取，TOML 只保存引用。
配置及命令细节见 [快速开始](docs/zh/getting-started.md) 和 [配置](docs/zh/configuration.md)。
所有命令支持 `--config PATH`；`fleetsh COMMAND --help` 可查看选项。

VPS 配置保存在本机当前用户的配置目录：Windows 为 `%APPDATA%\fleetsh\config.toml`，
macOS 为 `~/Library/Application Support/fleetsh/config.toml`，
Linux 为 `$XDG_CONFIG_HOME/fleetsh/config.toml` 或 `~/.config/fleetsh/config.toml`。
[存储位置与备份迁移](docs/zh/configuration.md#storage) 说明自定义清单、主机信任文件及凭据迁移。

文档源文件位于 main 的 `docs/`，由持续集成构建，再通过组织 App 发布到 `gh-pages`。
项目采用 **GPL-3.0-only**，许可证全文见 [LICENSE](LICENSE)。

使用案例：[给所有 VPS 打补丁](docs/zh/cases/patch-all-vps.md)，涵盖预演、apt 批量更新、结果检查和按需重启。
