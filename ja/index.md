---
title: 概要
permalink: /ja/
lang: ja
locale: ja
page_key: ''
---

# VPS の管理を、一つのコマンドで。

fleetsh は、個人開発者と小規模チーム向けの、Go で作られた軽量 VPS 管理ツールです。
Windows、macOS、Linux で動作する単一の実行ファイルから、SSH と OS の認証情報ストアを利用します。

> **開発版です。** ホスト管理、SSH、認証情報、プロキシ、並列コマンド実行、
> 更新、再起動を実装し、ローカルのテスト環境で検証しています。
> 安定版はまだリリースしていません。

[インストールと PATH の設定](installation/) · [使い始める](getting-started/) ·
[製品設計](product/) · [ロードマップ](roadmap/)

[利用例：すべての VPS を更新する](cases/patch-all-vps/)

```sh
fleetsh ls
fleetsh ssh hk1
fleetsh exec '@web' "uptime" --parallel 10
fleetsh update '@all' --dry-run --sudo
fleetsh reboot '@all' --parallel 2 --sudo
```

Node.js 22+ があれば、[npm パッケージ](https://www.npmjs.com/package/fleetsh)を使用できます。

```sh
npx fleetsh version
npx fleetsh stats
npm install -g fleetsh
```

ランチャーは GitHub Release を自動で追従します。`--release bundled` で元の版、
`--release vVERSION` で特定の Go 版を選べます。
[npx/npm の実行とバージョン選択](installation/)を参照してください。

## 基本方針

- 単一の実行ファイルで動作し、言語ランタイムやリモート Agent は不要。
- 中央サーバーやインフラ自動化用 DSL は不要。
- 持ち運びやすい TOML のホスト一覧を使用し、パスワードを平文で保存しない。
- 踏み台を含めてホスト鍵を厳密に検証する。
- 並列数と実行時間を制限し、ホストごとの結果を構造化して返す。
- 更新と再起動には明示的な承認を求め、再起動後の起動 ID を検証する。

## オープンな開発

[KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh) で開発し、
GPL-3.0-only で公開しています。[アーキテクチャ](architecture/)と
[セキュリティ設計](security/)も参照してください。
