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

<a id="groups"></a>

## グループへの所属を変更する

1 台のホストを複数のグループに所属させられます。グループはホストの所属情報から決まるため、
事前に別の登録は不要です。ホスト追加後も変更できます。

```sh
fleetsh edit hk1 --add-groups production,monitoring
fleetsh edit hk1 --remove-groups asia
fleetsh edit hk1 --groups web,production
fleetsh edit hk1 --groups ""
fleetsh ls '@production'
```

`--add-groups` は既存の所属を保持し、重複した追加を無視します。
`--remove-groups` は指定した所属だけを削除します。未所属の名前は無視します。
`--groups` は一覧全体を置き換え、空の値を指定するとすべて解除します。
異なる名前の追加と削除は同時に指定できますが、`--groups` と組み合わせたり、
同じ名前を追加と削除の両方に指定したりすることはできません。
グループの編集でホストのパスワードを入力・変更することはありません。

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

<a id="passwords"></a>

## ホストのパスワードを保存する

`--auth password` で SSH ホストを追加し、`--credential` を省略すると、非表示の入力で
パスワードを 2 回確認します。fleetsh が一意の `fleetsh-ssh-ALIAS-RANDOM` 参照を生成し、
OS ストアに秘密値を保存してホストに関連付けます。先に `credential add` を実行する必要はありません。

```sh
fleetsh add sg1 --host sg1.example.com --user ubuntu --auth password --groups asia,web
fleetsh edit sg1 --save-password
fleetsh show sg1
```

`edit --save-password` は新しい参照にパスワードを保存します。
以前の認証情報は残すため、共有するほかのホストには影響しません。
明示的な参照なしでパスワード認証に切り替える場合も、入力を確認して保存します。
`show` または `credential ls` で参照を確認し、不要な古い参照は
`credential rm REFERENCE` で削除できます。ホストの削除だけでは認証情報を削除しません。

接続のたびに入力する場合は `--no-save-password` を使います。
編集ではホストの参照を解除し、OS ストアの保存済み項目は保持します。
非対話で追加する場合は、既存の `--credential REFERENCE` を指定するか、
`--no-save-password` を明示してください。空でない参照とパスワード保存オプションは併用できません。

入力前にホスト設定を検証します。入力や保存が失敗してもホスト一覧は変更しません。
ホスト一覧の書き込みに失敗した場合、新しく保存した秘密値を削除します。

### fleetsh が作成した認証情報の識別

| プラットフォーム | 表示される識別情報 |
| --- | --- |
| Windows Credential Manager | 対象名 `fleetsh:REFERENCE`、コメント `Created by fleetsh (KanataLabs). Managed VPS credential.` |
| macOS Keychain | サービス `fleetsh`、ラベル `fleetsh:REFERENCE`、同じ作成元コメント |
| Linux Secret Service | サービス属性 `fleetsh`、`fleetsh:REFERENCE` で始まるラベルと作成元の説明 |

自動保存と手動の `credential add` は、新規または置き換えた項目にこの識別情報を付けます。
既存の参照と `fleetsh` サービスは引き続き使えます。古い項目は置換時に説明が追加されます。

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

<a id="proxies"></a>

## 全体とホストごとの SSH プロキシ

既存の `[defaults]` にプロキシを設定すると、SSH 接続の共通経路になります。

```toml
[defaults]
proxy = "socks5://127.0.0.1:1080"
# proxy_credential = "local-proxy"
```

`socks5://`、`http://`、`https://` に対応します。HTTP/HTTPS は CONNECT を使い、
HTTPS ではプロキシの証明書を検証します。既定ポートは順に 1080、80、443 です。
URL に認証情報、パス、クエリ、フラグメントを含めないでください。
認証が必要な場合は `fleetsh credential add local-proxy` の非表示入力で
`username:password` を保存し、`proxy_credential` で参照します。TOML に秘密値は保存しません。

ホスト側で URL を指定しない場合、全体設定を継承します。ホストの URL は全体設定を上書きし、
そのホストの `proxy_credential` だけを使います。別の接続先へ全体の認証情報を流用しません。
全体 URL の継承時は、ホストの認証情報参照だけを変更できます。
`proxy = "direct"` を指定すると全体プロキシを使わず直接接続します。

```sh
fleetsh edit hk1 --proxy http://127.0.0.1:7890
fleetsh edit hk1 --proxy socks5://127.0.0.1:1080 --proxy-credential local-proxy
fleetsh edit hk1 --proxy direct --proxy-credential ""
fleetsh edit hk1 --proxy "" --proxy-credential ""
```

最後の例は継承に戻します。プロキシは `ssh`、`exec`、`alive`、`stats`、`update`、
`reboot`、`forward` に適用されます。設定は接続時に解決され、編集しても継承と明示設定を区別します。

`proxy_jump` は接続先への経路を踏み台に切り替えます。踏み台自体は、上書きしない限り全体の
プロキシを継承します。同じホストで `proxy` と `proxy_jump` を併用できません。
循環や存在しない踏み台を拒否し、接続先と踏み台の鍵をそれぞれ検証します。

Unix の Agent は `SSH_AUTH_SOCK` を使います。Windows の既定は
`\\.\pipe\openssh-ssh-agent` で、`SSH_AUTH_SOCK` に別の接続先も設定できます。
利用できる鍵を 1 本以上読み込んでください。オフラインの説明は `fleetsh docs proxies` にあります。

<a id="forwarding"></a>

## 3 種類の SSH ポート転送

既存の SSH ホストに、名前付きの設定を追加します。

```toml
[[hosts.hk1.forwards]]
name = "web"
type = "local"
listen = "127.0.0.1:8080"
destination = "127.0.0.1:80"

[[hosts.hk1.forwards]]
name = "reverse"
type = "remote"
listen = "127.0.0.1:9000"
destination = "127.0.0.1:3000"

[[hosts.hk1.forwards]]
name = "socks"
type = "dynamic"
listen = "127.0.0.1:1080"
```

| 種類 | 待ち受ける場所 | 接続先へ通信する場所 |
| --- | --- | --- |
| `local` | このコンピューター | VPS |
| `remote` | VPS | このコンピューター |
| `dynamic` | このコンピューターの SOCKS5 | VPS。接続先は SOCKS クライアントが指定 |

```sh
fleetsh forward hk1 --dry-run
fleetsh forward hk1 web
fleetsh forward hk1 socks
fleetsh forward hk1
fleetsh forward hk1 --dry-run --json
```

名前を指定すると 1 件、省略すると全設定を起動します。ドライランは設定を読むだけで、
SSH 接続、パスワード参照、ポートの待ち受けを行いません。実際の転送は前面で動作し、
Ctrl+C または切断時に待ち受けと通信を閉じます。起動失敗時は全設定を解除します。
他のコマンドでは自動起動しません。通常の `edit` は転送設定を保持します。

名前はホスト内で一意にします。待ち受け先には IP または `localhost` と
0–65535 のポートを指定します。0 は空きポートを選び、実際のアドレスを表示します。
接続先のポートは 1–65535 です。IPv6 は `[::1]:1080` のように角括弧を使います。
`--connect-timeout` は接続、起動、接続先へのダイヤルに適用されます。開始済みの通信に
固定の有効期限はありません。セッションごとの同時接続上限は 128 です。

非公開アクセスにはループバックで待ち受けます。`0.0.0.0`/`::` は他の端末へ公開します。
動的転送は認証なしの SOCKS5 TCP CONNECT で、BIND と UDP には対応しません。
ホスト名は VPS 側で解決します。リモート待ち受けは sshd の
`AllowTcpForwarding`/`GatewayPorts` に従い、サーバー設定は変更しません。
オフラインの例は `fleetsh docs forwarding` にあります。

独自コマンド別名と OpenSSH の入出力は[ロードマップ](../roadmap/)にあります。

## ファイルと権限

ホスト一覧とホスト鍵の保存先は[上記](#storage)を参照してください。
Unix ではファイルを 0600、新規ディレクトリを 0700 で作成します。
Windows ではユーザーディレクトリの ACL を使います。
ホスト一覧と `known_hosts` は通常のファイルに限り、シンボリックリンクは拒否します。
秘密値は OS ストアに保持します。
