# Single-package npm launcher

The approved distribution adds one public npm package, `fleetsh`, to the existing
Go binary releases. The package is GPL-3.0-only and requires Node.js 22 or newer.
There are no runtime dependencies, platform-specific npm packages or install hooks.

The launcher detects the Node process platform and architecture, downloads the
matching pinned GitHub Release archive on first use, checks the SHA256 embedded
in the npm package, extracts only the executable in memory, verifies its pinned
binary digest, then atomically installs it in the user's cache. Later invocations
verify the cached digest and run without downloading. A writable cache may be
selected with `FLEETSH_NPM_CACHE`. Inventory and credential paths remain unchanged.

Archives and output sizes are bounded. Extraction never writes archive paths.
Concurrent first runs may download twice, but atomically install a complete
directory. Corrupted caches fail with the exact directory to remove. Unsupported
platforms fail before downloading. Downloads use HTTPS and explicit timeouts.

The Go child inherits terminal handles, arguments and working directory. The
launcher forwards termination signals and returns the child's exit status;
installation diagnostics use stderr so JSON stdout stays clean.

The npm version matches the Go release. The initial package is
`0.1.0-alpha.1`, published under the `alpha` dist-tag. It reuses existing immutable
release artifacts. Preparation verifies all six archives against release checksums
and embeds both archive and binary digests. Future versions regenerate this
manifest before packing; launcher-only fixes require a separate npm version and
an explicitly modeled binary-version mapping if one becomes necessary.

Verification covers real release extraction for all six targets, checksum failure,
cache reuse and concurrency, bounded downloads, unsupported platforms, literal
argument forwarding, stdio and child exit/signal behavior. CI runs the Node suite
and a fresh-cache packaged-binary smoke test on Windows, macOS and Linux.
Documentation remains English-first with separate Japanese and Chinese pages.
