---
title: すべての VPS を更新する
description: 同時実行数を制限し、ホストごとの結果を確認しながら VPS 群で apt update と apt upgrade を実行します。
permalink: /ja/cases/patch-all-vps/
lang: ja
locale: ja
page_key: 'cases/patch-all-vps/'
---

# すべての VPS を更新する

複数の Debian または Ubuntu VPS に `apt update && apt upgrade` 相当の処理を実行する例です。
fleetsh で対象を選び、更新内容を確認し、同時実行数を制限してホストごとの結果を調べます。

## 1. ホストを準備する

fleetsh をインストールして [PATH を設定](../../installation/)し、ホスト一覧に追加します。
次は鍵認証を使う 2 台の例です。

```sh
fleetsh init
fleetsh add hk1 --host hk1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups apt
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519 --groups apt
fleetsh ls '@all'
```

各ホストのフィンガープリントを独立した情報源で確認し、対話的に信頼します。
ほかの別名でも繰り返してください。

```sh
fleetsh ssh hk1 --connect-timeout 60s
```

接続を確認したらリモートシェルを終了します。
無人実行は未知の鍵や変更された鍵を拒否します。
PowerShell では `@all` などのグループ指定を引用符で囲みます。Bash と Zsh でも同じ書式を使えます。

以降は SSH ユーザーが sudo を利用できることを前提とします。
`sudo_credential` がなければ、パスワード不要の sudo を使います。
sudo にパスワードが必要な場合は、非表示の入力で認証情報を保存し、ホストに関連付けます。

```sh
fleetsh credential add hk1-sudo
fleetsh edit hk1 --sudo-credential hk1-sudo
```

必要に応じて他のアカウントでも繰り返します。root で接続する場合は `--sudo` を省略します。

## 2. 更新内容を確認する

```sh
fleetsh update '@all' --dry-run --sudo --parallel 5
```

各 OS を調べるために接続し、予定するコマンドを表示します。パッケージは更新しません。
Debian/Ubuntu では次のコマンドを使用します。

```sh
DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
```

パッケージ一覧の取得とインストール済みパッケージの更新をスクリプト向けに実行します。
`&&` により、一覧の取得が成功した場合だけアップグレードします。
利用可能な更新には通常の更新とセキュリティ更新の両方が含まれ、
セキュリティパッチだけに限定するものではありません。

OS が混在する場合、`update` は対応するパッケージマネージャーを選びます。
事前確認が失敗した場合や未対応 OS が含まれる場合は、すべての対象への更新を中止します。
コンソール専用のホストはスキップします。

大きなホスト一覧から apt 用グループだけを選ぶ場合は、
以下の `'@all'` を `'@apt'` に置き換えてください。

## 3. 更新を適用する

```sh
fleetsh update '@all' --sudo --parallel 5 --timeout 30m
```

表示された別名を確認して承認します。同時に更新するホストは最大 5 台です。
1 台ずつ処理する場合は `--parallel 5` の代わりに `--serial` を使います。
タイムアウトは各ホストの更新コマンドに適用します。
失敗したコマンドの自動再試行や、一部の失敗後のパッケージのロールバックは行いません。
更新でサービスが再起動することがあるため、適切な保守時間帯に実行してください。

### apt コマンドを直接実行する

選択したすべてのホストが Debian/Ubuntu なら、apt を直接実行することもできます。

```sh
fleetsh exec '@all' "DEBIAN_FRONTEND=noninteractive apt update && DEBIAN_FRONTEND=noninteractive apt upgrade -y" --sudo --parallel 5 --timeout 30m
```

`--sudo` は `&&` の両側を含むシェルコマンド全体に適用します。
バッチ実行には対話的な標準入力がないため、`-y` でアップグレードを承認します。
`exec` は即座に実行し、組み込み更新の OS 事前確認、ドライラン、承認確認は行いません。
スクリプトには apt-get が適しており、組み込みの更新も apt-get を使います。
パッケージ固有の確認で無人更新が失敗することもあるため、再実行前に対象ホストの stderr を確認してください。

## 4. 結果を確認し、自動化する

対話モードの出力には、ホストごとの成功と失敗、stdout、stderr、集計を表示します。
終了コードは、0 が全実行対象の成功、1 がコマンド失敗・タイムアウト・キャンセル、
2 がローカルの設定や認証情報の準備エラー、
3 が接続・認証・プロキシ・ホスト鍵のエラーです。
パッケージ一覧の取得に失敗したホストでは、アップグレードを実行しません。

明示的に承認した自動実行には `--yes --json` を使います。

```sh
fleetsh update '@all' --sudo --parallel 5 --timeout 30m --yes --json > patch-results.json
```

JSON の `results` と `summary` には、ホストごとのエラーも含まれます。
レポートに加えて、プロセスの終了コードも確認してください。
PowerShell では fleetsh の実行直後に `$LASTEXITCODE`、
Bash/Zsh では `$?` を確認します。原因を解消した後、失敗した別名だけを再実行してください。

## 5. 再起動が必要か確認する

Debian/Ubuntu では再起動要求のファイルを確認します。

```sh
fleetsh exec '@all' "if test -f /var/run/reboot-required; then echo REBOOT_REQUIRED; else echo NO_REBOOT_MARKER; fi" --parallel 5
```

ファイルがない場合は、この仕組みで OS が再起動を要求していないことを示します。
すべてのサービスが更新後のライブラリを使用していることを保証するものではありません。
再起動が必要な別名を選び、ドライランの後に実行します。

```sh
fleetsh reboot hk1,sg1 --dry-run --sudo
fleetsh reboot hk1,sg1 --sudo --parallel 2 --wait-timeout 5m
```

対象ホストを確認して承認します。成功には、新しい Linux 起動 ID と uptime の確認が必要です。
パッケージ更新自体は自動で再起動しません。

コマンドの詳細は[使い始める](../../getting-started/)、
認証情報とホスト鍵の扱いは[セキュリティ設計](../../security/)を参照してください。
