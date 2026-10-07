# fleetsh foundation design

The user-provided Lightweight VPS Fleet Manager brief is the product baseline.
The repository is public at KanataLabs/fleetsh, uses Go, is licensed GPL-3.0-only,
and publishes documentation from gh-pages.

Considered foundations: documentation only; a minimal executable plus documentation;
or implementing all remote features immediately. Choose the minimal executable
because it establishes repeatable builds without pretending security-sensitive
SSH or credential functionality is ready.

The first implementation exposes help, version and configuration initialization.
Configuration creation is exclusive and contains no active hosts or secret values.
Future packages implement inventory, credentials, transport, execution and actions.
The initial CLI has no third-party dependencies.

Documentation uses GitHub Pages' built-in Jekyll pipeline. Main holds editable docs
and the publishing workflow; gh-pages holds a copy of the documentation tree.
A workflow requests a Pages build after syncing because pushes made with the
repository GITHUB_TOKEN do not trigger Pages automatically.

Normalize vps/vpsctl examples to fleetsh. Use one config.toml for the initial design.
HTTP CONNECT and command aliases are v0.2; an encrypted credential fallback is
deferred pending a separate security design. Do not claim the v0.1 scope is implemented.
