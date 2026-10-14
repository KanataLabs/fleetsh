// SPDX-License-Identifier: GPL-3.0-only
package executor

import "testing"

func TestFailureExitCodePrecedence(t *testing.T) {
	for _, test := range []struct {
		name string
		rows []Result
		want int
	}{
		{"empty", nil, 0},
		{"success", []Result{{Success: true}}, 0},
		{"skipped", []Result{{Skipped: true, ErrorKind: "command"}}, 0},
		{"connection", []Result{{ErrorKind: "connection"}}, 3},
		{"local preparation", []Result{{ErrorKind: "preparation"}}, 2},
		{"local before connection", []Result{{ErrorKind: "preparation"}, {ErrorKind: "authentication"}}, 2},
		{"local after connection", []Result{{ErrorKind: "authentication"}, {ErrorKind: "preparation"}}, 2},
		{"command after local", []Result{{ErrorKind: "preparation"}, {ErrorKind: "command"}}, 1},
		{"command before local", []Result{{ErrorKind: "command"}, {ErrorKind: "preparation"}}, 1},
		{"timeout", []Result{{ErrorKind: "timeout"}}, 1},
		{"cancellation", []Result{{ErrorKind: "canceled"}}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ExitCode(test.rows); got != test.want {
				t.Fatalf("exit=%d want=%d", got, test.want)
			}
		})
	}
}
