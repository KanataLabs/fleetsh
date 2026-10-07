# fleetsh

面向个人开发者和小团队的轻量 VPS 管理工具。使用 Go 编写，目标是单个二进制文件，
在 Windows、macOS、Linux 上运行；不安装远程 Agent，也不要求中心服务器。

[文档站](https://kanatalabs.github.io/fleetsh/) · [English](README.md) · [产品设计](docs/product.md) · [路线图](docs/roadmap.md)

**当前状态：项目基础阶段。** 已实现 `help`、`version`、`init`，可创建不会覆盖已有文件的
配置示例。SSH、凭据管理、远程执行等功能尚未实现，也尚未发布 v0.1。

## 本地运行

需要 Go 1.27 或更新版本：

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go run ./cmd/fleetsh help
go run ./cmd/fleetsh version
go run ./cmd/fleetsh init
go test ./...
```

构建：`go build -o dist/fleetsh ./cmd/fleetsh`。Windows 输出文件名使用 `fleetsh.exe`。
默认配置目录采用系统用户配置目录下的 `fleetsh/config.toml`；
也可运行 `fleetsh init --config ./config.toml`。

## 目标体验（规划中）

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec @web "uptime" --parallel 10
fleetsh update @all --dry-run
fleetsh reboot @all --parallel 2
```

v0.1 规划包含主机资产、分组/标签、SSH 密码/密钥/Agent、系统凭据存储、
ProxyJump/SOCKS5、严格主机密钥校验、并发执行、JSON 输出、更新和重启。
HTTP CONNECT、导入导出等安排在后续版本，详见路线图。

配置中只保存凭据引用，禁止明文保存密码或私钥。工具边界是“连接服务器、执行命令”，
不引入 YAML playbook、远程 Agent、中心守护进程或配置管理 DSL。

文档源文件位于 main 分支的 `docs/`，发布到 `gh-pages` 分支。
项目采用 **GPL-3.0-only**，许可证全文见 [LICENSE](LICENSE)。
