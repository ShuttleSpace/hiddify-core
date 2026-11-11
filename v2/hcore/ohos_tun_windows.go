//go:build windows

package hcore

import (
	"fmt"

	tun "github.com/sagernet/sing-tun"
)

// OHOS builds are GOOS=linux; the Windows build must still compile the shared
// hcore package, but SetupTun is never called there.
func openOhosTun(_ *tun.Options, _ int) (tun.Tun, error) {
	return nil, fmt.Errorf("ohos: tun fd injection is not supported on windows")
}
