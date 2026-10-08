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

The single npm package wraps the released Go binary. It requires Node.js 22 or
newer and has no runtime dependencies or install hooks. Run `npm test` and
`npm pack --pack-destination dist` to verify changes. CI tests the launcher and
packed artifacts on three operating systems.

The npm launcher version is independent of the Go version. Normal Go releases
do not require npm publications. The Release workflow calls
`node scripts/build-release-manifest.mjs vVERSION dist` after building the six
archives, then publishes `fleetsh-manifest.json` alongside archives and SHA256SUMS.
This manifest is required for automatic selection of future releases and contains
both archive and binary hashes. If publishing locally with the Publisher App,
upload it with the other artifacts. Draft/incomplete releases must stay unpublished
until all assets are ready.

The launcher prefers stable releases and falls back to previews only if no stable
release exists. Release metadata is cached for one hour; exact Go tags, bundled
selection, refresh and offline modes are described in the installation pages.
CI pins the bundled tag for deterministic binary smoke tests; selector tests use
controlled fixtures. Use `node scripts/npm-release-manifest.mjs vVERSION` only to
refresh the embedded fallback intentionally; it does not change the npm version.
Bump `package.json` when changing the launcher, then publish under `latest`. The launcher and Go program have independent
release lifecycles. npm authentication is separate from the GitHub Publisher App.
Do not commit npm credentials.

Add tests for credential handling, selector behavior, transport failures,
cancellation and filesystem safety. Use local test SSH servers rather than real
VPS credentials. Test new dependencies for Windows, macOS and Linux compatibility.

Use English as the source and default documentation language, with separate
Japanese and Chinese translations. Clearly distinguish implemented behavior from planned
behavior. Edit documentation in `docs/` on `main`, never directly on `gh-pages`.

Never commit real inventory, passwords, private keys or access tokens. Use reserved
example domains and private/documentation IP addresses in examples.

Contributions are licensed under GPL-3.0-only. Preserve the license and copyright
notices. There is no CLA requirement.

Documentation has separate English (docs/en/), Japanese (docs/ja/) and Chinese
(docs/zh/) trees. Update all language versions when changing supported behavior. Keep each page's
locale, lang, page_key and permalink aligned with its counterpart. Relative
documentation links should stay in the current language. Shared navigation labels
live in docs/_data/locales.yml; legacy root pages only redirect.

English is the default documentation language; keep translations in their own trees.
Documentation is built in CI. Automatic gh-pages publication uses the organization
App only when KANATALABS_PUBLISHER_PRIVATE_KEY is configured as an Actions secret.
Without that secret, use scripts/publish-docs.ps1 with the local Publisher utility.
The script requires a clean main checkout and uses the App for every Git operation.
Never paste a private key or installation token into this repository.
The configured site URL is https://kanatalabs.github.io/fleetsh/. GitHub currently
redirects this default URL through the organization site's custom domain.
Do not add a corporate /fleetsh redirect back to this default URL: it would loop.
