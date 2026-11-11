//go:build !windows

package hcore

import (
	"fmt"

	tun "github.com/sagernet/sing-tun"
	"golang.org/x/sys/unix"
)

// openOhosTun duplicates the injected fd (so the engine's Close() cannot close
// the system VPN handle) and builds a sing-tun device directly on it. With a
// non-zero FileDescriptor, tun.New never opens /dev/net/tun and skips all
// kernel route/address configuration, which was already done by the OHOS
// VpnExtensionAbility's VpnConfig.
func openOhosTun(options *tun.Options, fd int) (tun.Tun, error) {
	dupFd, err := unix.Dup(fd)
	if err != nil {
		return nil, fmt.Errorf("ohos: dup tun fd: %w", err)
	}
	options.FileDescriptor = dupFd
	if options.Name == "" {
		options.Name = "tun0"
	}
	return tun.New(*options)
}
