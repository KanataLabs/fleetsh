---
title: 概要
permalink: /ja/
lang: ja
locale: ja
page_key: ''
---

# VPS ホストの管理

fleetsh は VPS の接続設定を保存し、SSH でコマンドを実行するツールです。
Windows、macOS、Linux で動作し、保存するパスワードは OS の認証情報ストアを使います。

> 現在のリリース：**v0.1.0-beta.1**。[リリースノート](https://github.com/KanataLabs/fleetsh/releases/tag/v0.1.0-beta.1)。

[インストールと PATH の設定](installation/) · [使い始める](getting-started/) ·
[fleetsh について](product/) · [ロードマップ](roadmap/)

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

## オープンな開発

[KanataLabs/fleetsh](https://github.com/KanataLabs/fleetsh) で開発し、
GPL-3.0-only で公開しています。[アーキテクチャ](architecture/)と
[セキュリティ設計](security/)も参照してください。
