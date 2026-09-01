//go:build windows

package utils

import (
	"context"
	"errors"
	"net"

	"github.com/Microsoft/go-winio"
)

// CRIPath is containerd's named pipe. Unlike Linux there is no socket to mount;
// a HostProcess pod reaches the pipe directly.
const CRIPath = `npipe://./pipe/containerd-containerd`

const defaultProtocol = npipeProtocol

var ErrFifoUnsupported = errors.New("named FIFOs are not supported on windows")

func criDialer(protocol string) (func(ctx context.Context, addr string) (net.Conn, error), error) {
	if protocol != npipeProtocol {
		return nil, ErrProtocolNotSupported
	}

	return winio.DialPipeContext, nil
}

func mkfifo(path string, mode uint32) error {
	return ErrFifoUnsupported
}
