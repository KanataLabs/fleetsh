// SPDX-License-Identifier: GPL-3.0-only
package transport

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

type bufferedProxyConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedProxyConn) Read(p []byte) (int, error) { return c.reader.Read(p) }

type proxyHeaderReader struct {
	r         io.Reader
	remaining int
}

func (r *proxyHeaderReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, errors.New("proxy response headers exceed 64 KiB")
	}
	if r.remaining > 0 && len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.r.Read(p)
	if r.remaining > 0 {
		r.remaining -= n
	}
	return n, err
}

func dialProxy(ctx context.Context, value, target string, auth *proxy.Auth) (net.Conn, error) {
	u, err := url.Parse(value)
	if err != nil {
		return nil, errors.New("invalid proxy URL")
	}
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "socks5":
			port = "1080"
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	address := net.JoinHostPort(u.Hostname(), port)
	if u.Scheme == "socks5" {
		d, err := proxy.SOCKS5("tcp", address, auth, &net.Dialer{})
		if err != nil {
			return nil, errors.New("proxy initialization failed")
		}
		cd, ok := d.(proxy.ContextDialer)
		if !ok {
			return nil, errors.New("proxy does not support cancellation")
		}
		return cd.DialContext(ctx, "tcp", target)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("unsupported proxy scheme")
	}
	var conn net.Conn
	if u.Scheme == "https" {
		conn, err = (&tls.Dialer{NetDialer: &net.Dialer{}, Config: &tls.Config{MinVersion: tls.VersionTLS12}}).DialContext(ctx, "tcp", address)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	request := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: target}, Host: target, Header: make(http.Header)}
	if auth != nil {
		request.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth.User+":"+auth.Password)))
	}
	if err := request.Write(conn); err != nil {
		return nil, err
	}
	limited := &proxyHeaderReader{r: conn, remaining: 64 << 10}
	reader := bufio.NewReader(limited)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		return nil, errors.New("invalid HTTP CONNECT response")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP CONNECT rejected (status %d)", response.StatusCode)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	limited.remaining = -1
	success = true
	return &bufferedProxyConn{Conn: conn, reader: reader}, nil
}
