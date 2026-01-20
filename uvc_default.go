//go:build !android

package uvc

import (
	"sync/atomic"

	usb "github.com/kevmo314/go-usb"
)

func NewUVCDevice(fd uintptr) (*UVCDevice, error) {
	dev := &UVCDevice{closed: &atomic.Bool{}}

	handle, err := usb.WrapSysDevice(int(fd))
	if err != nil {
		return nil, err
	}
	dev.handle = handle

	return dev, nil
}
