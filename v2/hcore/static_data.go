package hcore

import (
	"context"
	"sync"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/common/monitoring"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
)

type HiddifyInstance struct {
	StartedService *daemon.StartedService
	HiddifyOptions *config.HiddifyOptions
	// activeConfigPath string
	CoreLogFactory              log.Factory
	coreInfoObserver            *monitoring.Broadcaster[*CoreInfoResponse]
	CoreState                   CoreStates
	logObserver                 *monitoring.Broadcaster[*LogMessage]
	systemInfoObserver          *monitoring.Broadcaster[*SystemInfo]
	outboundsInfoObserver       *monitoring.Broadcaster[*OutboundGroupList]
	mainOutboundsInfoObserver   *monitoring.Broadcaster[*OutboundGroupList]
	lock                        sync.Mutex
	globalPlatformInterface     libbox.PlatformInterface
	globalPlatformInterfaceMode SetupMode
	previousStartRequest        *StartRequest
	debug                       bool
	ListenPort                  uint16
	BaseContext                 context.Context
	endPauseTimer               *time.Timer // only for ios

	logLevel LogLevel
}

var static = &HiddifyInstance{
	CoreState:                 CoreStates_STOPPED,
	coreInfoObserver:          monitoring.NewBroadcaster[*CoreInfoResponse](context.Background()),
	logObserver:               monitoring.NewBroadcaster[*LogMessage](context.Background()),
	systemInfoObserver:        monitoring.NewBroadcaster[*SystemInfo](context.Background()),
	outboundsInfoObserver:     monitoring.NewBroadcaster[*OutboundGroupList](context.Background()),
	mainOutboundsInfoObserver: monitoring.NewBroadcaster[*OutboundGroupList](context.Background()),
}

// retainedPlatformInterfaces keeps gomobile foreign proxies reachable from Go.
// A finalizer on these proxies decrements the corresponding Java Seq ref, which
// causes later callbacks to abort with "Unknown reference".
var retainedPlatformInterfaces []libbox.PlatformInterface

func isBackgroundPlatformMode(mode SetupMode) bool {
	return mode == SetupMode_GRPC_BACKGROUND || mode == SetupMode_GRPC_BACKGROUND_INSECURE
}

func updatePlatformInterfaceLocked(mode SetupMode, platformInterface libbox.PlatformInterface) {
	static.BaseContext = libbox.BaseContext(platformInterface)
	if platformInterface != nil {
		retainedPlatformInterfaces = append(retainedPlatformInterfaces, platformInterface)
	}
	if platformInterface != nil || !isBackgroundPlatformMode(static.globalPlatformInterfaceMode) {
		static.globalPlatformInterface = platformInterface
		static.globalPlatformInterfaceMode = mode
	}
}

func currentPlatformInterface() libbox.PlatformInterface {
	mu.Lock()
	defer mu.Unlock()
	return static.globalPlatformInterface
}

func currentBaseContext() context.Context {
	mu.Lock()
	defer mu.Unlock()
	return static.BaseContext
}
