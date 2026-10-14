package hcore

import (
	"context"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
)

func (s *CoreService) Restart(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	return Restart(static.BaseContext, in)
}

func Restart(ctx context.Context, in *StartRequest) (coreResponse *CoreInfoResponse, err error) {
	defer config.DeferPanicToError("startmobile", func(recovered_err error) {
		coreResponse, err = errorWrapper(MessageType_UNEXPECTED_ERROR, recovered_err)
	})
	log.Debug("[Service] Restarting")
	// if static.CoreState != CoreStates_STARTED {
	// 	return errorWrapper(MessageType_INSTANCE_NOT_STARTED, fmt.Errorf("instance not started"))
	// }
	// if static.Box == nil {
	// 	return errorWrapper(MessageType_INSTANCE_NOT_FOUND, fmt.Errorf("instance not found"))
	// }

	// Stop can fail after the box has already given up its resources: Box.Close
	// aggregates errors and keeps going, so every component is closed including
	// the inbound that frees the mixed port. Report the cleanup failure but still
	// start the configuration the user asked for. Otherwise a transient
	// networksetup error while releasing the macOS system proxy leaves the app
	// stuck in the mode it was trying to leave.
	if _, err := Stop(); err != nil {
		log.Warn("[Service] stopping before restart reported: ", err)
	}

	if C.IsAndroid && static.HiddifyOptions.EnableTun {
		select {
		case <-ctx.Done():
			return SetCoreStatus(CoreStates_STOPPED, MessageType_INSTANCE_NOT_STARTED, "restart cancelled"), nil
		case <-time.After(time.Second):
		}
	}
	return StartService(ctx, in)
}
