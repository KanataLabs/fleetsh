// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
)

const documentationSiteURL = "https://kanatalabs.com/fleetsh/"
const documentationURL = documentationSiteURL + "en/"

// usageError distinguishes command syntax errors from configuration/remote failures.
type usageError struct {
	command *cobra.Command
	cause   error
}

func (e usageError) Error() string { return e.cause.Error() }
func (e usageError) Unwrap() error { return e.cause }

type commandGuide struct {
	long, example, page string
}

var commandGuides = map[string]commandGuide{
	"fleetsh doctor": {
		long: `Diagnose one existing host without changing configuration or trusted host keys.
HOST is required. Shows the inventory path, effective proxy/jump route and login
preparation, then verifies the network connection, host key, authentication and
a small remote health check. Unknown keys require a verified first login with ssh.
Failures include a suggested next command. --json emits structured checks.`,
		example: `  fleetsh doctor hk1
  fleetsh doctor hk1 --connect-timeout 30s
  fleetsh doctor hk1 --json`,
		page: "operations/#diagnostics",
	},
	"fleetsh run": {
		long: `Execute a saved [commands.NAME] command on selected hosts.
SELECTOR and NAME are required. The command can supply sudo and timeout defaults;
explicit --sudo/--timeout flags override them. --dry-run lists targets and the
resolved command without SSH or credential-store access. No automatic retries.`,
		example: `  fleetsh run '@web' docker-update --dry-run
  fleetsh run '@web' docker-update --parallel 2
  fleetsh run hk1 disk --json`,
		page: "operations/#saved-commands",
	},
	"fleetsh completion": {
		long: `Generate a completion script for bash, zsh, fish or powershell.
SHELL is required. Host, group, saved-command and documentation completion reads
only local inventory data; it never reads secrets or connects to a VPS.
Install a generated script for the native fleetsh command (npm install -g fleetsh
also provides that command). Completing arbitrary npx invocations is shell-dependent.`,
		example: `  fleetsh completion powershell > fleetsh-completion.ps1
  fleetsh completion bash > fleetsh-completion.bash
  fleetsh completion zsh > _fleetsh`,
		page: "operations/#completion",
	},
	"fleetsh reboot-required": {
		long: `Check whether Linux hosts report a required restart; defaults to @all.
Debian/Ubuntu use the reboot-required marker. Other distributions can use
needs-restarting -r if available. Unsupported or unavailable checks explicitly
report unknown, not no. This command never reboots hosts.`,
		example: `  fleetsh reboot-required
  fleetsh reboot-required '@web' --json`,
		page: "operations/#maintenance",
	},
	"fleetsh import": {
		long: `Import supported OpenSSH settings from FILE. FILE is required.
Supports literal Host aliases, wildcard defaults, HostName, User, Port, one
IdentityFile and a single ProxyJump alias. First matching values win.
Use --user USER when a host has no User setting. --dry-run previews without writes.
Existing aliases are never overwritten. Include/Match stop import; hosts requiring
ProxyCommand or unsupported expansion tokens are skipped with explanations.
Other unsupported directives are listed as notes, without echoing their values.
Imported passwords are not saved; use edit HOST to choose password authentication.`,
		example: `  fleetsh import ~/.ssh/config --dry-run
  fleetsh import ~/.ssh/config --user ubuntu
  fleetsh import ./hosts.conf --json`,
		page: "operations/#import-export",
	},
	"fleetsh export": {
		long: `Export host settings as toml (default), json or ssh-config.
Saved OS credential-store values and references are excluded; private key files
are never read. Saved command definitions are excluded. User-supplied descriptions
and connection settings are exported as entered.
Output defaults to stdout. --output FILE writes UTF-8; existing files require
--overwrite. OpenSSH output omits console-only hosts and reports unrepresented
proxy URLs and port-forward profiles. Do not treat it as a complete migration
of every OpenSSH authentication preference.`,
		example: `  fleetsh export --format json
  fleetsh export --format toml --output inventory.toml
  fleetsh export --format ssh-config --output ssh-config`,
		page: "operations/#import-export",
	},
	"fleetsh upload": {
		long: `Upload one regular LOCAL_FILE to REMOTE_FILE on selected hosts through SFTP.
SELECTOR, LOCAL_FILE and REMOTE_FILE are required. The destination parent directory
must exist and be writable by the SSH user. --sudo and directory transfers are
not supported. Existing destinations require --overwrite. --dry-run connects to
nothing and changes no files. Failed transfers may leave a partial destination;
review it before retrying. Uses the normal SSH authentication, proxy and key trust.`,
		example: `  fleetsh upload hk1 ./script.sh /tmp/script.sh --dry-run
  fleetsh upload '@web' ./script.sh /tmp/script.sh
  fleetsh upload hk1 ./script.sh /tmp/script.sh --overwrite`,
		page: "operations/#transfers",
	},
	"fleetsh download": {
		long: `Download one regular REMOTE_FILE through SFTP to LOCAL_PATH.
SELECTOR, REMOTE_FILE and LOCAL_PATH are required. For one host, LOCAL_PATH may be
a file or existing directory. For multiple hosts it is a directory containing
host-specific subdirectories; paths include a short hash to avoid alias collisions.
Existing files require --overwrite; remote directories/symlinks are not supported.
--dry-run lists exact paths without connecting or creating directories.
Failed transfers may leave a partial destination; review it before retrying.`,
		example: `  fleetsh download hk1 /var/log/syslog ./syslog.txt
  fleetsh download '@web' /var/log/syslog ./logs --dry-run
  fleetsh download '@web' /var/log/syslog ./logs --parallel 2`,
		page: "operations/#transfers",
	},
	"fleetsh": {
		long: `fleetsh manages a local VPS inventory, SSH sessions and parallel remote commands.
One Go binary; no central server or remote agent. Development build, GPL-3.0-only.

Getting started:
  Add a host (creates a missing inventory), then connect to verify its SSH fingerprint.
  Use --help on any command for flags and examples, or docs TOPIC for offline guides.

Selectors (ls, exec, alive, stats, update, reboot):
  hk1          one host alias
  hk1,sg1      several hosts
  '@web'       every host in group web
  '@web,sg1'   a union of groups and hosts, with duplicates removed
  '@all'       every host; --tag production narrows the selection
  Quote selectors containing @ in PowerShell. Console-only hosts skip remote work.

Configuration:
  --config PATH selects a local TOML inventory; add creates it automatically if missing.
  init is optional: it creates an empty commented example without overwriting.
  Otherwise use the OS user config directory. Run docs config for exact locations.
  Passwords live in the OS credential store; the inventory contains references.
  Saved the wrong password? Run fleetsh edit HOST --save-password, then ssh HOST.
  In a terminal, edit HOST opens a menu. Scripts require a change option.
  Use show HOST to inspect existing settings.

Exit codes:
  0 success; 1 remote command failure, timeout or cancellation;
  2 local configuration, arguments or credential preparation failure;
  3 SSH connection, authentication or host-key failure.
  For mixed batch failures, 1 takes precedence over 3.

Offline guides: fleetsh docs`,
		example: `  fleetsh add hk1 --host hk1.example.com --user ubuntu --auth password --groups web
  fleetsh ssh hk1
  fleetsh edit hk1 --save-password
  fleetsh ls '@web'
  fleetsh exec '@web' "uptime" --parallel 5
  fleetsh update '@all' --dry-run --sudo
  fleetsh docs passwords`,
		page: "",
	},
	"fleetsh alive": {
		long: `Check whether selected hosts can authenticate over SSH and execute a small command.
With no selector, check @all. This is SSH liveness, not an ICMP ping.
Uses configured keys, credentials, global/per-host proxies and strict host trust.
Console-only assets are skipped. Supports --tag, --parallel, --serial, --timeout,
--connect-timeout and --json with the same results/exit codes as exec.`,
		example: `  fleetsh alive
  fleetsh alive '@web' --timeout 10s
  fleetsh alive --json`,
		page: "getting-started/",
	},
	"fleetsh stats": {
		long: `Print one Linux resource snapshot per selected SSH host; default selector is @all.
Includes whoami, CPU busy percentage sampled over one second, memory/swap usage
in MiB and percentages, and free disk space from df -h -P.
Uses /proc, awk, sleep, whoami and df; no remote agent or sudo is required.
Memory usage uses MemAvailable (or free/buffers/cache on older kernels).
Swap disabled is shown as zero. CPU is total host CPU, not a container quota.
The monitor alias runs the same command. --json emits the standard batch report
with each snapshot in stdout. --table shows compact columns, including root disk
free space. --watch 5s refreshes after each completed snapshot; Ctrl+C stops it.
Failed hosts stay visible without stopping other hosts. --watch --json emits one
compact JSON report per line with a timestamp. Use docs monitoring for examples.`,
		example: `  fleetsh stats
  fleetsh stats '@web' --parallel 5 --timeout 15s
  fleetsh stats '@all' --watch 5s
  fleetsh stats hk1 --table
  fleetsh monitor hk1
  fleetsh stats --json`,
		page: "getting-started/",
	},
	"fleetsh forward": {
		long: `Start named TCP port-forward profiles from hosts.HOST.forwards in the inventory.
With NAME, start that profile; otherwise start all profiles for this exact alias.
local: listen on this machine and connect to a destination from the VPS.
remote: listen on the VPS and connect to a destination from this machine.
dynamic: offer local SOCKS5 TCP CONNECT through the VPS (no auth, BIND or UDP).
Use explicit loopback listen addresses unless broader access is intended.
Remote listening depends on sshd forwarding/GatewayPorts policy.

--dry-run displays the profiles without SSH, credentials or listeners; --json
is supported for dry-run output only. Actual forwarding uses verified SSH and
configured global/per-host proxies. It runs until Ctrl+C or SSH disconnects,
then closes listeners and streams. Failed startup rolls back every profile.
Port 0 asks for an available listen port; actual bound addresses are printed.
Up to 128 connections are active per session. Run docs forwarding for TOML.`,
		example: `  fleetsh forward hk1 --dry-run
  fleetsh forward hk1 web
  fleetsh forward hk1
  fleetsh forward hk1 socks --connect-timeout 30s
  fleetsh forward hk1 --dry-run --json`,
		page: "configuration/#forwarding",
	},
	"fleetsh init": {
		long: `Create an example TOML inventory at --config PATH or the OS user config directory.
The file starts with no active hosts and never overwrites an existing file.
The command prints the configuration path. It stores no passwords.

init is optional: add creates a missing inventory automatically while saving its first host.
Run docs config for paths, defaults and backup guidance.`,
		example: `  fleetsh init
  fleetsh init --config ./lab/config.toml`,
		page: "getting-started/",
	},
	"fleetsh version": {
		long: `Print the build version. This command needs no inventory or network connection.
Development builds may include a Git revision; there is no stable release yet.`,
		example: `  fleetsh version
  fleetsh --version`,
		page: "installation/",
	},
	"fleetsh ls": {
		long: `List the selected hosts, including their addresses, authentication modes, groups
and tags. With no selector, list every host. --tag filters the selected set.

Selectors accept host aliases, comma-separated unions, @GROUP and @all.
Unknown hosts/groups or an empty filtered selection are errors.
Use --json for a machine-readable array; no secret values are included.`,
		example: `  fleetsh ls
  fleetsh ls '@web' --tag production
  fleetsh ls 'hk1,sg1,@web' --json`,
		page: "getting-started/",
	},
	"fleetsh show": {
		long: `Show one host's local configuration as JSON, including credential references.
Secret values are never included. This command does not connect to the host.
HOST must be an exact inventory alias; groups and selectors are not accepted.`,
		example: `  fleetsh show hk1
  fleetsh show hk1 --config ./lab/config.toml`,
		page: "configuration/",
	},
	"fleetsh add": {
		long: `Add a new host alias; a missing inventory and its parent directories are created automatically.
init is optional. Existing inventories are read and preserved; invalid or unreadable
files are reported rather than replaced. Missing required options, invalid host
settings and cancelled password input do not create an inventory file.
Use the same --config PATH on every command for a custom inventory.

In an interactive terminal, add HOST asks for a missing address, username or key
path. In scripts and --json mode, provide required options explicitly.

Required arguments/options (values must not be empty):
  HOST          Local inventory alias, not the hostname or IP address.
  --host ADDRESS  Hostname or IP address; required for every host.
  --user USER     Required for SSH hosts (--connection ssh is the default).
  --key PATH      Required when --auth key is selected.

Console-only hosts still need --host; --user is optional for them.
Missing/empty required options are listed together with this help before reading
inventory or prompting for credentials. Invalid host options also show help.
Defaults for new hosts are port 22, SSH and password authentication.
--auth is optional. Use --auth key with --key PATH for a private key, or --auth
agent for a running SSH agent with a loaded key. Existing hosts keep their mode;
to switch one to password login, run edit HOST --auth password.

Password hosts without --credential prompt twice using hidden terminal input and
save automatically to the OS credential store. Generated references start with
fleetsh-ssh-; entries have a fleetsh namespace and creator notes.
--no-save-password leaves the reference empty and prompts on each connection.
--credential REF reuses a separately managed secret. Explicit references cannot
be combined with --save-password or --no-save-password.

A host may belong to multiple groups and tags. Aliases and group names contain
1-128 letters, digits, dots, underscores or hyphens, starting with a letter/digit.
The host alias all is reserved. Password storage needs an available OS keyring.
Run docs passwords, docs groups or docs proxies for more details.`,
		example: `  fleetsh add hk1 --host hk1.example.com --user ubuntu --auth password --groups asia,web
  fleetsh add sg1 --host sg1.example.com --user ubuntu --auth key --key ~/.ssh/id_ed25519
  fleetsh add lab1 --host lab1.example.com --user ubuntu --auth password --no-save-password
  fleetsh add console1 --host console.example.com --connection console-only --description "Provider console"`,
		page: "getting-started/#required-host-options",
	},
	"fleetsh edit": {
		long: `Change only the supplied fields of an existing host; other fields are preserved.
HOST is a required existing alias. The host alias remains the same.
In an interactive terminal, bare edit HOST opens a field-selection menu.
In scripts or --json mode, at least one change option is required; otherwise
this command displays help and does not change the inventory.
Use show HOST to inspect settings. Omitted flags keep their saved values.

Saved the wrong password? Run edit HOST --save-password, enter the correct server
password twice at the hidden prompts, then ssh HOST to verify it. This replaces
only the locally saved password, not the password on the server. --auth password
alone keeps an existing password reference. To switch from key/agent and save a
password explicitly, use edit HOST --auth password --save-password.

The resulting host must still have --host, --user for SSH, and --key with key
authentication. You can omit these flags if their saved values meet the requirements.
Changing --connection to ssh or --auth to key may require --user or --key.
Clearing a required value reports the missing option and shows this help.

--add-groups LIST appends memberships; --remove-groups LIST removes selected ones.
They can be combined for different names. --groups LIST replaces every membership;
--groups "" clears them. Do not combine --groups with add/remove group flags.
--tags LIST similarly replaces tags. Each host may belong to several groups.

--save-password prompts twice and associates a newly generated credential reference.
Switching to password authentication without a reference saves automatically.
--no-save-password detaches the reference and prompts on future connections.
Detached/previous credentials remain in the OS store; remove them explicitly only
when no other host needs them. Group-only edits never prompt for a password.`,
		example: `  fleetsh edit hk1 --save-password
  fleetsh ssh hk1
  fleetsh edit hk1 --auth password --save-password
  fleetsh edit hk1 --add-groups web,production
  fleetsh edit hk1 --remove-groups staging --add-groups production
  fleetsh edit hk1 --groups asia,web
  fleetsh edit hk1 --groups ""
  fleetsh edit hk1 --auth password --no-save-password`,
		page: "configuration/",
	},
	"fleetsh rm": {
		long: `Remove one host alias from the local inventory. No remote action is performed.
Associated OS credentials and trusted host keys are retained because other hosts
may share them. Use credential rm or hostkey reset separately when appropriate.`,
		example: `  fleetsh rm hk1`,
		page:    "configuration/",
	},
	"fleetsh ssh": {
		long: `Open an interactive SSH shell for one inventory alias, with terminal support.
Supports agent, private-key and password authentication, ProxyJump and SOCKS5/HTTP/HTTPS proxies.
HOST is an exact alias; group selectors and --json are not supported.

On the first connection, independently verify the displayed fingerprint before
accepting it. Trust is saved in known_hosts beside the selected inventory.
Unknown keys require a real terminal; changed keys are always rejected.
Run docs troubleshooting for host-key and authentication issues.`,
		example: `  fleetsh ssh hk1
  fleetsh ssh hk1 --connect-timeout 60s
  fleetsh ssh hk1 --config ./lab/config.toml`,
		page: "getting-started/",
	},
	"fleetsh exec": {
		long: `Execute COMMAND through the remote shell on every selected SSH host immediately.
Both arguments are required: SELECTOR is a host alias or group; COMMAND is the
remote shell command. Use '@all' to explicitly select every host.
Quote the entire command as one argument; quote @ selectors in PowerShell.
Console-only hosts are reported as skipped. Commands receive no interactive stdin.

--parallel accepts 1-256 workers; default is the inventory setting (normally 10).
--serial runs one host at a time and cannot be combined with --parallel.
--timeout sets each command's deadline (normally 30m); --connect-timeout sets the
connection deadline (normally 10s). Positive durations include 30s, 5m and 1h.
--sudo wraps the whole command and uses passwordless sudo or sudo_credential.

--json emits per-host results and a summary; output is bounded at 4 MiB per stream
per host. Inspect each result and the process exit code. Failed commands are not
automatically retried. Run docs exec for automation and shell quoting examples.`,
		example: `  fleetsh exec '@all' "whoami"
  fleetsh exec hk1 "df -h"
  fleetsh exec '@web' "uptime" --parallel 5 --timeout 30s
  fleetsh exec 'hk1,sg1' "id -un && uname -s" --serial --json
  fleetsh exec '@all' "systemctl status nginx --no-pager" --sudo --tag production`,
		page: "getting-started/",
	},
	"fleetsh update": {
		long: `Detect /etc/os-release on all selected SSH hosts and plan Linux package upgrades.
Supports Debian/Ubuntu (apt-get), RHEL/Rocky/AlmaLinux/Fedora (dnf), CentOS (dnf/yum)
and Arch/Manjaro (pacman). If any preflight fails, no updates are dispatched.

--dry-run connects and probes the OS, then shows planned commands without updating.
Without --dry-run, interactive confirmation is required; --yes confirms explicitly
for automation. --sudo uses passwordless sudo or the host's sudo_credential.
--dist selects apt-get dist-upgrade on Debian/Ubuntu; other distributions keep
their normal upgrade command. Defaults are shared with exec.

--check-reboot checks restart requirements after each successful update.
Updates do not reboot hosts automatically. Run reboot-required separately or
read docs patching for a fleet workflow.`,
		example: `  fleetsh update '@all' --dry-run --sudo
  fleetsh update '@web' --sudo --parallel 5 --timeout 30m
  fleetsh update '@web' --sudo --yes --json
  fleetsh update hk1 --dist --sudo --dry-run`,
		page: "cases/patch-all-vps/",
	},
	"fleetsh reboot": {
		long: `Reboot selected Linux SSH hosts and wait for a changed boot ID plus a successful
uptime probe. A reconnect alone does not count as a verified reboot.

--dry-run shows the targets without connecting or rebooting. Otherwise interactive
confirmation is required; --yes confirms explicitly for automation. Use --sudo
when the SSH user needs privilege elevation. Console-only hosts are skipped.
Default concurrency is 2; --serial or --parallel overrides it. --wait-timeout is
the post-reboot verification deadline (default 5m) and is separate from --timeout.
--batch-size N waits for the entire batch to finish its boot-ID/health checks
before starting the next. Any failed batch stops subsequent reboots.`,
		example: `  fleetsh reboot '@web' --dry-run --sudo
  fleetsh reboot '@web' --sudo --serial --wait-timeout 10m
  fleetsh reboot hk1 --sudo --yes --json`,
		page: "getting-started/",
	},
	"fleetsh credential": {
		long: `Manage reusable secrets in Windows Credential Manager, macOS Keychain or Linux
Secret Service. The inventory stores references; secret values stay in the OS store.
New entries have a fleetsh namespace and creator notes. No plaintext fallback.

For a host's SSH password, add --auth password or edit --save-password saves it
automatically. These commands remain useful for shared SSH passwords, encrypted-key
passphrases, sudo passwords and proxy credentials. Run docs passwords for examples.`,
		example: `  fleetsh credential ls
  fleetsh credential add shared-login
  fleetsh edit hk1 --auth password --credential shared-login`,
		page: "configuration/#passwords",
	},
	"fleetsh credential add": {
		long: `Save a reusable secret under REFERENCE using hidden terminal input twice.
An inventory must exist: add a host first, or use init for an empty example.
Reference names follow the same naming rules as host aliases.
Use --replace to overwrite an existing secret; every host using it sees the change.

For proxy authentication, enter username:password at the prompt. For an encrypted private key,
enter its passphrase. For sudo, enter the sudo password. Do not put secret values
in command arguments or inventory files. The native store must be available.`,
		example: `  fleetsh credential add shared-login
  fleetsh credential add admin-sudo
  fleetsh edit hk1 --sudo-credential admin-sudo
  fleetsh credential add local-proxy
  fleetsh credential add shared-login --replace`,
		page: "security/",
	},
	"fleetsh credential ls": {
		long: `List credential references tracked by, or used by hosts in, the selected inventory.
It does not enumerate all OS-store entries or verify that each reference exists.
Secret values are never shown. --json emits an array of reference names.`,
		example: `  fleetsh credential ls
  fleetsh credential ls --json
  fleetsh credential ls --config ./lab/config.toml`,
		page: "configuration/#passwords",
	},
	"fleetsh credential rm": {
		long: `Delete REFERENCE from the OS credential store and the inventory's reference index.
Host fields referring to it remain unchanged and will need editing before use.
Check every host using a shared reference before deleting it. No remote password
is changed by this command; future authentication may fail if it still needs it.`,
		example: `  fleetsh show hk1
  fleetsh edit hk1 --save-password
  fleetsh credential rm old-login`,
		page: "configuration/#passwords",
	},
	"fleetsh hostkey": {
		long: `Inspect or remove explicit SSH trust stored in known_hosts beside the inventory.
The first connection requires fingerprint verification. Changed keys are rejected.
Trust is associated with the target address and port, so aliases may share it.
Run docs troubleshooting for the host-key replacement workflow.`,
		example: `  fleetsh hostkey show hk1
  fleetsh hostkey reset hk1`,
		page: "security/",
	},
	"fleetsh hostkey show": {
		long: `Show locally trusted SSH fingerprints for this host's address and port.
This command does not connect to the server and cannot show its current live key.
Compare fingerprints using an independently trusted channel before trusting changes.`,
		example: `  fleetsh hostkey show hk1
  fleetsh hostkey show hk1 --config ./lab/config.toml`,
		page: "security/",
	},
	"fleetsh hostkey reset": {
		long: `Remove local trust for this host's address and port after independent verification
of a replacement fingerprint. Requires interactive confirmation, or explicit --yes.

Reconnect using ssh to inspect and trust the replacement key. --yes only confirms
removal; it does not trust a new key. Shared/wildcard records may need manual review.`,
		example: `  fleetsh hostkey show hk1
  fleetsh hostkey reset hk1
  fleetsh ssh hk1`,
		page: "security/",
	},
	"fleetsh docs": {
		long: `Read English documentation embedded in the binary, without an inventory, network
connection or credential store. With no topic, list the available offline guides.
Run docs TOPIC to read a guide. Output is plain text, including when --json is set.
Topics: quickstart, config, passwords (alias: password), groups, selectors, exec,
proxies, patching, monitoring, forwarding, diagnostics, commands, transfers,
troubleshooting.
With npm, use npx fleetsh docs password (keep the space before docs).
To correct a saved password: npx fleetsh edit HOST --save-password.

The online documentation has separate English, Japanese and Chinese editions.
English is the default. Links below are printed for reference; no browser is opened.`,
		example: `  fleetsh docs
  fleetsh docs quickstart
  fleetsh docs config
  fleetsh docs password
  fleetsh docs passwords
  fleetsh docs groups
  fleetsh docs patching`,
		page: "getting-started/",
	},
}

func documentCommands(cmd *cobra.Command) {
	if guide, ok := commandGuides[cmd.CommandPath()]; ok {
		cmd.Long = guide.long + "\n\nDocumentation: " + documentationURL + guide.page
		cmd.Example = guide.example
	}
	if validate := cmd.Args; validate != nil {
		cmd.Args = func(command *cobra.Command, args []string) error {
			if err := validate(command, args); err != nil {
				if strings.Contains(err.Error(), "arg(s)") {
					words := strings.Fields(command.Use)
					requirements := strings.Join(words[1:], " ")
					if requirements == "" {
						requirements = "no positional arguments"
					}
					err = fmt.Errorf("%s requires %s; you provided %d argument(s). Run %s --help for examples.", command.Name(), requirements, len(args), command.CommandPath())
				}
				return usageError{command: command, cause: err}
			}
			return nil
		}
	}
	cmd.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return usageError{command: command, cause: err}
	})
	for _, child := range cmd.Commands() {
		documentCommands(child)
	}
}
