// SPDX-License-Identifier: GPL-3.0-only
package credentials

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/term"
	"io"
	"os"
)

// ReadSecretContext restores terminal state even when a pending password read is canceled.
func ReadSecretContext(ctx context.Context, in io.Reader, out io.Writer, label string) (string, error) {
	file, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", errors.New("secret input requires an interactive terminal; configure a credential reference for automation")
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	state, err := term.GetState(int(file.Fd()))
	if err != nil {
		return "", errors.New("cannot inspect terminal state")
	}
	fmt.Fprint(out, label+": ")
	type answer struct {
		data []byte
		err  error
	}
	done := make(chan answer, 1)
	go func() { data, err := term.ReadPassword(int(file.Fd())); done <- answer{data, err} }()
	select {
	case <-ctx.Done():
		_ = term.Restore(int(file.Fd()), state)
		fmt.Fprintln(out)
		return "", ctx.Err()
	case result := <-done:
		fmt.Fprintln(out)
		if result.err != nil {
			return "", errors.New("cannot read hidden secret input")
		}
		if len(result.data) == 0 {
			return "", errors.New("secret cannot be empty")
		}
		return string(result.data), nil
	}
}
