package uvc

import (
	"log"
	"syscall"
	"testing"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

func TestAutoExposure(t *testing.T) {
	fd, err := syscall.Open("/dev/bus/usb/001/002", syscall.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := NewUVCDevice(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	go ctx.EventLoop()

	info, err := ctx.DeviceInfo()
	if err != nil {
		t.Fatal(err)
	}

	// get format descriptors
	for _, iface := range info.ControlInterfaces {
		log.Printf("got control interface: %#v", iface)
		if iface.CameraTerminal != nil {
			err := iface.CameraTerminal.SetAutoExposureMode(descriptors.AutoExposureModeAuto)
			if err != nil {
				t.Fatal(err)
			}

			aeMode, err := iface.CameraTerminal.GetAutoExposureMode()
			if err != nil {
				t.Fatal(err)
			}

			if aeMode != descriptors.AutoExposureModeAuto {
				t.Fatalf("TestAutoExposure: expected ae mode 1 (auto), got %d", aeMode)
			}
		}
	}
}

func TestAutoFocus(t *testing.T) {
	fd, err := syscall.Open("/dev/bus/usb/001/002", syscall.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := NewUVCDevice(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	go ctx.EventLoop()

	info, err := ctx.DeviceInfo()
	if err != nil {
		t.Fatal(err)
	}

	// get format descriptors
	for _, iface := range info.ControlInterfaces {
		log.Printf("got control interface: %#v", iface)
		if iface.CameraTerminal != nil {
			err := iface.CameraTerminal.SetAutoFocus(true)
			if err != nil {
				t.Fatal(err)
			}

			status, err := iface.CameraTerminal.GetAutoFocus()
			if err != nil {
				t.Fatal(err)
			}

			if !status {
				t.Fatalf("TestAutoFocus: expected true, got %t", status)
			}
		}
	}
}
