package uvc

import (
	"log"
	"syscall"
	"testing"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

func TestDeviceInfoGoUSB(t *testing.T) {
	fd, err := syscall.Open("/dev/bus/usb/001/002", syscall.O_RDWR, 0)
	if err != nil {

		t.Fatal(err)
	}

	uvcDevice, err := NewUVCDevice1(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	info, err := uvcDevice.GetDeviceInfo()
	if err != nil {
		t.Fatal(err)
	}

	for _, iface := range info.StreamingInterfaces {
		for _, desc := range iface.Descriptors {
			if _, ok := desc.(*descriptors.MJPEGFrameDescriptor); !ok {
				continue
			}

			resp, err := iface.ClaimDeviceFrameReader(uvcDevice, info.bcdUVC, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			log.Printf("got negotiated format: %#v", resp)

			// fr, err := resp.ReadFrame()
			// if err != nil {
			// 	t.Fatal(err)
			// }
			// log.Printf("got frame: %#v", fr)

			break
		}
	}
	t.Fail()

}

func TestDeviceInfo(t *testing.T) {
	fd, err := syscall.Open("/dev/bus/usb/001/002", syscall.O_RDWR, 0)
	if err != nil {

		t.Fatal(err)
	}

	ctx, err := NewUVCDevice(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	info, err := ctx.DeviceInfo()
	if err != nil {
		t.Fatal(err)
	}

	// get format descriptors
	for _, iface := range info.StreamingInterfaces {
		for _, desc := range iface.Descriptors {
			if _, ok := desc.(*descriptors.MJPEGFrameDescriptor); !ok {
				continue
			}

			resp, err := iface.ClaimFrameReader(0, 0)
			if err != nil {
				t.Fatal(err)
			}
			log.Printf("got negotiated format: %#v", resp)

			fr, err := resp.ReadFrame()
			if err != nil {
				t.Fatal(err)
			}
			log.Printf("got frame: %#v", fr)

			break
		}
	}
	t.Fail()
}
