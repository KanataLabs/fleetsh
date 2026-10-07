// SPDX-License-Identifier: GPL-3.0-only
// Package cli implements the foundation commands.
package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/KanataLabs/fleetsh/internal/config"
)

const help = `fleetsh — lightweight, agentless VPS fleet management

Usage:
  fleetsh help
  fleetsh version
  fleetsh init [--config PATH]

Commands available now:
  help      Show this help
  version   Print the build version
  init      Create an example configuration without overwriting existing files

SSH, inventory management, credentials and remote execution are planned.
See https://kanatalabs.com/fleetsh/roadmap/

Copyright (C) 2026 KanataLabs contributors.
Licensed under GPL-3.0-only. No warranty; see LICENSE in the source repository.
`

// Run returns 0 on success and 2 on local usage or configuration errors.
func Run(args []string, version string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(out, help)
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		if len(args) != 1 {
			fmt.Fprintln(errOut, "fleetsh: help does not accept arguments")
			return 2
		}
		fmt.Fprint(out, help)
		return 0
	case "version", "--version":
		if len(args) != 1 {
			fmt.Fprintln(errOut, "fleetsh: version does not accept arguments")
			return 2
		}
		fmt.Fprintf(out, "fleetsh %s\n", version)
		return 0
	case "init":
		flags := flag.NewFlagSet("init", flag.ContinueOnError)
		flags.SetOutput(errOut)
		path := flags.String("config", "", "configuration file path (default: OS user config directory)")
		if err := flags.Parse(args[1:]); err != nil {
			if err == flag.ErrHelp {
				return 0
			}
			return 2
		}
		if flags.NArg() != 0 {
			fmt.Fprintln(errOut, "fleetsh: init does not accept positional arguments")
			return 2
		}
		var err error
		if *path == "" {
			*path, err = config.DefaultPath()
		}
		if err == nil {
			err = config.Init(*path)
		}
		if err != nil {
			fmt.Fprintf(errOut, "fleetsh: %v\n", err)
			return 2
		}
		fmt.Fprintf(out, "Created %s\nNo credentials were stored.\n", *path)
		return 0
	default:
		fmt.Fprintf(errOut, "fleetsh: unknown command %q; run 'fleetsh help'\n", args[0])
		return 2
	}
}
