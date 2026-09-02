//go:build windows

package utils

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

// defaultCRIPath is containerd's named pipe. Unlike Linux there is no socket to
// mount; a HostProcess pod reaches the pipe directly.
const defaultCRIPath = `npipe://./pipe/containerd-containerd`

// CRIPath is the CRI endpoint a Windows worker connects to. The manager
// propagates a configurable address via EnvEraserRuntimeAddress (a Windows
// named pipe cannot be hostPath-mounted like a Linux socket); when the env var
// is unset the default containerd pipe is used.
var CRIPath = resolveCRIPath()

func resolveCRIPath() string {
	if addr := os.Getenv(EnvEraserRuntimeAddress); addr != "" {
		return addr
	}
	return defaultCRIPath
}

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
