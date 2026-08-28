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

// The handoff endpoints, rooted where the manager mounts the shared-data volume
// on this platform. These are Unix domain sockets, so the whole path has to fit
// in sun_path -- see maxSocketPath in handoff_windows.go.
//
// The literal path is what a HostProcess container sees on containerd 1.7,
// which is what AKS runs and what these were measured against. Kubernetes
// documents that containerd 1.6 and the fallback hosts instead expose volume
// mounts beneath CONTAINER_SANDBOX_MOUNT_POINT, where none of this exists --
// so the worker handoff requires containerd 1.7 or newer.
const (
	ScanErasePath            = `C:\run\eraser.sh\shared-data\scanErase`
	CollectScanPath          = `C:\run\eraser.sh\shared-data\collectScan`
	EraseCompleteCollectPath = `C:\run\eraser.sh\shared-data\eraseCompleteCollect`
	EraseCompleteScanPath    = `C:\run\eraser.sh\shared-data\eraseCompleteScan`
)

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
