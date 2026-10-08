---
title: 使い始める
permalink: /ja/getting-started/
lang: ja
locale: ja
page_key: 'getting-started/'
---

# 使い始める

fleetsh は v0.1 の基本的な操作を実装しています。現在は開発版です。
[インストールと PATH の設定](../installation/)を行うと、どのディレクトリからでも実行できます。

## ホスト一覧を初期化する

ホスト一覧は現在の OS ユーザーのローカル設定ディレクトリに保存されます。
Windows は `%APPDATA%\fleetsh\config.toml`、
macOS は `~/Library/Application Support/fleetsh/config.toml`、
Linux は `$XDG_CONFIG_HOME/fleetsh/config.toml` または `~/.config/fleetsh/config.toml` です。
[保存先、独自のホスト一覧、バックアップ](../configuration/#storage)も参照してください。

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups asia,web
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth password --credential sg1-login --groups asia
fleetsh credential add sg1-login
fleetsh ls
fleetsh ls '@asia' --json
fleetsh show hk1
fleetsh edit hk1 --port 2222
```

パスワード入力は表示されません。秘密値は OS の認証情報ストアに保存し、
ホスト一覧には参照名だけを保存します。秘密鍵のパスフレーズも同じ `--credential` で参照できます。
参照がなければ、非表示の入力で確認します。

すべてのコマンドで `--config PATH` を指定できます。`~/` で始まるパスは展開されます。
ホスト一覧は厳密に検証し、未知のフィールドや平文の `password` フィールドを拒否します。
`edit` は TOML を書き直すため、コメントを保持しません。

## ホストの身元を確認する

```sh
fleetsh ssh hk1 --connect-timeout 60s
fleetsh hostkey show hk1
```

初回接続では SHA256 フィンガープリントを表示し、対話的な `y` の入力を求めます。
信頼する前に、独立した情報源で確認してください。この確認も接続タイムアウトに含まれるため、
必要に応じて `--connect-timeout` を調整します。
変更されたホスト鍵は常に拒否し、無人実行では未知の鍵も拒否します。

正当な鍵の交換を独立した情報源で確認した後は、次のように実行します。

```sh
fleetsh hostkey reset hk1
fleetsh ssh hk1 --connect-timeout 60s
```

専用の `known_hosts` は、選択したホスト一覧のファイルと同じディレクトリにあります。
SSH の標準入力が端末の場合は PTY と raw モードを使い、端末サイズの変更も転送します。
OpenSSH 引数の引き渡しとポート転送は今後の実装予定です。

## コマンドを実行する

PowerShell では `fleetsh ls '@asia'` のようにグループ指定を引用符で囲みます。
これにより `@` を変数展開として扱うことを防ぎます。同じ書式を Bash と Zsh でも使えます。

```sh
fleetsh exec hk1 "uptime"
fleetsh exec '@asia' "df -h" --parallel 10
fleetsh exec hk1,sg1 "uptime" --serial --json
fleetsh exec all "docker ps" --tag web --timeout 60s
fleetsh exec hk1 "apt-get update" --sudo
```

ホストに `sudo_credential` がない場合、`--sudo` はパスワード不要の `sudo -n` を使います。
参照した sudo パスワードは SSH チャネルの標準入力で送信し、コマンドには埋め込みません。
コマンドには対話的な標準入力を渡さないため、対話操作には `ssh` を使ってください。

出力はホストごとにまとめ、stdout と stderr をそれぞれ 4 MiB に制限します。
JSON は `results` と `summary` を持ち、ホスト、終了状態、時間、エラー分類、
出力の切り詰め、スキップの情報を含みます。コンソール専用のホストは明示的にスキップします。

終了コードは、0 が全実行対象の成功、1 がコマンド失敗・キャンセル・実行タイムアウト、
2 がローカルの引数・設定・認証情報の準備エラー、
3 が接続・認証・プロキシ・ホスト鍵の失敗（接続タイムアウトを含む）です。
複数種類のリモート失敗がある場合は、コマンド失敗を優先します。
対話シェルのリモート終了状態が非ゼロの場合は 1、通信エラーの場合は 3 を返します。

## Linux VPS を更新、再起動する

```sh
fleetsh update '@asia' --dry-run --sudo
fleetsh update '@asia' --sudo
fleetsh update hk1 --dist --sudo --yes
fleetsh reboot '@asia' --dry-run --sudo
fleetsh reboot '@asia' --parallel 2 --sudo --wait-timeout 5m
```

更新のドライランは OS を調べるために接続しますが、パッケージ更新は実行しません。
対応 OS ID は Debian、Ubuntu、RHEL、Rocky、AlmaLinux、Fedora、CentOS、Arch、Manjaro です。
未対応 OS や事前確認の失敗がある場合、すべての対象への更新を中止します。
再起動のドライランでは接続しません。

実際の更新と再起動は対象の別名を表示し、確認を求めます。
自動実行を明示的に承認する場合は `--yes` を指定してください。
再起動の標準並列数は 2 です。Linux の起動 ID が変わり、uptime の確認に成功した場合のみ成功とします。
コマンドを自動で再試行することはありません。

一連の保守手順は[すべての VPS を更新する](../cases/patch-all-vps/)を参照してください。
