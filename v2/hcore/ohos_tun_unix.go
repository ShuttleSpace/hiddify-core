//go:build !windows

package hcore

import (
	"fmt"

	tun "github.com/sagernet/sing-tun"
	"golang.org/x/sys/unix"
)

// dupOhosTunFd validates fd and returns a duplicate carrying FD_CLOEXEC, so
// the engine's Close() cannot close the OS-owned VPN handle and no child
// process inherits the vNIC fd. F_DUPFD_CLOEXEC matches the fork's
// detachRuntimePoller convention (plain unix.Dup clears CLOEXEC).
func dupOhosTunFd(fd int) (int, error) {
	if fd <= 0 {
		return 0, fmt.Errorf("ohos: invalid tun fd %d", fd)
	}
	// F_GETFD rejects a stale/closed fd number; otherwise a reused number would
	// silently be dup'd onto an unrelated descriptor.
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); err != nil {
		return 0, fmt.Errorf("ohos: invalid tun fd %d: %w", fd, err)
	}
	dupFd, err := unix.FcntlInt(uintptr(fd), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return 0, fmt.Errorf("ohos: dup tun fd: %w", err)
	}
	return dupFd, nil
}

// openOhosTun duplicates the injected fd and builds a sing-tun device directly
// on it. With a non-zero FileDescriptor, tun.New never opens /dev/net/tun and
// skips all kernel route/address configuration, which was already done by the
// OHOS VpnExtensionAbility's VpnConfig.
func openOhosTun(options *tun.Options, fd int) (tun.Tun, error) {
	dupFd, err := dupOhosTunFd(fd)
	if err != nil {
		return nil, err
	}
	options.FileDescriptor = dupFd
	// inbound.go already set options.Name via CalculateInterfaceName before
	// calling OpenInterface; register it with the (no-op) interface monitor so
	// MyInterfaces() reflects the TUN, as in the Android chain.
	if options.InterfaceMonitor != nil {
		options.InterfaceMonitor.RegisterMyInterface(options.Name)
	}
	tunDevice, err := tun.New(*options)
	if err != nil {
		// tun.New took ownership only on success; close the dup on failure so
		// a failed start does not leak the descriptor.
		unix.Close(dupFd)
		return nil, err
	}
	return tunDevice, nil
}
