package main

/*
#include <stdint.h>
*/
import "C"

import (
	hcore "github.com/hiddify/hiddify-core/v2/hcore"
)

//export SetupTun
func SetupTun(fd C.int) {
	// The OHOS VpnExtensionAbility opens the virtual NIC and hands its file
	// descriptor to Dart, which calls this before start(). Storing it here
	// lets hcore.Start register the fd-injecting platform interface. A value
	// <= 0 clears the fd so a later start falls back to the normal path.
	hcore.SetOhosTunFd(int32(fd))
}
