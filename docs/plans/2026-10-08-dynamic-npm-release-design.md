# Dynamic GitHub Release selection

The npm package is a persistent launcher with its own version. Normal Go releases
do not require npm publications. Default selection prefers GitHub's latest stable
release, falling back to the most recently published preview only when no stable
release exists. The preview selector includes stable and preview releases ordered
by publication time. Drafts and non-version tags are excluded.

Leading launcher options `--release latest|preview|bundled|vVERSION`,
`--refresh`, `--offline` and `--launcher-help` select behavior without colliding
with arguments to the Go command. Matching environment variables are supported.
`bundled` selects the original pinned binary. Explicit tags stay fixed regardless
of the installed npm version. Go help remains available; npm help adds launcher options.

Release metadata is cached for one hour in the binary cache. Fresh metadata and
exact cached tags need no GitHub query. Explicit refresh overrides the TTL.
Availability errors use the previous verified metadata, or the bundled release
on first use. Digest/format mismatches fail rather than silently downgrade.
Offline mode disables both metadata and archive downloads.

Each future GitHub Release includes `fleetsh-manifest.json` with archive and binary
SHA256 values for all six targets. The launcher verifies its digest against the
GitHub API and cross-checks each archive digest/size against the release assets.
The release workflow creates this manifest from local build artifacts and publishes
it with the existing archives; public mutations retain the Publisher App identity.
The initial alpha release uses the launcher's existing embedded manifest.

Automatic selection was introduced in npm launcher 0.1.0-alpha.2. The launcher
is now published as 0.1.0 under npm's latest tag, independently of the Go release
channel. Users run npx fleetsh or npm install -g fleetsh; its bundled binary
remains v0.1.0-alpha.1. CI pins the bundled Go tag for deterministic smoke tests, while
unit tests verify automatic channel changes, TTL, refresh, offline behavior,
explicit version selection and rejection of malformed metadata/digests.
