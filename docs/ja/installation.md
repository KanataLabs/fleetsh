---
title: インストールと PATH
permalink: /ja/installation/
lang: ja
locale: ja
page_key: 'installation/'
---

# インストールと PATH の設定

実行ファイルを常設ディレクトリに配置し、その**ディレクトリ**を PATH に追加すると、
どの作業ディレクトリからでも `fleetsh` を実行できます。
実行時に Go、Python、Node.js のランタイムは不要です。
現在はソースから開発版をビルドできます。Go プログラムの安定版はまだありません。

## npx で実行、または npm でインストールする

Node.js 22 以降があれば、単一の [npm パッケージ](https://www.npmjs.com/package/fleetsh)で実行できます。
npm の `latest` タグは独立したランチャー `0.1.0` を提供します。
ダウンロードする Go プログラムはプレビュー版の場合もあります。

```sh
npx fleetsh version
npx fleetsh init
npx fleetsh alive
npx fleetsh stats
npx fleetsh ssh hk1
```

一度グローバルにインストールすると、コマンドを直接実行できます。

```sh
npm install -g fleetsh
fleetsh version
fleetsh stats
```

npm はグローバル実行ファイルディレクトリにコマンドを作成します。
Windows では `npm config get prefix` の出力先、macOS/Linux ではその配下の `bin` です。
必要に応じて、このディレクトリを PATH に追加してください。
すでに PATH にあれば Go バイナリーを手動でコピーする必要はありません。

### GitHub Release の自動追従とコマンドラインでのバージョン選択

npm ランチャー `0.1.0-alpha.2` 以降では、npm と Go プログラムのバージョンは独立しています。
既定の `latest` は最新の正式な GitHub Release を優先し、正式版がない場合のみ
最後に公開されたプレビューを使用します。Go の更新には GitHub Release の公開だけが必要で、
npm パッケージの再公開はランチャー自体を変更するときだけ行います。

ランチャーのオプションは Go サブコマンドの**前**に指定します。

```sh
npx fleetsh --release latest stats
npx fleetsh --release preview stats
npx fleetsh --release bundled version
npx fleetsh --release v0.1.0-alpha.1 stats
npx fleetsh --refresh version
npx fleetsh --offline stats
npx fleetsh --launcher-help
```

| オプション | 動作 |
| --- | --- |
| `--release latest` | 正式版を優先し、正式版がない場合のみプレビューを使用（既定） |
| `--release preview` | プレビューを含めて最後に公開されたリリース |
| `--release bundled` | ランチャーで固定した元のバイナリー。現在は `v0.1.0-alpha.1` |
| `--release vVERSION` | npm のバージョンとは独立して Go のリリースを固定 |
| `--refresh` | 1 時間のメタデータキャッシュを無視して GitHub を確認 |
| `--offline` | 既存のキャッシュのみを使用し、GitHub に接続しない |
| `--launcher-help` | Go のダウンロードや起動なしでランチャーのヘルプを表示 |

グローバルインストール後も `fleetsh --release bundled version` または
`fleetsh --release v0.1.0-alpha.1 stats` を使用できます。
環境変数 `FLEETSH_RELEASE`、`FLEETSH_REFRESH=1`、`FLEETSH_OFFLINE=1` でも設定でき、
コマンドラインの指定が優先されます。

`fleetsh version` は実行中の **Go プログラム**のバージョン、
`npm ls -g fleetsh --depth=0` は **npm ランチャー**のバージョンを表示します。
`npx fleetsh@0.1.0` はランチャーだけを固定します。
Go も固定する場合は `--release vVERSION` を併用してください。
元の npm ランチャー `0.1.0-alpha.1` は Go `v0.1.0-alpha.1` に固定され、
これらのオプションをサポートしません。一度 `npm install -g fleetsh` で更新してください。

### ダウンロード、キャッシュとオフライン実行

通常のオンライン実行では、GitHub のリリース情報を最大 1 時間に一度確認します。
新しいバージョンを選ぶと自動でダウンロードし、バージョン別にキャッシュします。
リリースのマニフェストにはアーカイブと実行ファイルの SHA256 が含まれます。
GitHub のアセットダイジェストでマニフェストを検証し、続いてアーカイブとバイナリーを検証します。
キャッシュのバイナリーも実行ごとに検証します。最初のリリースでは既存の固定マニフェストを使用します。

ネットワーク障害やレート制限では、検証済みのリリース情報を再利用します。
初回は固定した元のマニフェストに戻れます。形式やハッシュの不一致はエラーになります。
オフライン実行には、選択したバイナリーを事前にダウンロードしておく必要があります。
npx はランチャーの解決やインストールで npm レジストリーに接続する場合があります。
グローバルインストールなら、その npx の処理を避けられます。
`--offline` はランチャーによる GitHub への接続を制御します。

| OS | バイナリーとリリース情報のキャッシュ |
| --- | --- |
| Windows | `%LOCALAPPDATA%\fleetsh\Cache\npm` |
| macOS | `~/Library/Caches/fleetsh/npm` |
| Linux | `$XDG_CACHE_HOME/fleetsh/npm` または `~/.cache/fleetsh/npm` |

`FLEETSH_NPM_CACHE` で別の書き込み可能なディレクトリを指定できます。
ダウンロードはホスト一覧の SSH プロキシ設定を使いません。
Windows、macOS、Linux の x64 と ARM64 をサポートします。
実行時の npm 依存、プラットフォーム別パッケージ、インストールフックはありません。
引数、対話入出力、終了コードを Go に引き渡し、設定、ホスト鍵、資格情報の保存先は変わりません。
ダウンロードと状態の通知は stderr を使うため、stdout の JSON を妨げません。

キャッシュの検証エラーでは、エラーに示されたディレクトリだけを削除してオンラインで再実行します。
Node.js または GitHub への接続がない環境では、別の環境で
[リリースアーカイブ](https://github.com/KanataLabs/fleetsh/releases)を取得し、以下の PATH の手順を使用してください。

## ソースからビルドする

ビルドには Go 1.27 以降が必要です。

```sh
git clone https://github.com/KanataLabs/fleetsh.git
cd fleetsh
go build -trimpath -o dist/fleetsh ./cmd/fleetsh
```

Windows PowerShell では次のように実行します。

```powershell
git clone https://github.com/KanataLabs/fleetsh.git
Set-Location fleetsh
go build -trimpath -o dist/fleetsh.exe ./cmd/fleetsh
```

`go install github.com/KanataLabs/fleetsh/cmd/fleetsh@latest` も利用できます。
インストール先は `go env GOBIN` の出力先です。
GOBIN が空の場合は `go env GOPATH` の `bin` サブディレクトリになります。
そのディレクトリも PATH に追加してください。

## Windows：現在のユーザーにインストールする（推奨、管理者権限不要）

ビルドしたプロジェクトのディレクトリで実行します。

```powershell
$binDir = Join-Path $env:LOCALAPPDATA 'fleetsh\bin'
New-Item -ItemType Directory -Path $binDir -Force | Out-Null
Copy-Item -LiteralPath '.\dist\fleetsh.exe' -Destination (Join-Path $binDir 'fleetsh.exe') -Force

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$entries = @($userPath -split ';' | Where-Object { $_ })
if ($entries -notcontains $binDir) {
    [Environment]::SetEnvironmentVariable('Path', (($entries + $binDir) -join ';'), 'User')
}

# 現在の PowerShell セッションにも適用
if (($env:Path -split ';') -notcontains $binDir) {
    $env:Path = $env:Path + ';' + $binDir
}

Get-Command fleetsh
fleetsh version
```

既存のユーザー PATH を保持します。ほかのターミナルは閉じて開き直してください。
Windows Terminal が起動済みの場合は、アプリケーション全体の終了と再起動が必要なことがあります。

GUI でも設定できます。「アカウントの環境変数を編集」を検索し、
ユーザーの Path を選んで編集し、新しい項目に `%LOCALAPPDATA%\fleetsh\bin` を追加します。
その後、ターミナルを再起動してください。

### Windows：すべてのユーザーにインストールする（管理者権限が必要）

管理者として起動した PowerShell で、システム用ディレクトリにコピーし、
Machine PATH を更新します。

```powershell
$binDir = Join-Path $env:ProgramFiles 'fleetsh\bin'
New-Item -ItemType Directory -Path $binDir -Force | Out-Null
Copy-Item -LiteralPath '.\dist\fleetsh.exe' -Destination (Join-Path $binDir 'fleetsh.exe') -Force

$machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
$entries = @($machinePath -split ';' | Where-Object { $_ })
if ($entries -notcontains $binDir) {
    [Environment]::SetEnvironmentVariable('Path', (($entries + $binDir) -join ';'), 'Machine')
}
$env:Path = $env:Path + ';' + $binDir
fleetsh version
```

実行ファイルの配置先は共有できますが、設定ディレクトリと OS の認証情報ストアは
引き続きユーザーごとに分かれます。

## macOS：現在のユーザーにインストールする（標準の Zsh）

プロジェクトのディレクトリで実行します。

```sh
mkdir -p "$HOME/.local/bin"
install -m 0755 dist/fleetsh "$HOME/.local/bin/fleetsh"

# 既存の .zprofile を保持して一度だけ登録
touch "$HOME/.zprofile"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.zprofile" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zprofile"

export PATH="$HOME/.local/bin:$PATH"
command -v fleetsh
fleetsh version
```

Zsh のログインシェルは `.zprofile` を読み込みます。
非ログインシェルの場合は、同じ `export PATH="$HOME/.local/bin:$PATH"` の行を
`.zshrc` に追加してください。

## Linux：現在のユーザーにインストールする（Bash）

```sh
mkdir -p "$HOME/.local/bin"
install -m 0755 dist/fleetsh "$HOME/.local/bin/fleetsh"

touch "$HOME/.bashrc"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.bashrc" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"

# .profile でログインシェルにも設定
touch "$HOME/.profile"
grep -Fqx 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.profile" ||
  printf '\n%s\n' 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.profile"

export PATH="$HOME/.local/bin:$PATH"
command -v fleetsh
fleetsh version
```

Zsh では、前述の `.zshrc` / `.zprofile` の方法を利用してください。
独自の `.bash_profile` を使っている場合は、そこから `.profile` を読み込むか、
同じ PATH 設定を追加します。Fish では `fish_add_path "$HOME/.local/bin"` を利用できます。

### macOS / Linux：システム全体にインストールする

```sh
sudo mkdir -p /usr/local/bin
sudo install -m 0755 dist/fleetsh /usr/local/bin/fleetsh
command -v fleetsh
fleetsh version
```

多くの環境では、すでに `/usr/local/bin` が PATH に含まれています。
`command -v` で見つからない場合は、シェル設定に
`export PATH="/usr/local/bin:$PATH"` を追加してください。

## 更新、確認、アンインストール

更新時は、新しい実行ファイルを同じ配置先にコピーします。PATH の再設定は不要です。
Windows では、更新前に実行中の fleetsh を終了してください。
Unix のシェルが古い場所をキャッシュしている場合は `hash -r`、
Zsh では `rehash` を実行します。

`fleetsh version` で確認します。Windows の `Get-Command fleetsh -All`、
Unix の `type -a fleetsh` で、複数の実行ファイルが存在しないか確認できます。
PATH の順序によって実行されるファイルが決まります。

アンインストール時は、実行ファイルを削除し、PATH またはシェル設定から配置先の項目を削除します。
設定と認証情報は残ります。保存済み認証情報も削除する場合は、
先に `fleetsh credential rm REFERENCE` を実行してください。
