---
title: fleetsh について
permalink: /ja/product/
lang: ja
locale: ja
page_key: 'product/'
---

# fleetsh について

fleetsh は VPS の接続設定をローカルに保存するツールです。
SSH ログイン、複数ホストでのコマンド実行、リソース確認、パッケージ更新、
ファイル転送に使えます。

Go の実行ファイルとして Windows、macOS、Linux で動作します。
接続先には SSH が必要ですが、fleetsh のインストールは不要です。npm からも起動できます。

## よく使う操作

| 操作 | コマンド |
| --- | --- |
| ホストを追加 | `fleetsh add hk1` |
| 設定や保存したパスワードを変更 | `fleetsh edit hk1` |
| ログイン失敗を診断 | `fleetsh doctor hk1` |
| グループでコマンドを実行 | `fleetsh exec '@web' "uptime"` |
| リソースを継続確認 | `fleetsh stats '@all' --watch 5s` |
| 保存したコマンドを実行 | `fleetsh run '@web' docker-update` |
| 再起動の必要性を確認 | `fleetsh reboot-required '@all'` |
| ファイルをアップロード | `fleetsh upload hk1 ./script.sh /tmp/script.sh` |

ターミナルの `add` と `edit` は入力を案内します。スクリプトでは引数を指定してください。
ホストは複数のグループに所属できます。`@` で始まるセレクターは PowerShell で引用符を付けます。

最初のホストは[使い始める](../getting-started/)を、
その他の操作は[日常の操作](../operations/)を参照してください。

## 設定と認証情報

接続設定はローカルの TOML ファイルに保存します。
パスワードや鍵のパスフレーズは OS の認証情報ストアに保存し、設定には参照だけを記録します。

SSH は鍵、パスワード、agent、プロキシ、踏み台に対応します。
ローカル、リモート、動的ポート転送は名前付き設定で指定します。
例と保存先は[設定](../configuration/)、ホスト鍵と秘密値の扱いは[セキュリティ](../security/)を参照してください。

## 現在の制限

SFTP は通常ファイルを 1 つずつ転送します。OpenSSH インポートはヘルプに示す項目に対応し、
未対応の設定を通知します。リソース監視には Linux の情報を使います。
コマンドを自動で再試行しません。

今後の作業は[ロードマップ](../roadmap/)に記載しています。
[KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh) で開発し、GPL-3.0-only を採用しています。
