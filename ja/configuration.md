---
title: 設定
permalink: /ja/configuration/
lang: ja
locale: ja
page_key: 'configuration/'
---

# 設定

<a id="storage"></a>

## VPS の設定を保存する場所

ホスト一覧は、fleetsh を実行するコンピューターの、現在の OS ユーザーの領域に保存します。
一つの `config.toml` にホストのアドレス、SSH ユーザーとポート、グループ、タグ、
プロキシ、既定値、認証情報の参照名を保存します。各 VPS は `[hosts.ALIAS]` の項目です。
実行ファイルを PATH に追加しても設定の保存先は変わりません。
作業ディレクトリを変えても同じ既定のホスト一覧を使用します。

| OS | 既定のホスト一覧 |
| --- | --- |
| Windows | `%APPDATA%\fleetsh\config.toml` |
| macOS | `~/Library/Application Support/fleetsh/config.toml` |
| Linux | `XDG_CONFIG_HOME` が設定されていれば `$XDG_CONFIG_HOME/fleetsh/config.toml`、それ以外は `~/.config/fleetsh/config.toml` |

Windows では通常 `C:\Users\<user>\AppData\Roaming\fleetsh\config.toml` になります。
`fleetsh init` はファイルを作成してパスを表示します。既存ファイルは保持します。
初期化後、次のコマンドで既定のホスト一覧を確認できます。

Windows PowerShell：

```powershell
$configFile = Join-Path $env:APPDATA "fleetsh\config.toml"
Write-Output $configFile
Get-Item -LiteralPath $configFile
notepad $configFile
```

macOS：

```sh
config_file="$HOME/Library/Application Support/fleetsh/config.toml"
printf '%s\n' "$config_file"
ls -l "$config_file"
```

Linux：

```sh
config_file="${XDG_CONFIG_HOME:-$HOME/.config}/fleetsh/config.toml"
printf '%s\n' "$config_file"
ls -l "$config_file"
```

これらは既定のパスを確認するコマンドです。`--config` を指定した場合は、
その指定先のファイルを確認してください。

### 別のホスト一覧を使用する

独自のホスト一覧を使うすべてのコマンドに `--config PATH` を指定します。

```sh
fleetsh --config "./inventories/production/config.toml" init
fleetsh --config "./inventories/production/config.toml" add web1 --host web1.example.com --user ubuntu --auth agent
fleetsh --config "./inventories/production/config.toml" ls
fleetsh --config "./inventories/production/config.toml" ssh web1
```

相対パスは現在の作業ディレクトリを基準に解決します。`~/` は現在のユーザーのホームに展開します。
空白を含むパスは引用符で囲んでください。この指定はその実行だけに適用し、
既定の保存先の変更や、ほかのホスト一覧との統合は行いません。
異なるディレクトリから実行する場合は絶対パスを使ってください。
`inventories/production/` と `inventories/staging/` のようにディレクトリを分けると、
信頼したホスト鍵のファイルも分離できます。

### 関連ファイルと認証情報

| データ | 保存先 |
| --- | --- |
| VPS のホスト一覧と認証情報の名前 | 既定の `config.toml`、または `--config` で選択したファイル |
| 信頼済み SSH ホスト公開鍵 | ホスト一覧と同じディレクトリの `known_hosts`。ホスト鍵を信頼した際に作成 |
| SSH 秘密鍵 | 各ホストの `key` に指定したパスから読み込み |
| 保存済みのパスワード、秘密鍵のパスフレーズ、プロキシ認証情報 | Windows Credential Manager、macOS Keychain、Linux Secret Service |
| ファイルロック | 各ファイルに隣接する `<inventory-path>.lock` と `known_hosts.lock` |

fleetsh は、ホスト一覧に隣接した `known_hosts` を使用します。
同じディレクトリ内のホスト一覧はこのファイルを共有します。`~/.ssh/known_hosts` は使用しません。
秘密値は現在のユーザーの OS 認証情報ストアで、サービス名 `fleetsh` の下に保存します。
異なるディレクトリのホスト一覧でも、同じ参照名なら同じ秘密値を使用します。
分離する場合は `production-web1-login` と `staging-web1-login` のように名前を分けてください。

### バックアップと別のコンピューターへの移行

1. fleetsh の実行が終了してから、ホスト一覧と隣接した `known_hosts` を移行先の設定ディレクトリにコピーします。ロックファイルは書き込みの調整用で、バックアップは不要です。
2. SSH 秘密鍵は別途移行し、アクセス権限を制限したまま、必要に応じて `key` のパスを変更します。
3. 移行先で `fleetsh credential add REFERENCE` を実行し、秘密値を再登録します。独自のホスト一覧には `--config PATH` も指定します。TOML をコピーしても OS ストアの秘密値は移行されません。
4. 接続前に、選択したホスト一覧で `fleetsh ls` を実行します。`known_hosts` を移行しなかった場合は、ホストのフィンガープリントを再確認して信頼してください。

ホスト一覧にはアドレスやユーザー名などのインフラ情報が含まれます。
バックアップは非公開で保管してください。ファイルの権限は本ページ末尾に記載しています。

## ホスト一覧の形式

`fleetsh init` はコメント付きの例を作成します。CLI でホストを追加するか、TOML を編集してください。
厳密な解析で、未知のフィールドと平文の秘密値を拒否します。
CLI の変更にはファイルロックとアトミックな置換を使い、コメントを書き直します。

```toml
[defaults]
connect_timeout = "10s"
command_timeout = "30m"
parallel = 10

[hosts.hk1]
host = "hk1.example.com"
port = 22
user = "ubuntu"
auth = "key"
key = "~/.ssh/id_ed25519"
groups = ["asia", "web"]
tags = ["production"]

[hosts.sg1]
host = "sg1.example.com"
user = "ubuntu"
auth = "password"
credential = "sg1-login"
sudo_credential = "sg1-sudo"
groups = ["asia"]

[hosts.jump-hk]
host = "jump.example.com"
user = "ubuntu"
auth = "agent"

[hosts.private1]
host = "10.0.0.10"
user = "ubuntu"
auth = "agent"
proxy_jump = "jump-hk"

[hosts.proxy1]
host = "target.example.com"
user = "ubuntu"
auth = "agent"
proxy = "socks5://127.0.0.1:1080"
proxy_credential = "local-proxy"

[hosts.console1]
host = "console.example.com"
connection = "console-only"
```

ホストの既定値は、ポート 22、接続方式 `ssh`、認証方式 `agent` です。
SSH ホストにはユーザー名、鍵認証には鍵のパスが必要です。
鍵の相対パスは現在の作業ディレクトリから解決し、`~/` も展開します。
参照を指定しないパスワードと秘密鍵のパスフレーズは対話的に入力できます。

別名、グループ、タグ、参照名は英数字、ピリオド、アンダースコア、ハイフンを使用し、
英数字で始まる 128 文字以内の名前にします。別名 `all` は予約済みです。
並列数は 1〜256、タイムアウトは正の時間を指定します。

## 対象の選択

| 指定 | 意味 |
| --- | --- |
| `hk1` | 1 台のホスト |
| `hk1,sg1` | 明示した複数ホスト。重複を除去し並べ替え |
| `all` / `@all` | ホスト一覧のすべてのホスト |
| `@asia` | asia グループのホスト |
| `--tag production` | 選択済みのホストをタグで絞り込み |

未知のホストやグループ、空のリモート対象はローカルエラーになります。
コンソール専用のホストは一覧に表示し、リモート操作ではスキップします。

## 認証情報ストア

`credential`、`proxy_credential`、`sudo_credential` は名前だけを保存します。
`fleetsh credential add REFERENCE`、`ls`、`rm REFERENCE` で管理します。
既存の秘密値を置き換える場合は `add --replace` が必要です。

秘密値には Windows Credential Manager、macOS Keychain、Linux Secret Service を使用します。
Linux には、動作中のユーザー D-Bus セッションと、GNOME Keyring などの解錠済み Secret Service が必要です。
ストアが利用できない場合はエラーとなり、平文保存には切り替えません。

`credential ls` は、このホスト一覧で登録または参照した認証情報を表示します。
無関係な OS ストアの項目は列挙しません。トップレベルの `credentials` 配列は、
このホスト一覧から追加した参照名を記録します。
ストアから削除してもホスト側の参照は残るため、必要に応じてホスト設定も変更してください。

## プロキシと SSH Agent

踏み台には `proxy_jump`、SOCKS5 には `proxy` を使います。両方の同時指定はできません。
循環する踏み台や、存在しない踏み台は拒否します。接続先と踏み台の鍵を検証します。
SOCKS5 ではプロキシ側が接続先のホスト名を解決します。

認証付き SOCKS5 は、`credential add` で `username:password` を保存し、
`proxy_credential` を設定します。URL にユーザー名やパスワードを含めることは禁止します。

Unix の Agent は `SSH_AUTH_SOCK` を使用します。
Windows は既定で `\\.\pipe\openssh-ssh-agent` を使い、
`SSH_AUTH_SOCK` で別のソケットやパイプを指定できます。Agent には鍵を 1 本以上読み込んでください。

HTTP CONNECT、独自のコマンド別名、OpenSSH のインポートとエクスポート、
ポート転送は[ロードマップ](../roadmap/)を参照してください。

## ファイルと権限

ホスト一覧とホスト鍵の保存先は[上記](#storage)を参照してください。
Unix ではファイルを 0600、新規ディレクトリを 0700 で作成します。
Windows ではユーザーディレクトリの ACL を使います。
ホスト一覧と `known_hosts` は通常のファイルに限り、シンボリックリンクは拒否します。
秘密値は OS ストアに保持します。
