# Contributing to fleetsh

Start with the [roadmap](docs/en/roadmap.md), [architecture](docs/en/architecture.md) and
[security requirements](docs/en/security.md). Discuss large changes in a GitHub issue
before implementation. Small fixes can go directly to a pull request.

Use Go 1.27 or newer. Keep changes focused and run:

```sh
gofmt -w cmd internal
go test ./...
go vet ./...
go build ./cmd/fleetsh
```

Add tests for credential handling, selector behavior, transport failures,
cancellation and filesystem safety. Use local test SSH servers rather than real
VPS credentials. Test new dependencies for Windows, macOS and Linux compatibility.

Write public documentation in English; the product brief and Chinese introduction
may remain in Chinese. Clearly distinguish implemented behavior from planned
behavior. Edit documentation in `docs/` on `main`, never directly on `gh-pages`.

Never commit real inventory, passwords, private keys or access tokens. Use reserved
example domains and private/documentation IP addresses in examples.

Contributions are licensed under GPL-3.0-only. Preserve the license and copyright
notices. There is no CLA requirement.

Documentation has separate English (docs/en/) and Chinese (docs/zh/) trees.
Update both language versions when changing supported behavior. Keep each page's
locale, lang, page_key and permalink aligned with its counterpart. Relative
documentation links should stay in the current language. Shared navigation labels
live in docs/_data/locales.yml; legacy root pages only redirect.
