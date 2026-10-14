---
title: 日常の操作
permalink: /ja/operations/
lang: ja
locale: ja
page_key: 'operations/'
---

# 日常の操作

ホストの編集、接続診断、日常の操作を説明します。CLI ヘルプは英語です。
独自のホスト一覧を使う場合は、すべてのコマンドに同じ `--config PATH` を指定します。

<a id="interactive"></a>

## 対話的な追加と編集

ターミナルで `npx fleetsh add hk1` を実行すると、不足するアドレス、ユーザー名、
鍵のパスを確認し、その後に非表示でパスワードを入力します。
新しいホストは標準で SSH パスワード認証を使います。引数をすべて指定することもできます。

`npx fleetsh edit hk1` はメニューを開き、アドレス、ユーザー、保存したパスワード、
グループ、タグ、説明、認証方法を変更できます。キャンセルでは保存しません。
グループとタグは一覧を置き換え、空の入力で消去します。
スクリプトと `--json` では変更オプションが必要で、メニューを開きません。

```sh
npx fleetsh edit hk1
npx fleetsh edit hk1 --save-password
npx fleetsh edit hk1 --add-groups production
npx fleetsh show hk1
```

変更するのはローカルの保存値だけで、サーバーのパスワードは変更しません。
保存が成功すると、更新したホスト名を表示します。

<a id="diagnostics"></a>

## 接続を診断する

```sh
npx fleetsh doctor hk1
npx fleetsh doctor hk1 --connect-timeout 30s --json
```

設定のパス、実際のプロキシ／踏み台経路、ローカル認証準備、ネットワーク、
ホスト鍵、SSH 認証、簡単なリモート確認を表示します。
診断は設定を変更せず、新しい鍵を自動で信頼しません。
未知の鍵は指紋を別の方法で確認してから `fleetsh ssh hk1` で明示的に受け入れます。

エラーは原因と次のコマンドを説明します。JSON は構造化した確認項目と既存の終了コードを保持します。
保存したパスワードが間違っている場合は `fleetsh edit hk1 --save-password` を使います。

<a id="monitoring"></a>

## リソース使用量を継続して見る

```sh
npx fleetsh stats '@all' --table
npx fleetsh stats '@all' --watch 5s
npx fleetsh stats '@web' --watch 5s --json
```

表はホストの状態、CPU、メモリ、swap、ユーザー、ルートファイルシステムの空きを表示します。
各回に時刻を付けます。ほかのディスクは通常の `stats` で確認できます。
1 台が失敗してもほかのホストを表示します。

間隔は 1 秒以上で、前の確認が終了してから数えます。確認は重複せず、Ctrl+C で停止します。
JSON は 1 行につき 1 レポートを出力し、`timestamp`、`results`、`summary` を含みます。
設定と認証情報は開始時に読み込みます。変更後は監視コマンドを再実行してください。
リモート Agent は不要です。

<a id="saved-commands"></a>

## よく使うコマンドを保存する

ホスト一覧に追加します。

```toml
[commands.disk]
command = "df -h"
timeout = "30s"

[commands.docker-update]
command = "docker compose pull && docker compose up -d"
sudo = true
timeout = "10m"
```

```sh
fleetsh run '@web' docker-update --dry-run
fleetsh run '@web' docker-update --parallel 2
fleetsh run hk1 disk --json
```

セレクターとコマンド名は必須です。`--sudo` と `--timeout` は保存した既定値を上書きします。
プレビューは接続せず、コマンドを自動で再試行しません。
定義にパスワードやトークンを入れないでください。

<a id="completion"></a>

## シェル補完を有効にする

バイナリをインストールするか `npm install -g fleetsh` を実行し、`fleetsh` を PATH に置きます。
`bash`、`zsh`、`fish`、`powershell` に対応します。

```powershell
fleetsh completion powershell > fleetsh-completion.ps1
. ./fleetsh-completion.ps1
```

毎回有効にする場合は、絶対パスを使う読み込み行を PowerShell プロファイルに追加します。
Bash は生成したスクリプトを読み込みます。Zsh は `_fleetsh` を `fpath` に置き
`compinit` を初期化します。Fish は
`~/.config/fish/completions/fleetsh.fish` に置きます。

補完はローカルのホスト、グループ、保存したコマンド名だけを読みます。
秘密値やリモート接続は使いません。対応する入口は `fleetsh` コマンドです。
`npx fleetsh` の補完はシェルに依存します。

<a id="maintenance"></a>

## 再起動の必要性と段階的な再起動

```sh
fleetsh update '@all' --sudo --check-reboot
fleetsh reboot-required '@all'
fleetsh reboot '@web' --sudo --batch-size 2 --wait-timeout 5m
```

Debian／Ubuntu は reboot-required マーカーを使います。
ほかの OS では、インストール済みの `needs-restarting -r` を使えます。
未対応の場合は「不明」と表示します。更新だけでは再起動しません。

各バッチの新しい起動 ID と正常性の確認後に次のバッチを始めます。
失敗すると後続を停止し、未実行ホストはスキップと表示します。
引き続き確認が必要です。自動化で承認する場合は `--yes` を指定します。
バッチの大きさと同時実行数は別々に指定できます。

<a id="import-export"></a>

## OpenSSH のインポートとホスト設定のエクスポート

```sh
fleetsh import ~/.ssh/config --dry-run
fleetsh import ~/.ssh/config --user ubuntu
fleetsh export --format toml --output inventory.toml
fleetsh export --format json
fleetsh export --format ssh-config --output ssh-config
```

明示的な別名、ワイルドカードの既定値、`HostName`、`User`、`Port`、
1 つの `IdentityFile`、1 つの踏み台別名 `ProxyJump` に対応し、最初の一致を優先します。
ユーザー名がない場合は `--user USER` が必要です。
同じ別名があれば停止し、既存ホストを上書きしません。

`Include` と `Match` はインポートを停止します。`ProxyCommand`、
複数／アドレス形式の ProxyJump、未対応の展開変数が必要なホストは説明付きでスキップします。
その他の未対応設定も一覧にし、ローカルコマンドを実行しません。
インポートしたホストは鍵または agent 認証を使います。パスワード認証は `edit HOST` で選択します。

OS の秘密値、認証情報の参照、保存したコマンド定義はエクスポートしません。
説明と接続設定は入力した内容を保持します。OpenSSH では表せないプロキシ URL と転送設定を通知します。
ファイルは UTF-8 で保存し、既存ファイルには `--overwrite` が必要です。

<a id="transfers"></a>

## SFTP でアップロード／ダウンロードする

```sh
fleetsh upload '@web' ./script.sh /tmp/script.sh --dry-run
fleetsh upload '@web' ./script.sh /tmp/script.sh
fleetsh download hk1 /var/log/syslog ./syslog.txt
fleetsh download '@web' /var/log/syslog ./logs
```

既存の SSH 認証、プロキシ、ホスト鍵を使います。
通常ファイル 1 つが対象で、再帰ディレクトリと `--sudo` は未対応です。
アップロード先の親ディレクトリは存在し、SSH ユーザーに書き込み権限が必要です。

単一ホストではローカルのファイル名または既存ディレクトリを指定できます。
複数ホストでは別名と短いハッシュを含む個別ディレクトリを作り、名前の衝突を避けます。
プレビューは正確なパスを表示し、ファイルを作成しません。
既存ファイルには `--overwrite` が必要です。中断では途中のファイルが残ることがあります。
再試行の前に確認してください。
