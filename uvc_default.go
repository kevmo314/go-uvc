//go:build !android

package uvc

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
)

func NewUVCDevice(fd uintptr) (*UVCDevice, error) {
	dev := &UVCDevice{}
	if ret := C.libusb_init(&dev.usbctx); ret < 0 {
		return nil, fmt.Errorf("libusb_init_context failed: %d", libusberror(ret))
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
