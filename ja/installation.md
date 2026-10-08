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
現在はソースから開発版をビルドできます。安定版はまだありません。

## npx で実行、または npm でインストールする

Node.js 22 以降があれば、単一の npm パッケージを使用できます。

```sh
npx fleetsh@alpha version
npx fleetsh@alpha init
npx fleetsh@alpha stats
npx fleetsh@alpha ssh hk1
```

グローバルにインストールすると、コマンドを直接実行できます。

```sh
npm install -g fleetsh@alpha
fleetsh version
```

npm はグローバル実行ファイルディレクトリにコマンドを作成します。
Windows では `npm config get prefix` の出力先、macOS/Linux ではその配下の
`bin` です。Node.js のインストールで PATH に追加されていない場合は、
該当ディレクトリを追加してください。すでに PATH にあれば手動のコピーは不要です。

現在の npm バージョンは `0.1.0-alpha.1` で、`alpha` はプレビューのチャンネルです。
`npx fleetsh@0.1.0-alpha.1 version` でバージョンを固定できます。
初回の**実行**時に対応する Go バイナリーを GitHub Releases からダウンロードし、
パッケージ内で固定したアーカイブと実行ファイルの SHA256 を検証してキャッシュします。
インストールフック、実行時 npm 依存、プラットフォーム別パッケージはありません。
初回は GitHub への接続と書き込み可能なキャッシュが必要です。
以後、ランチャーは検証済みのバイナリーを再利用します。
npx によるバージョンの解決では npm レジストリーへの接続が発生する場合があります。

| OS | バイナリーのキャッシュ |
| --- | --- |
| Windows | `%LOCALAPPDATA%\fleetsh\Cache\npm` |
| macOS | `~/Library/Caches/fleetsh/npm` |
| Linux | `$XDG_CACHE_HOME/fleetsh/npm` または `~/.cache/fleetsh/npm` |

`FLEETSH_NPM_CACHE` で別のディレクトリを指定できます。
ダウンロードはホスト一覧の SSH プロキシ設定を使用しません。
Windows、macOS、Linux の x64 と ARM64 をサポートします。
引数、対話入出力、終了コードを Go プログラムに引き渡し、
ホスト一覧と資格情報の保存先は変わりません。
ダウンロードの通知は stderr に出力するため、stdout の JSON を妨げません。

キャッシュの検証エラーが出た場合は、エラーに示されたディレクトリだけを削除し、
再実行してください。Node.js や GitHub への接続がない環境では、
別の環境で[リリースアーカイブ](https://github.com/KanataLabs/fleetsh/releases)を取得し、
以下のバイナリーと PATH の手順を使用できます。

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
