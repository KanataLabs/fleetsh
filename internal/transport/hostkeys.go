// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type HostKeys struct {
	Path  string
	Trust func(context.Context, string, ssh.PublicKey) bool
	mu    sync.Mutex
}

func (k *HostKeys) ensure() error {
	if err := os.MkdirAll(filepath.Dir(k.Path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(k.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		info, e := os.Lstat(k.Path)
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return errors.New("known_hosts must be a regular file")
		}
		return nil
	}
	if err != nil {
		return err
	}
	return f.Close()
}
func (k *HostKeys) Callback(ctx context.Context) ssh.HostKeyCallback {
	return func(addr string, remote net.Addr, key ssh.PublicKey) error {
		k.mu.Lock()
		defer k.mu.Unlock()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := k.ensure(); err != nil {
			return err
		}
		check, err := knownhosts.New(k.Path)
		if err != nil {
			return errors.New("cannot parse known_hosts")
		}
		err = check(addr, remote, key)
		if err == nil {
			return nil
		}
		var mismatch *knownhosts.KeyError
		if !errors.As(err, &mismatch) || len(mismatch.Want) > 0 {
			return errors.New("HOST KEY CHANGED or host identity could not be verified")
		}
		if k.Trust == nil || !k.Trust(ctx, addr, key) {
			return fmt.Errorf("unknown host key for %s (%s); connect interactively to verify and trust", addr, ssh.FingerprintSHA256(key))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		lock := flock.New(k.Path + ".lock")
		ok, err := lock.TryLockContext(ctx, 25*time.Millisecond)
		if err != nil || !ok {
			return errors.New("cannot lock known_hosts")
		}
		defer lock.Close()
		// Another process may have trusted a different key while the prompt was open.
		check, err = knownhosts.New(k.Path)
		if err != nil {
			return err
		}
		err = check(addr, remote, key)
		if err == nil {
			return nil
		}
		if !errors.As(err, &mismatch) || len(mismatch.Want) > 0 {
			return errors.New("HOST KEY CHANGED during trust confirmation")
		}
		f, err := os.OpenFile(k.Path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		existing, e := os.ReadFile(k.Path)
		if e != nil {
			f.Close()
			return e
		}
		if len(existing) > 0 && existing[len(existing)-1] != '\n' {
			if _, e = f.WriteString("\n"); e != nil {
				f.Close()
				return e
			}
		}
		_, writeErr := fmt.Fprintln(f, knownhosts.Line([]string{knownhosts.Normalize(addr)}, key))
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
}
func (k *HostKeys) Lookup(addr string) ([]knownhosts.KnownKey, error) {
	if err := k.ensure(); err != nil {
		return nil, err
	}
	check, err := knownhosts.New(k.Path)
	if err != nil {
		return nil, errors.New("cannot parse known_hosts")
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	err = check(addr, &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 22}, key)
	var ke *knownhosts.KeyError
	if errors.As(err, &ke) {
		return ke.Want, nil
	}
	return nil, err
}

// Reset refuses shared records, so one host cannot inadvertently untrust others.
func (k *HostKeys) Reset(ctx context.Context, addr string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	lock := flock.New(k.Path + ".lock")
	ok, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil || !ok {
		return errors.New("cannot lock known_hosts")
	}
	defer lock.Close()
	keys, err := k.Lookup(addr)
	if err != nil {
		return err
	}
	remove := map[int]bool{}
	for _, key := range keys {
		remove[key.Line] = true
	}
	if len(remove) == 0 {
		return errors.New("no trusted key for this host")
	}
	data, err := os.ReadFile(k.Path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	kept := []string{}
	for i, line := range lines {
		if remove[i+1] {
			fields := strings.Fields(line)
			if len(fields) > 0 && (strings.Contains(fields[0], ",") || strings.ContainsAny(fields[0], "*?")) {
				return errors.New("shared/wildcard known_hosts record requires manual editing")
			}
			if len(fields) > 0 && strings.HasPrefix(fields[0], "@") {
				return errors.New("marked known_hosts record requires manual editing")
			}
			continue
		}
		kept = append(kept, line)
	}
	f, err := os.CreateTemp(filepath.Dir(k.Path), ".known-hosts-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.WriteString(strings.Join(kept, "\n") + "\n")
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), k.Path)
}
