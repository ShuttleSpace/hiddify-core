package hcore

import (
	"context"

	"github.com/hiddify/hiddify-core/v2/hcommon"
	"github.com/sagernet/sing-box/common/settings"
)

// SetSystemProxyEnabled hands the OS back without stopping the tunnel.
//
// Only the disabling direction is offered. Enabling is the inbound's job: it is the
// thing that owns the mixed port the proxy has to point at, so there is nothing to
// turn on until the box is (re)started with set_system_proxy, which the service
// mode change already arranges for the next connection.
func (s *CoreService) SetSystemProxyEnabled(ctx context.Context, in *SetSystemProxyEnabledRequest) (*hcommon.Response, error) {
	if in.IsEnabled {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: "the system proxy is enabled by the inbound that owns it; switch the service mode instead",
		}, nil
	}
	if err := settings.ReleaseSystemProxy(); err != nil {
		return &hcommon.Response{
			Code:    hcommon.ResponseCode_FAILED,
			Message: err.Error(),
		}, err
	}
	return &hcommon.Response{
		Code:    hcommon.ResponseCode_OK,
		Message: "",
	}, nil
}
