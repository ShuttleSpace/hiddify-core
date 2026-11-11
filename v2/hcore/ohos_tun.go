package hcore

import (
	"fmt"
	"sync/atomic"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	tun "github.com/sagernet/sing-tun"
)

// ohosTunFd holds the virtual-NIC file descriptor handed over by the
// OpenHarmony VpnExtensionAbility through the SetupTun C export. It is zero
// until OHOS explicitly sets it, which keeps every other platform on the
// original code path (see the guard in start.go).
var ohosTunFd atomic.Int32

// SetOhosTunFd stores the injected OHOS tun fd. Call before start().
func SetOhosTunFd(fd int32) {
	ohosTunFd.Store(fd)
}

// GetOhosTunFd returns the injected OHOS tun fd, or 0 when unset.
func GetOhosTunFd() int32 {
	return ohosTunFd.Load()
}

// OhosPlatformInterface is the libbox.PlatformInterface used on OHOS. It only
// needs to return the already-opened vNIC fd from OpenTun; everything else is
// inherited from MobilePlatformInterface, whose methods are nil-safe no-ops.
type OhosPlatformInterface struct {
	MobilePlatformInterface
}

var _ libbox.PlatformInterface = (*OhosPlatformInterface)(nil)

func (p *OhosPlatformInterface) OpenTun(_ libbox.TunOptions) (int32, error) {
	fd := GetOhosTunFd()
	if fd <= 0 {
		return 0, fmt.Errorf("ohos: tun fd not set (call SetupTun before start)")
	}
	return fd, nil
}

// ohosTunPlatformInterface is the adapter.PlatformInterface registered on
// OHOS. It embeds the libbox wrapper (for the many platform callbacks) but
// overrides OpenInterface to inject the fd directly, avoiding the wrapper's
// getTunnelName(TUNGETIFF) call that an OHOS vNIC fd may not support. All
// UsePlatform* switches are forced off so the engine never invokes a platform
// callback that has no OHOS implementation; the only one kept on is
// UsePlatformInterface, which routes tun creation through OpenInterface.
type ohosTunPlatformInterface struct {
	adapter.PlatformInterface
}

var _ adapter.PlatformInterface = (*ohosTunPlatformInterface)(nil)

func newOhosTunPlatformInterface() adapter.PlatformInterface {
	return &ohosTunPlatformInterface{
		PlatformInterface: libbox.WrapPlatformInterface(&OhosPlatformInterface{}),
	}
}

func (p *ohosTunPlatformInterface) OpenInterface(options *tun.Options, platformOptions option.TunPlatformOptions) (tun.Tun, error) {
	fd := GetOhosTunFd()
	if fd <= 0 {
		return nil, fmt.Errorf("ohos: tun fd not set (call SetupTun before start)")
	}
	return openOhosTun(options, int(fd))
}

func (p *ohosTunPlatformInterface) UsePlatformInterface() bool { return true }

func (p *ohosTunPlatformInterface) UsePlatformAutoDetectInterfaceControl() bool { return false }
func (p *ohosTunPlatformInterface) UsePlatformDefaultInterfaceMonitor() bool    { return false }
func (p *ohosTunPlatformInterface) UsePlatformNetworkInterfaces() bool          { return false }
func (p *ohosTunPlatformInterface) UsePlatformWIFIMonitor() bool                { return false }
func (p *ohosTunPlatformInterface) UsePlatformConnectionOwnerFinder() bool      { return false }
func (p *ohosTunPlatformInterface) UsePlatformNotification() bool               { return false }
func (p *ohosTunPlatformInterface) UsePlatformNeighborResolver() bool           { return false }
func (p *ohosTunPlatformInterface) UsePlatformShell() bool                      { return false }
func (p *ohosTunPlatformInterface) UsePlatformBridge() bool                     { return false }
func (p *ohosTunPlatformInterface) UsePlatformAutoRedirect() bool               { return false }
func (p *ohosTunPlatformInterface) UnderNetworkExtension() bool                 { return false }
func (p *ohosTunPlatformInterface) NetworkExtensionIncludeAllNetworks() bool    { return false }
