// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"errors"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// timedSigner prevents a stalled local agent from hanging authentication forever.
type timedSigner struct {
	ssh.Signer
	conn net.Conn
}

func (s timedSigner) call(fn func() (*ssh.Signature, error)) (*ssh.Signature, error) {
	type result struct {
		signature *ssh.Signature
		err       error
	}
	done := make(chan result, 1)
	go func() { signature, err := fn(); done <- result{signature, err} }()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case r := <-done:
		return r.signature, r.err
	case <-timer.C:
		_ = s.conn.Close()
		return nil, errors.New("SSH agent signing timed out")
	}
}
func (s timedSigner) Sign(random io.Reader, data []byte) (*ssh.Signature, error) {
	return s.call(func() (*ssh.Signature, error) { return s.Signer.Sign(random, data) })
}
func (s timedSigner) SignWithAlgorithm(random io.Reader, data []byte, algorithm string) (*ssh.Signature, error) {
	return s.call(func() (*ssh.Signature, error) {
		if signer, ok := s.Signer.(ssh.AlgorithmSigner); ok {
			return signer.SignWithAlgorithm(random, data, algorithm)
		}
		if algorithm != "" && algorithm != s.PublicKey().Type() {
			return nil, errors.New("agent key does not support signature algorithm")
		}
		return s.Signer.Sign(random, data)
	})
}
