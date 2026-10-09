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

## npx または npm で実行する

Node.js 22 以降を使用します。

```sh
npx fleetsh init
npx fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519
npx fleetsh ssh hk1
npx fleetsh alive
npx fleetsh stats
```

または `npm install -g fleetsh` でグローバルにインストールし、
すべての例の `npx fleetsh` を `fleetsh` に置き換えてください。
直接バイナリーと同じホスト一覧、OS の資格情報ストアを使用します。

ランチャーは GitHub Release を自動で追従し、正式版を優先します。
正式版がない場合のみプレビューを使用します。
子コマンドの前に `npx fleetsh --release bundled version` を指定すると元の版を、
`--release v0.1.0-alpha.1` を指定すると特定の Go 版を選択します。
`npx fleetsh --launcher-help` または
[バージョン選択、キャッシュと PATH](../installation/)を参照してください。

## ヘルプとオフラインガイド

実行ファイルには、例付きの英語コマンドヘルプと 11 のオフラインガイドが含まれます。
`init` の実行前でも、設定ファイル、ネットワーク接続、資格情報ストアなしで参照できます。
オンライン文書は言語ごとに分かれており、既定は英語です。

位置引数の不足・過剰や不正なフラグがある場合、エラーと構文・使用例・
フラグのヘルプを標準エラー出力に表示し、終了コード 2 を返します。
`npx fleetsh exec` と `npx fleetsh exec whoami` はヘルプを表示します。
`exec` にはホスト・グループの指定と、一つに引用したコマンドが必要です。
例: `npx fleetsh exec '@all' "whoami"`。
ヘルプのリンクは HTTPS の配信先を直接使用し、GitHub Pages の既定 URL
から継承される HTTP リダイレクトを避けます。

```sh
fleetsh --help
fleetsh help exec
fleetsh add --help
fleetsh docs
fleetsh docs passwords
fleetsh docs groups
fleetsh docs patching
```

| トピック | 内容 |
| --- | --- |
| `quickstart` | 最初のホスト、SSH の信頼設定、基本操作 |
| `config` | OS ごとの保存先、既定値、バックアップ |
| `passwords` | 自動保存、更新、共有する参照、sudo |
| `groups` | 複数グループ、所属の追加と削除、タグ |
| `selectors` | ホストとグループの和集合、`@all`、タグ絞り込み |
| `exec` | 並列数、引用符、タイムアウト、sudo、JSON |
| `proxies` | 全体とホストごとの SOCKS5、HTTP/HTTPS CONNECT、ProxyJump |
| `patching` | 更新の事前確認、apt などの更新、再起動 |
| `monitoring` | SSH 接続確認、CPU・メモリー・swap・ユーザー・ディスク |
| `forwarding` | ローカル・リモート・動的 TCP 転送 |
| `troubleshooting` | 終了コード、信頼、認証、資格情報ストア |

`fleetsh docs TOPIC` でガイドを表示します。オフラインの本文は英語のプレーンテキストです。
`--json` は対応する一覧や一括操作の結果に適用され、文書表示には適用されません。
ヘルプの表示は、例の実行や掲載リンクのブラウザー起動を行いません。

## ホスト一覧を初期化する

ホスト一覧は現在の OS ユーザーのローカル設定ディレクトリに保存されます。
Windows は `%APPDATA%\fleetsh\config.toml`、
macOS は `~/Library/Application Support/fleetsh/config.toml`、
Linux は `$XDG_CONFIG_HOME/fleetsh/config.toml` または `~/.config/fleetsh/config.toml` です。
[保存先、独自のホスト一覧、バックアップ](../configuration/#storage)も参照してください。

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups asia,web
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth password --groups asia
fleetsh ls
fleetsh ls '@asia' --json
fleetsh show hk1
fleetsh edit hk1 --port 2222
```

<a id="required-host-options"></a>

### ホストの追加に必須の引数とオプション

`fleetsh add HOST` の `HOST` はローカルの別名で、接続先アドレスではありません。

| 引数・オプション | 必須になる条件 |
| --- | --- |
| `HOST` | 常に必要: 新しいローカル別名 |
| `--host ADDRESS` | 常に必要: console-only ホストも含む |
| `--user USER` | SSH ホスト。既定は `--connection ssh` |
| `--key PATH` | `--auth key` を選択した場合 |

新規ホストでは `--auth` は省略可能で既定は `password`、`--port` の既定は 22 です。
`--auth` を省略するとパスワードを 2 回確認し、OS の認証情報ストアに保存します。
`--auth agent` は鍵を読み込んだ SSH Agent が動作している場合に使います。
秘密鍵ファイルには `--auth key --key PATH` を指定してください。
既存ホストの認証方式は維持します。`SSH agent unavailable` が表示され、パスワードでログインする場合は、
`fleetsh edit HOST --auth password` の後に `fleetsh ssh HOST` を実行してください。
別のホスト一覧を使う場合は、両方のコマンドに同じ `--config PATH` を指定します。
パスワードは非表示の入力で確認するため、パスワード用の必須オプションはありません。
追加時に必須値が不足または空の場合、ホスト一覧の読み込みや秘密値の入力より前に、
不足しているオプションとヘルプを表示します。不正なホスト設定でもヘルプを表示します。
`edit HOST` は省略した値を保持し、変更後の値を検証します。
`--connection ssh` や `--auth key` に変更すると、`--user` や `--key` が必要になる場合があります。
console-only ホストは `--user` を省略できます。
詳細は `fleetsh add --help` と `fleetsh edit --help` を参照してください。

パスワード入力は表示されません。秘密値は OS の認証情報ストアに保存し、
ホスト一覧には参照名だけを保存します。秘密鍵のパスフレーズも同じ `--credential` で参照できます。
参照がなければ、非表示の入力で確認します。

パスワード認証のホスト追加では、非表示の入力で 2 回確認し、自動で保存します。
生成した参照は `fleetsh-ssh-` で始まり、ホスト一覧には名前だけを保持します。
`fleetsh edit sg1 --save-password` で、参照を手動管理せずに更新できます。
[パスワード保存](../configuration/#passwords)と[グループ変更](../configuration/#groups)も参照してください。

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
ポート転送は名前付き設定と `fleetsh forward` を使います。OpenSSH 引数の引き渡しは今後の予定です。

## コマンドひとつでリモート監視

```sh
fleetsh alive
fleetsh stats
fleetsh alive '@web' --timeout 10s
fleetsh stats '@web' --parallel 5 --timeout 15s
fleetsh monitor hk1
fleetsh stats --json
```

`alive` は SSH 認証とコマンド実行の可否を確認します。ICMP の確認ではありません。
`stats`（別名 `monitor`）は Linux のスナップショットを 1 回表示します。
内容は `whoami`、1 秒間の CPU 使用率、メモリーと swap の使用量（MiB と割合）、
空き容量を含む `df -h -P` です。両コマンドの既定対象は `@all` で、
セレクター、タグ、並列数、タイムアウトを指定できます。コンソール専用ホストはスキップします。
JSON は通常の一括実行レポートで、スナップショットは各ホストの `stdout` に入ります。

Agent のインストールや sudo は不要です。Linux の `/proc`、`awk`、`sleep`、
`whoami`、`df` が必要です。メモリー使用量は総量から利用可能量を引き、swap 無効時は 0 と表示します。
CPU はホストの値で、コンテナーの割り当て上限ではありません。`--sudo` は表示されるユーザーを変更します。
オフラインの説明は `fleetsh docs monitoring` にあります。

ポート転送は `fleetsh forward HOST [NAME]` で起動します。
[ローカル・リモート・動的転送の設定](../configuration/#forwarding)を参照してください。

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
