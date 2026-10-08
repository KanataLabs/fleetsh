# Host and credential usability

The CLI should create and associate a password credential while adding a password-authenticated SSH host. Users should not need to invent a reference or run credential add first. Keep credential commands for shared credentials, key passphrases, proxies and explicit management.

- Add with password authentication and no explicit reference prompts twice using hidden terminal input, creates a unique fleetsh-ssh- reference and saves the host and reference together.
- Edit --save-password creates a new private reference rather than overwriting a secret potentially shared by another host. Switching to password auth without an explicit reference follows the same flow.
- --no-save-password retains per-connection prompting. Automation can supply an existing reference or explicitly choose this option.
- Keep the fleetsh OS service and Windows fleetsh: target prefix for backward compatibility. New and replaced entries receive a Created by fleetsh (KanataLabs) comment or label.
- Validate before prompting or saving. On inventory persistence failure, remove the newly created secret; report cleanup failure without exposing the value.
- Existing --groups replaces all memberships. Add --add-groups and --remove-groups to edit, retaining unrelated memberships and deduplicating additions. One host can remain in several groups.
- Test failed prompts, storage failures, validation, rollback, shared-reference isolation, non-terminal behavior, native metadata and group selectors. Update English, Japanese and Chinese separately; English remains the default.
