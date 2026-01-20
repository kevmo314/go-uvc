//go:build !android && !windows

package uvc

import (
	"sync/atomic"

	usb "github.com/kevmo314/go-usb"
)

func NewUACDevice(fd uintptr) (*UACDevice, error) {
	dev := &UACDevice{closed: &atomic.Bool{}}

	handle, err := usb.WrapSysDevice(int(fd))
	if err != nil {
		return nil, err
	}
	dev.handle = handle

	return dev, nil
}
