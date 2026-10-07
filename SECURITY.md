# Security policy

fleetsh currently provides a development implementation; there is no stable release
or supported release line yet. Review [the security design](docs/security.md) when
contributing SSH, credential or action changes.

Report vulnerabilities through
[GitHub private vulnerability reporting](https://github.com/KanataLabs/fleetsh/security/advisories/new).
Never publish credentials, real inventory or vulnerability details in public issues.

Tests use local SSH/proxy fixtures and temporary OS-store entries. Unix permissions
are not a substitute for Windows directory ACLs.
