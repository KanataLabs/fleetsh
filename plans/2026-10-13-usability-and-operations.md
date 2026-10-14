# Fleetsh usability and daily operations implementation plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make host setup, failures, monitoring and common fleet operations understandable and usable from a terminal while preserving scripted behavior.

**Architecture:** Reuse Cobra commands, inventory transactions, native credential storage and the existing SSH manager/executor. Interactive editing changes existing flags before the normal mutation path; network diagnostics observe the existing connection. No daemon, playbook framework or central service.

**Tech Stack:** Go, Cobra, TOML, x/crypto/ssh; a small SFTP client dependency if the full scope is selected.

---

## Scope and execution

The user approved the preceding functional suggestions and requested human-readable errors. The user explicitly selected all eight suggestions. Apply the repository's minimal-design and whole-stage validation rules instead of per-file test/commit cycles. Existing authorization permits normal organization App commits, pushes, releases and Pages publication. Never expose personal GitHub identity or secret values.

1. Interactive host add/edit and success feedback: internal/cli/inventory.go, host_mutation.go and a new interactive.go. Only a real terminal prompts; --json/nonterminal operations retain explicit validation. Tests inject line input and secret prompts, cover cancellation and unchanged inventory.
2. Human-readable errors: improve CLI syntax, local file/credential errors and remote result messages, preserving exit codes, output redaction and JSON structure. Include the failing host, reason and an actionable next command.
3. doctor HOST: configuration path, resolved route, authentication preparation, network/host key/login and small read-only remote check. Reuse SSH manager, never automatically trust unknown keys. Tests use local SSH fixtures for success, wrong password, unknown keys and refused connections.
4. stats --table and --watch DURATION: retain existing default snapshot/JSON contract, add a compact table with timestamp and root disk space, repeated snapshots with Ctrl+C cancellation, and independent failed-host results. No overlapping refreshes; minimum interval 1s; JSON watch emits one report per line.
5. Named commands/run: optional [commands.NAME] with command, sudo, timeout; require explicit selector/name, dry-run shows resolved targets, no implicit retries. Add inventory validation and fixture tests.
6. Shell completion: Cobra-generated Bash/Zsh/Fish/PowerShell scripts and inventory-based host/group/command/topic completion; no credential access or remote connection.
7. Maintenance checks: reboot-required (explicitly supported OS checks), update --check-reboot and reboot --batch-size with health verification and stop after failed batch. Do not automatically repeat remote mutations.
8. OpenSSH import/export: literal supported entries, dry-run preview, explicit conflict errors, unsupported directives reported; never import executable ProxyCommand or plaintext secrets. Atomic inventory updates and no silent overwrites. Export host connection data only; warn about settings OpenSSH cannot represent.
9. SFTP upload/download: one file per selected host, explicit remote/local paths, no recursive or agent dependency, no overwrite by default, dry-run without connections, bounded workers, cancellation and per-host results. Group downloads use host-specific local destinations. Enforce existing host-key/auth/proxy rules.
10. English default CLI/help and separate en/ja/zh documentation, embedded offline guides, updated actual-feature roadmap.

## Unified verification and release

After the complete implementation and self-review, run gofmt, git diff --check, go test ./... -count=1 -timeout 120s, go vet ./..., npm test, and any new dependency license checks. Concentrate discovered fixes, then rerun affected checks. Commit/push with Kanata Labs identity and Publisher App; wait for multi-platform CI, publish documentation, tag/build a new alpha release, validate package and upload hashes, and test the public npx launcher using isolated inventories. Do not use or mutate the user's real VPS configuration/passwords for fixtures.