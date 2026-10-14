// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func unknownHostError(host string) error {
	return fmt.Errorf("Host %q is not in this inventory.\nNext: run fleetsh ls to list hosts, or fleetsh add %s --host ADDRESS --user USER to add it.", host, host)
}
func readableError(err error) string {
	message := err.Error()
	var missing missingInventoryError
	if errors.As(err, &missing) {
		return message
	}
	var usage usageError
	if errors.As(err, &usage) {
		return message
	}
	var file *os.PathError
	if errors.As(err, &file) {
		reason := "Check that the path is correct and you have permission to access it."
		if errors.Is(err, os.ErrNotExist) {
			reason = "The file or its parent directory does not exist."
		}
		if errors.Is(err, os.ErrPermission) {
			reason = "Access was denied. Check the file and directory permissions."
		}
		return strings.Replace(message, file.Error(), fmt.Sprintf("Cannot access local file %q. %s", file.Path, reason), 1)
	}
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "passwords do not match"), strings.Contains(lower, "secrets do not match"):
		return message + "\nNext: repeat the command and enter the same value at both hidden prompts. Nothing was saved."
	case strings.Contains(lower, "credential store"):
		return message + "\nNext: unlock your system credential store and retry. On Linux, use an unlocked Secret Service in a user D-Bus session."
	case strings.Contains(lower, "credential not found"):
		return message + "\nNext: for an SSH password, run fleetsh edit HOST --save-password; otherwise check the reference with fleetsh show HOST."
	case strings.Contains(lower, "interactive terminal"), strings.Contains(lower, "unattended"):
		return message + "\nNext: run the command in a terminal, or configure an existing credential reference for unattended execution."
	case strings.Contains(lower, "inventory toml"):
		return message + "\nNext: run fleetsh docs config and check the configuration format. Store passwords through edit HOST --save-password."
	case strings.Contains(lower, "lock timed out"):
		return message + "\nNext: wait for the other fleetsh process to finish, then retry."
	default:
		return message
	}
}
func remoteFailure(host, kind, detail string) string {
	next := "Next: run fleetsh doctor " + host + " to check the connection."
	reason := "The connection to " + host + " failed."
	switch kind {
	case "transfer":
		reason = "The file transfer for " + host + " could not finish."
		next = detail + "\nNext: check the paths and file permissions. Existing files need an explicit --overwrite."
	case "preparation":
		reason = "The local SSH login setup for " + host + " is not ready."
		next = "Next: review the authentication message below; run fleetsh show " + host + " to check its settings."
	case "authentication":
		reason = "The server did not accept the SSH login for " + host + "."
		next = "Next: check the username with fleetsh show " + host + ". If you use a password, replace the local saved copy with fleetsh edit " + host + " --save-password."
	case "hostkey":
		reason = "The SSH host key for " + host + " is unknown or has changed."
		next = "Next: run fleetsh hostkey show " + host + " and verify the fingerprint independently. Use fleetsh ssh " + host + " to accept a verified new key."
	case "proxy":
		reason = "The proxy or jump-host connection for " + host + " failed."
	case "connection_timeout":
		reason = "The SSH connection to " + host + " timed out."
		next += " Check the address, port and firewall; use --connect-timeout 60s on a slow connection."
	case "timeout":
		reason = "The remote operation on " + host + " exceeded its time limit."
		next = "Next: check what completed before retrying. Use --timeout with a longer duration if needed."
	case "canceled":
		reason = "The operation on " + host + " was canceled."
		next = "Next: check the remote state before running the command again."
	case "command":
		reason = "The remote command on " + host + " failed."
		next = "Next: review its output and exit status above before retrying."
	}
	if kind == "connection" && strings.Contains(strings.ToLower(detail), "no such host") {
		reason = "The hostname for " + host + " could not be resolved. Check the address and DNS."
	}
	if kind == "connection" && strings.Contains(strings.ToLower(detail), "refused") {
		reason = "The SSH port on " + host + " refused the connection. Check that SSH is running and the port is correct."
	}
	return reason + "\n" + next
}
