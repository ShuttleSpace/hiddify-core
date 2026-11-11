package hcore

import (
	"fmt"
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
	tun "github.com/sagernet/sing-tun"
	"github.com/sagernet/sing/common/control"
	"github.com/sagernet/sing/common/logger"
	"github.com/sagernet/sing/common/x/list"
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

// OhosPlatformInterface is the libbox.PlatformInterface used as the delegate
// for the many platform callbacks of the libbox wrapper. On OHOS the fd is
// injected by ohosTunPlatformInterface.OpenInterface, so the wrapper's own
// OpenInterface (and its OpenTun/getTunnelName path) is never invoked; this
// type therefore only supplies nil-safe no-ops/errors for everything else.
type OhosPlatformInterface struct {
	MobilePlatformInterface
}

var _ libbox.PlatformInterface = (*OhosPlatformInterface)(nil)

// ohosTunPlatformInterface is the adapter.PlatformInterface registered on
// OHOS. It embeds the libbox wrapper (for the platform callbacks) but overrides
// OpenInterface to inject the fd directly, avoiding the wrapper's
// getTunnelName(TUNGETIFF) call that an OHOS vNIC fd may not support.
//
// The default-interface monitor is a no-op (Start/Close return nil): returning
// false would make the engine fall back to the Linux netlink monitor, which
// OHOS sandboxes may forbid, while the libbox wrapper's monitor would call
// back into a platform (MobilePlatformInterface) that has no OHOS
// implementation and fail with "platform is nil".
type ohosTunPlatformInterface struct {
	adapter.PlatformInterface

	mu            sync.Mutex
	myTunAddress  []netip.Addr
	noopMonitor   *ohosInterfaceMonitor
}

var _ adapter.PlatformInterface = (*ohosTunPlatformInterface)(nil)

func newOhosTunPlatformInterface() adapter.PlatformInterface {
	return &ohosTunPlatformInterface{
		PlatformInterface: libbox.WrapPlatformInterface(&OhosPlatformInterface{}),
		noopMonitor:       &ohosInterfaceMonitor{},
	}
}

func (p *ohosTunPlatformInterface) OpenInterface(options *tun.Options, platformOptions option.TunPlatformOptions) (tun.Tun, error) {
	fd := GetOhosTunFd()
	if fd <= 0 {
		return nil, fmt.Errorf("ohos: tun fd not set (call SetupTun before start)")
	}
	// Keep the tun addresses so MyInterfaceAddress (used by direct/dns
	// outbounds to recognise our own interface) matches the Android chain.
	p.setMyTunAddress(myTunAddress(options))
	return openOhosTun(options, int(fd))
}

func (p *ohosTunPlatformInterface) setMyTunAddress(addresses []netip.Addr) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.myTunAddress = addresses
}

func (p *ohosTunPlatformInterface) MyInterfaceAddress() []netip.Addr {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.myTunAddress
}

func (p *ohosTunPlatformInterface) UsePlatformInterface() bool { return true }

// UsePlatformDefaultInterfaceMonitor is true, and CreateDefaultInterfaceMonitor
// returns a no-op monitor. Together they keep the engine off the Linux netlink
// path while avoiding the libbox wrapper's "platform is nil" monitor.
func (p *ohosTunPlatformInterface) UsePlatformDefaultInterfaceMonitor() bool { return true }

func (p *ohosTunPlatformInterface) CreateDefaultInterfaceMonitor(_ logger.Logger) tun.DefaultInterfaceMonitor {
	return p.noopMonitor
}

// The following platform capabilities have no OHOS implementation. Leaving
// them on would route the engine into Linux-specific code paths (netlink,
// SO_BINDTODEVICE, nftables/iptables, ...) or into a MobilePlatformInterface
// stub that returns "platform is nil" / os.ErrInvalid. Keep them off for the
// first cut; see docs/superpowers plans for the OHOS protect(fd) follow-up.
func (p *ohosTunPlatformInterface) UsePlatformAutoDetectInterfaceControl() bool { return false }
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

// ohosInterfaceMonitor is a tun.DefaultInterfaceMonitor that does nothing but
// record the TUN interface name, so MyInterfaces()/MyInterfaceAddress() keep
// working without a real network monitor. Start/Close succeed (no netlink).
type ohosInterfaceMonitor struct {
	mu           sync.Mutex
	myInterfaces []string
	callbacks    list.List[tun.DefaultInterfaceUpdateCallback]
}

var _ tun.DefaultInterfaceMonitor = (*ohosInterfaceMonitor)(nil)

func (m *ohosInterfaceMonitor) Start() error { return nil }
func (m *ohosInterfaceMonitor) Close() error { return nil }

func (m *ohosInterfaceMonitor) DefaultInterface() *control.Interface { return nil }
func (m *ohosInterfaceMonitor) OverrideAndroidVPN() bool            { return false }
func (m *ohosInterfaceMonitor) AndroidVPNEnabled() bool             { return false }

func (m *ohosInterfaceMonitor) RegisterCallback(callback tun.DefaultInterfaceUpdateCallback) *list.Element[tun.DefaultInterfaceUpdateCallback] {
	return m.callbacks.PushBack(callback)
}

func (m *ohosInterfaceMonitor) UnregisterCallback(element *list.Element[tun.DefaultInterfaceUpdateCallback]) {
	if element != nil {
		m.callbacks.Remove(element)
	}
}

func (m *ohosInterfaceMonitor) RegisterMyInterface(interfaceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.myInterfaces = append(m.myInterfaces, interfaceName)
}

func (m *ohosInterfaceMonitor) MyInterfaces() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.myInterfaces
}

func myTunAddress(options *tun.Options) []netip.Addr {
	addresses := make([]netip.Addr, 0, len(options.Inet4Address)+len(options.Inet6Address))
	for _, prefix := range options.Inet4Address {
		addresses = append(addresses, prefix.Addr())
	}
	for _, prefix := range options.Inet6Address {
		addresses = append(addresses, prefix.Addr())
	}
	return addresses
}
