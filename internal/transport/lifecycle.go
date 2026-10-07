// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"context"
	"net"
)

func (m *Manager) register(conn net.Conn) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		_ = conn.Close()
		return context.Canceled
	}
	m.closers = append(m.closers, conn)
	return nil
}
func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	conns := append([]net.Conn(nil), m.closers...)
	stop := m.stopClose
	m.mu.Unlock()
	if stop != nil {
		stop()
	}
	for _, conn := range conns {
		_ = conn.Close()
	}
}
