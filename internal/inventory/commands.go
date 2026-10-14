// SPDX-License-Identifier: GPL-3.0-only
package inventory

import (
	"fmt"
	"strings"
	"time"
)

type Command struct {
	Command string `toml:"command" json:"command"`
	Sudo    bool   `toml:"sudo,omitempty" json:"sudo,omitempty"`
	Timeout string `toml:"timeout,omitempty" json:"timeout,omitempty"`
}

func (inv *Inventory) validateCommands() error {
	for name, c := range inv.Commands {
		if !ValidName(name) {
			return fmt.Errorf("Command %q has an invalid name. Use letters, numbers, dots, underscores or hyphens.", name)
		}
		if strings.TrimSpace(c.Command) == "" || strings.ContainsRune(c.Command, '\x00') {
			return fmt.Errorf("Command %s needs a nonempty command string.", name)
		}
		if c.Timeout != "" {
			d, err := time.ParseDuration(c.Timeout)
			if err != nil || d <= 0 {
				return fmt.Errorf("Command %s timeout must be a positive duration, for example 30s.", name)
			}
		}
	}
	return nil
}
