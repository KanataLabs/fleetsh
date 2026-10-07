// SPDX-License-Identifier: GPL-3.0-only
package main

import (
	"os"

	"github.com/KanataLabs/fleetsh/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version, os.Stdout, os.Stderr))
}
