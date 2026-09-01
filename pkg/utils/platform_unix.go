//go:build !windows

package utils

import (
	"context"
	"net"

	"golang.org/x/sys/unix"
)

// CRIPath is the in-container path the manager mounts the host CRI socket to.
const CRIPath = "/run/cri/cri.sock"

// unixProtocol is the network protocol of a unix socket.
const unixProtocol = "unix"

const defaultProtocol = unixProtocol

func criDialer(protocol string) (func(ctx context.Context, addr string) (net.Conn, error), error) {
	if protocol != unixProtocol {
		return nil, ErrOnlySupportUnixSocket
	}

	return func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, unixProtocol, addr)
	}, nil
}

func mkfifo(path string, mode uint32) error {
	return unix.Mkfifo(path, mode)
}
