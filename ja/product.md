---
title: 製品設計
permalink: /ja/product/
lang: ja
locale: ja
page_key: 'product/'
---

# 軽量 VPS 管理ツール

> この文書は製品の設計概要です。実装済み機能と計画中の機能の両方を記載しています。
> 現在の実装範囲は[ロードマップ](../roadmap/)を参照してください。
> 当初の優先順位の違いも残しています。
> 以下の提案コマンドと設定は設計例です。現在対応する CLI と設定は、
> [設定](../configuration/)と[使い始める](../getting-started/)を参照してください。

## 1. 製品の位置付け

個人開発者と小規模チームのための軽量 VPS 管理ツールです。

主な目標：

- Python、Node.js、Java のランタイムが不要な単一実行ファイル。
- リモート Agent や必須の中央サーバーが不要。
- Windows、macOS、Linux で動作。
- 複数 VPS の接続情報と認証情報の参照を管理。
- SSH のパスワード、秘密鍵、Agent に対応。
- プロキシと踏み台に対応。
- 単一ホストやグループでコマンドを実行。
- 頻度の高い VPS 保守操作を少数の組み込み操作として提供。
- シンプルで移行しやすく、バックアップしやすい設定。
- 安全な既定値を使い、パスワードを平文で保存しない。

完全な構成管理や Ansible の置換は対象外です。

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec hk1 "uptime"
fleetsh exec all "uptime"
fleetsh exec '@web' "docker ps"
fleetsh update hk1
fleetsh update '@all'
fleetsh reboot hk1
```

## 2. 設計の基本方針

### 2.1 単一実行ファイル

Windows は `fleetsh.exe`、macOS/Linux は `fleetsh` を配布します。
実行に Python、pip、Node.js、Docker、Java、Ruby は不要です。
ssh-agent、Windows Credential Manager、macOS Keychain、Linux Secret Service など、
OS の既存機能を利用できます。可能な限り基本機能を実行ファイル内に保ちます。

### 2.2 Agent 不要

接続先には SSH サーバーが必要ですが、fleetsh の常駐ソフトウェアは不要です。

```text
fleetsh → SSH → VPS
```

SSH がないホストも `connection = "console-only"` として一覧に保存できます。
資産と認証情報の管理に利用し、リモートコマンドは実行しません。

## 3. ホスト管理

別名、ホスト名と IP、ポート、ユーザー名、認証方式、認証情報の参照、
プロキシ、グループとタグ、説明を保存します。

```toml
[hosts.hk1]
host = "1.2.3.4"
port = 22
user = "root"
auth = "password"
credential = "hk1-root"
groups = ["asia", "web"]

[hosts.sg1]
host = "sg1.example.com"
user = "ubuntu"
auth = "key"
key = "~/.ssh/id_ed25519"
groups = ["asia"]

[hosts.cn1]
host = "10.0.0.10"
user = "root"
auth = "password"
credential = "cn1-root"
proxy_jump = "jump-hk"
groups = ["china"]
```

`add`、`edit`、`rm`、`show`、`ls` を提供し、非対話で追加できます。

```sh
fleetsh add hk1 --host 1.2.3.4 --user root --auth password
```

## 4. グループとタグ

グループは重複できます。hk1 は asia と web、sg1 は asia と
ホスティング管理パネル用の bt に所属できます。ホストの明示指定とタグの絞り込みに対応します。

```sh
fleetsh ls '@asia'
fleetsh exec '@asia' "uptime"
fleetsh exec '@bt' "bt update"
fleetsh exec hk1,sg1,jp1 "uptime"
fleetsh exec all "docker ps" --tag web
```

PowerShell では `@` で始まる指定を引用符で囲みます。同じ書式を Bash/Zsh でも使えます。

## 5. 認証

秘密鍵ファイル、暗号化した秘密鍵、ssh-agent、パスワードに対応します。

```toml
auth = "key"
key = "~/.ssh/id_ed25519"
```

Agent 認証は `auth = "agent"`、パスワード認証は `auth = "password"` と
`credential = "hk1-root"` などの参照を使います。設定に `password` の値を保存しません。

## 6. 認証情報の管理

Windows Credential Manager、macOS Keychain、Linux Secret Service を使用します。
Linux では GNOME Keyring や対応する KWallet プロバイダーなどを利用できます。

当初は、AES-256-GCM とユーザー指定または安全に導出したマスターキーを使う、
暗号化したローカルデータベースへの切り替えも検討していました。
これは未実装です。現在は OS ストアが利用できない場合に失敗し、平文保存へ切り替えません。

```sh
fleetsh credential add hk1-root
fleetsh credential ls
fleetsh credential rm hk1-root
```

パスワードの入力と確認は非表示にし、シェル履歴に残さないようにします。

## 7. プロキシ

当初の設計では、以下の 3 種類を含むプロキシ対応を P0 としていました。

### 踏み台

```text
ローカル → jump-hk → cn1
```

ホスト一覧に踏み台を定義し、接続先に `proxy_jump = "jump-hk"` を指定します。
`fleetsh ssh cn1` が設定した経路を自動で利用します。

### SOCKS5

ローカルの Clash や sing-box を含め、`socks5://127.0.0.1:7890` のような URL を使用します。
認証なしとユーザー名・パスワード認証の両方に対応します。
現在は認証値を `proxy_credential` で参照し、URL に認証情報を埋め込みません。

### HTTP CONNECT

`http://127.0.0.1:8080` などの URL で CONNECT トンネルを作ります。
HTTP/HTTPS CONNECT、認証情報参照、全体とホストごとの経路は実装済みです。
[プロキシ設定](../configuration/#proxies)を参照してください。

## 8. SSH

`fleetsh ssh hk1` で通常の SSH クライアントのようにログインします。
PTY、対話シェル、パスワード・公開鍵・Agent 認証、独自ポート、
ホスト鍵検証、接続維持、タイムアウトに対応します。

当初は `fleetsh ssh hk1 -- -L 8080:localhost:80` のような
OpenSSH 引数の引き渡しも提案していました。
引数の引き渡しは今後の予定です。ローカル・リモート・動的転送は名前付き設定と
`fleetsh forward hk1 [NAME]` で実装済みです。SSH はプロセス内で動作します。

## 9. リモート実行

```sh
fleetsh exec hk1 "uptime"
fleetsh exec '@all' "uptime"
```

既定では並列で実行します。各ホストの結果、所要時間、stdout、stderr を表示します。
ホストごとの出力を区別できるようにします。

## 10. 並列数の制御

```sh
fleetsh exec '@all' "uptime" --parallel 10
fleetsh exec '@all' "uptime" --serial
```

既定の並列数は 10 です。直列実行は同時実行数を 1 にします。
特に更新と再起動では並列数の制御が必要です。

## 11. 終了コードとエラー

stdout、stderr、リモート終了状態、接続失敗、タイムアウト、認証失敗、プロキシ失敗を保持し、
成功、失敗、スキップの数を報告します。現在の終了コードは次のとおりです。

- 0：すべての実行対象が成功。
- 1：リモートコマンド失敗、タイムアウト、キャンセル。
- 2：ローカルの引数、設定、認証情報の準備エラー。
- 3：接続、認証、プロキシ、ホスト鍵の失敗。

複数種類のリモート失敗では 1 を 3 より優先します。
詳細は[アーキテクチャ](../architecture/)を参照してください。

## 12. sudo

```sh
fleetsh exec hk1 "apt-get update" --sudo
```

パスワードが必要な場合は `sudo_credential = "hk1-root"` などの参照を使います。
パスワードはチャネルの stdin で送信し、
`echo password | sudo -S` のようにログやプロセス引数へ露出させません。

## 13. 組み込み操作

頻度の高い少数の操作を提供し、Ansible のような YAML システムは導入しません。

### 基本的な監視と詳細な状態表示

`fleetsh alive` は SSH 接続確認、`fleetsh stats`（別名 `monitor`）は Linux の
CPU、メモリー、swap、ユーザー、空き容量を表示します。既定は全ホストで、JSON に対応します。

詳細な状態一覧は今後の予定です。

`fleetsh status '@all'` で SSH 接続可否、稼働時間、負荷、ディスク、
メモリー、OS、カーネル情報を表示する予定です。

### update

OS を検出し、Debian/Ubuntu は apt、RHEL/Rocky/AlmaLinux は dnf、
古い CentOS は yum、Arch は pacman を使います。
Debian/Ubuntu の既定動作はパッケージ一覧の取得とアップグレードです。
`--dist` で完全なディストリビューションアップグレードを選択します。

### reboot-required（計画中）

`fleetsh reboot-required '@all'` で再起動の必要性を確認する予定です。
Debian/Ubuntu では `/var/run/reboot-required` などを調べます。

### reboot

再起動を送信し、想定される SSH 切断を許容して再接続を待ち、正常性を確認します。
実装では、成功とする前に Linux 起動 ID の変化も必要とします。
既定で複数ホストの並列数を制限します。

```sh
fleetsh reboot '@all' --parallel 2
```

## 14. 独自のコマンドと別名（計画中）

単純な再利用可能コマンドを TOML に保存する予定です。

```toml
[commands.bt-update]
command = "bt update"

[commands.docker-update]
command = "docker compose pull && docker compose up -d"

[commands.disk]
command = "df -h"
```

`fleetsh run hk1 bt-update` や `fleetsh run '@bt' bt-update` を想定しています。
v0.2 の予定です。

## 15. コマンドプロファイル（計画中）

検出した OS に応じてコマンドを選ぶ予定です。

```toml
[commands.patch]
debian = "apt-get update && apt-get upgrade -y"
ubuntu = "apt-get update && apt-get upgrade -y"
rocky = "dnf upgrade -y"
```

想定する操作は `fleetsh run '@all' patch` です。
OS ごとのプロファイルは v0.3 の計画です。

## 16. ホスト鍵の安全性

既定で厳密に検証します。初回接続は SHA256 フィンガープリントを表示し、明示的な信頼を求めます。
変更されたフィンガープリントは接続を拒否します。

```sh
fleetsh hostkey show hk1
fleetsh hostkey reset hk1
```

既定でホスト鍵検証を無効にしません。

## 17. タイムアウトと再試行

```toml
[defaults]
connect_timeout = "10s"
command_timeout = "30m"
parallel = 10
```

`--timeout 60s` でコマンドの期限を上書きできます。
当初の `retries = 1` は今後の予定で、現在は未対応フィールドとして拒否します。
コマンドを自動で再試行しません。

## 18. 出力形式

既定では人が読むための出力を表示し、スクリプトには `--json` を提供します。

```sh
fleetsh exec '@all' "uptime" --json
```

当初の JSON は設計例でした。現在は `results` と `summary` を使い、
ホストごとの stdout、stderr、終了状態、時間、エラー、切り詰め、スキップを含みます。
JSON Lines と静かな出力モードは今後の選択肢です。

## 19. ドライラン

`fleetsh update '@all' --dry-run` は予定する更新コマンドを表示します。
OS の確認だけのために接続し、パッケージは更新しません。
再起動のドライランは接続しません。
主に組み込み操作に使う機能で、任意のシェルコマンドを意味のある形で模擬するものではありません。

## 20. 承認確認

更新と再起動は対象の別名を表示し、確認を求めます。
自動実行を明示的に承認するには `--yes` を使います。

## 21. 設定の保存先

| OS | ディレクトリ |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\` |
| Linux | `~/.config/fleetsh/`、または XDG 設定ディレクトリ |
| macOS | `~/Library/Application Support/fleetsh/` |

現在のファイルは `config.toml` と同じディレクトリの `known_hosts` です。
当初は別ファイルの `hosts.toml` も検討していました。秘密値はこれらのファイルに保存しません。

## 22. OpenSSH 設定のインポート（計画中）

`fleetsh import ~/.ssh/config` で Host、HostName、User、Port、IdentityFile、
ProxyJump の設定を変換する予定です。
当初は `use_openssh_config = true` も検討していました。
どちらも未実装です。インポートは移行の手間を減らすための機能です。

## 23. エクスポート（計画中）

`fleetsh export` と `fleetsh export ssh-config` により、
TOML、JSON、OpenSSH 設定形式の出力を提供する予定です。
ホスト情報を持ち運びやすくし、特定ツールへの依存を減らします。

## 24. 秘密値のマスキング

既知のパスワード、プロキシパスワード、sudo パスワードを診断から隠します。
秘密鍵やトークンの保護も設計の対象です。
現在はバッファに保持した出力から取得済み秘密値をマスキングしますが、
任意のアプリケーションの秘密値は識別できません。対話端末の出力はそのまま渡します。

## 25. 初版の対象外

複雑な YAML プレイブック、望ましい状態の構成管理、テンプレート言語や Jinja、
ロール、コレクション、パッケージのエコシステム、リモート Agent、中央デーモン、
Web UI、スケジュール実行、Terraform の置換、Kubernetes 管理、サービス検出、CMDB。

主な操作をホストへの接続とコマンドの実行に集中させます。

## 26. 推奨 MVP

ホスト一覧の add/edit/rm/ls/show、SSH、リモート実行、グループとタグ、
パスワード・鍵・Agent 認証、OS 認証情報ストア、ProxyJump と SOCKS5、
並列実行、期限、厳密なホスト鍵、JSON、更新、再起動。

当初の MVP は `run` も含んでいましたが、ロードマップでは v0.2 とします。

## 27. v0.2

reboot-required、詳細な状態表示、独自の別名、OpenSSH インポート、
設定のエクスポート、接続の再試行、段階的な実行。
sudo の認証情報参照と直列実行も当初はここに含まれていましたが、基本の実行処理で実装済みです。

## 28. v0.3

SFTP のアップロードとダウンロード、OS ごとのコマンドプロファイル、
任意の履歴とログ、PowerShell/Bash/Zsh 補完、任意の TUI。

## 29. 推奨する技術

Go は I/O、SSH、CLI、設定、認証情報ストア、プロキシ、並列処理の要件に適しています。
Cobra または urfave/cli、x/crypto/ssh、x/net/proxy、TOML、
OS 認証情報ストアの抽象化、goroutine を利用します。

```text
CLI
├── ホスト一覧
├── 認証情報ストア
├── SSH 接続の構築
│   ├── 直接接続
│   ├── ProxyJump
│   ├── SOCKS5
│   └── HTTP/HTTPS CONNECT
├── 実行: 単一 / 並列
├── 操作: exec / update / reboot / status（計画中）
└── 出力: 端末 / JSON
```

## 30. 想定する利用体験

```sh
# ホスト一覧を表示
fleetsh ls
# ログイン
fleetsh ssh hk1
# 1 台で実行
fleetsh exec hk1 "uptime"
# すべてのホストで実行
fleetsh exec '@all' "df -h"
# ホスティング管理パネルを更新
fleetsh exec '@bt' "bt update"
# OS を更新
fleetsh update '@all'
# 再起動の必要性を確認（計画中）
fleetsh reboot-required '@all'
# 同時再起動数を制限
fleetsh reboot '@all' --parallel 2
# 設定した SOCKS プロキシを使用
fleetsh ssh special1
# 状態を構造化して出力（計画中）
fleetsh status '@all' --json
```

ユーザーが理解する必要があるのは、自分のホストと実行したいコマンドです。
ホスト一覧用 DSL、プレイブック、ロール、コレクション、モジュール、
facts、テンプレート、Python 環境を新たに学ぶ必要をなくします。
