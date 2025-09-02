//go:build android

package uvc

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"sync/atomic"
)

func NewUACDevice(fd uintptr) (*UACDevice, error) {
	dev := &UACDevice{closed: &atomic.Bool{}}
	// On Android, skip libusb_init since we already have a valid file descriptor
	// from Android's UsbDeviceConnection. We pass NULL as the context.
	if ret := C.libusb_wrap_sys_device(nil, C.intptr_t(fd), &dev.handle); ret < 0 {
		return nil, fmt.Errorf("libusb_wrap_sys_device failed: %d", libusberror(ret))
	}
	if dev.device = C.libusb_get_device(dev.handle); dev.device == nil {
		return nil, fmt.Errorf("libusb_get_device failed")
	}
	return dev, nil
}

