// SPDX-License-Identifier: GPL-3.0-only
package forwarding

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
)

// socksDestination implements the SOCKS5 no-auth TCP CONNECT handshake.
// BIND, UDP ASSOCIATE and authentication are intentionally unsupported.
func socksDestination(conn net.Conn) (string, error) {
	var greeting [2]byte
	if _, err := io.ReadFull(conn, greeting[:]); err != nil {
		return "", err
	}
	if greeting[0] != 5 || greeting[1] == 0 {
		return "", errors.New("invalid SOCKS5 greeting")
	}
	methods := make([]byte, int(greeting[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}
	acceptable := false
	for _, method := range methods {
		if method == 0 {
			acceptable = true
		}
	}
	if !acceptable {
		_, _ = conn.Write([]byte{5, 255})
		return "", errors.New("SOCKS5 requires the no-auth method")
	}
	if _, err := conn.Write([]byte{5, 0}); err != nil {
		return "", err
	}
	var request [4]byte
	if _, err := io.ReadFull(conn, request[:]); err != nil {
		return "", err
	}
	if request[0] != 5 || request[2] != 0 {
		_ = socksReply(conn, 1)
		return "", errors.New("invalid SOCKS5 request")
	}
	if request[1] != 1 {
		_ = socksReply(conn, 7)
		return "", errors.New("only SOCKS5 TCP CONNECT is supported")
	}
	var host string
	switch request[3] {
	case 1, 4:
		size := 4
		if request[3] == 4 {
			size = 16
		}
		address := make([]byte, size)
		if _, err := io.ReadFull(conn, address); err != nil {
			return "", err
		}
		host = net.IP(address).String()
	case 3:
		var size [1]byte
		if _, err := io.ReadFull(conn, size[:]); err != nil {
			return "", err
		}
		address := make([]byte, int(size[0]))
		if _, err := io.ReadFull(conn, address); err != nil {
			return "", err
		}
		host = string(address)
	default:
		_ = socksReply(conn, 8)
		return "", errors.New("unsupported SOCKS5 address type")
	}
	var port [2]byte
	if _, err := io.ReadFull(conn, port[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint16(port[:])
	if host == "" || n == 0 || strings.ContainsAny(host, " \t\r\n\x00/\\") {
		_ = socksReply(conn, 1)
		return "", errors.New("invalid SOCKS5 destination")
	}
	return net.JoinHostPort(host, strconv.Itoa(int(n))), nil
}

func socksReply(conn net.Conn, code byte) error {
	_, err := conn.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0})
	return err
}
