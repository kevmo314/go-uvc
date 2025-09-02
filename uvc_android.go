//go:build android

package uvc

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>

#define LIBUSB_OPTION_NO_DEVICE_DISCOVERY 2

static inline int libusb_set_option_no_device_discovery(libusb_context *ctx) {
    return libusb_set_option(ctx, LIBUSB_OPTION_NO_DEVICE_DISCOVERY, NULL);
}
*/
import "C"
import (
	"fmt"
	"sync/atomic"
)

func NewUVCDevice(fd uintptr) (*UVCDevice, error) {
	dev := &UVCDevice{closed: &atomic.Bool{}}
	// On Android, we need to set NO_DEVICE_DISCOVERY option BEFORE libusb_init
	// since we already have a valid file descriptor from Android's UsbDeviceConnection
	if ret := C.libusb_set_option_no_device_discovery(nil); ret < 0 {
		return nil, fmt.Errorf("libusb_set_option failed: %d", libusberror(ret))
	}
	if ret := C.libusb_init(&dev.usbctx); ret < 0 {
		return nil, fmt.Errorf("libusb_init failed: %d", libusberror(ret))
	}
	if ret := C.libusb_wrap_sys_device(dev.usbctx, C.intptr_t(fd), &dev.handle); ret < 0 {
		return nil, fmt.Errorf("libusb_wrap_sys_device failed: %d", libusberror(ret))
	}
	if dev.device = C.libusb_get_device(dev.handle); dev.device == nil {
		return nil, fmt.Errorf("libusb_get_device failed")
	}
	// TODO: libuvc appears to check if the interrupt endpoint is readable, is that necessary?

	return dev, nil
}
