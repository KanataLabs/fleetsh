// SPDX-License-Identifier: GPL-3.0-only
package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/KanataLabs/fleetsh/internal/executor"
)

func (a *application) statsTable(rows []executor.Result, now time.Time) error {
	fmt.Fprintf(a.out, "Snapshot: %s\n", now.Format(time.RFC3339))
	w := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "HOST\tSTATUS\tCPU\tMEMORY\tSWAP\tUSER\tROOT FREE")
	for _, r := range rows {
		state := "online"
		if r.Skipped {
			state = "skipped"
		} else if !r.Success {
			state = "failed"
		}
		values := map[string]string{}
		for _, line := range strings.Split(r.Stdout, "\n") {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			switch key {
			case "USER", "CPU", "MEMORY", "SWAP", "ROOT_FREE":
				values[key] = strings.TrimSpace(value)
			}
		}
		for _, key := range []string{"CPU", "MEMORY", "SWAP", "USER", "ROOT_FREE"} {
			if values[key] == "" {
				values[key] = "-"
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.Host, state, values["CPU"], values["MEMORY"], values["SWAP"], values["USER"], values["ROOT_FREE"])
	}
	if err := w.Flush(); err != nil {
		return err
	}
	for _, r := range rows {
		if !r.Success && !r.Skipped {
			fmt.Fprintf(a.errOut, "[%s] %s\n", r.Host, remoteFailure(r.Host, r.ErrorKind, r.Error))
			if r.ErrorKind == "preparation" {
				fmt.Fprintln(a.errOut, r.Error)
			}
		}
	}
	return nil
}
