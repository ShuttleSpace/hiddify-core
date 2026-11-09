package hcore

import (
	"context"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
)

// Restart queues a restart of the core and answers at once with STOPPING; the outcome
// (STOPPING, STOPPED, STARTING, STARTED) is reported through CoreInfoListener.
func (s *CoreService) Restart(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	queue.Lock()
	defer queue.Unlock()
	queueRestartLocked(static.BaseContext, in)
	return &CoreInfoResponse{CoreState: CoreStates_STOPPING, MessageType: MessageType_EMPTY}, nil
}

// Restart stops then starts the core after the operations queued before it, and returns the
// outcome.
func Restart(ctx context.Context, in *StartRequest) (*CoreInfoResponse, error) {
	queue.Lock()
	result := queueRestartLocked(ctx, in)
	queue.Unlock()
	return wait(result)
}

// queueRestartLocked queues a restart: it cancels the starts queued before it, and a stop
// called after it cancels its start; queue must be locked.
func queueRestartLocked(ctx context.Context, in *StartRequest) <-chan operationResult {
	signalStopLocked()
	ctx = withStopSignalLocked(ctx)
	return enqueueLocked(operationStart, "restart", func() (*CoreInfoResponse, error) {
		log.Debug("[Service] Restarting")
		// Stop can fail after the box has already given up its resources: Box.Close
		// aggregates errors and keeps going, so every component is closed including
		// the inbound that frees the mixed port. Report the cleanup failure but still
		// start the configuration the user asked for. Otherwise a transient
		// networksetup error while releasing the macOS system proxy leaves the app
		// stuck in the mode it was trying to leave.
		if _, err := stopLocked(); err != nil {
			log.Warn("[Service] stopping before restart reported: ", err)
		}
		if C.IsAndroid && static.HiddifyOptions.EnableTun {
			sleepContext(ctx, time.Second)
		}
		if err := contextErr(ctx); err != nil {
			return cancelledStart(err)
		}
		return startServiceLocked(ctx, in)
	})
}
