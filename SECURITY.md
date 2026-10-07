# Security policy

fleetsh is in the foundation stage. There are no supported releases or implemented
SSH/credential features yet. Do not rely on the development branch for production
fleet operations.

Report vulnerabilities privately through
[GitHub private vulnerability reporting](https://github.com/KanataLabs/fleetsh/security/advisories/new).
Do not include credentials, real inventory or exploit details in public issues.

The [security design](docs/security.md) is a release gate for remote operations.
Windows file permissions are governed by the user's directory ACL; Unix mode bits
are not a substitute for Windows access control.
