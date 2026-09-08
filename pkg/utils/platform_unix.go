//go:build !windows

package utils

import (
	"context"
	"net"

	"golang.org/x/sys/unix"
)

// CRIPath is the in-container path the manager mounts the host CRI socket to.
const CRIPath = "/run/cri/cri.sock"

// The handoff endpoints, rooted where the manager mounts the shared-data volume
// on this platform. They are build-tagged rather than shared because a Windows
// worker cannot find a peer at a POSIX path.
const (
	ScanErasePath            = "/run/eraser.sh/shared-data/scanErase"
	CollectScanPath          = "/run/eraser.sh/shared-data/collectScan"
	EraseCompleteCollectPath = "/run/eraser.sh/shared-data/eraseCompleteCollect"
	EraseCompleteScanPath    = "/run/eraser.sh/shared-data/eraseCompleteScan"
)

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
