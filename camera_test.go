//go:build integration

package uvc

import (
	"log"
	"syscall"
	"testing"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

func TestAutoExposureMode(t *testing.T) {
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
	for _, iface := range info.ControlInterfaces {
		log.Printf("got control interface: %#v", iface)
		if iface.CameraTerminal != nil {
			setControl := &descriptors.AutoExposureModeControl{Mode: descriptors.AutoExposureModeManual}
			err := iface.CameraTerminal.Set(setControl)
			if err != nil {
				t.Fatal(err)
			}

			control := &descriptors.AutoExposureModeControl{}
			if err = iface.CameraTerminal.Get(control); err != nil {
				t.Fatal(err)
			}

			if control.Mode != descriptors.AutoExposureModeManual {
				t.Fatalf("TestAutoExposure: expected ae mode 1 (manual), got %d", control.Mode)
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

	info, err := ctx.DeviceInfo()
	if err != nil {
		t.Fatal(err)
	}

	// get format descriptors
	for _, iface := range info.ControlInterfaces {
		log.Printf("got control interface: %#v", iface)
		if iface.CameraTerminal != nil {

			supported := iface.CameraTerminal.IsControlRequestSupported(&descriptors.FocusAutoControl{})
			if !supported {
				t.Fatal("feature not supported")
			}

			setControl := &descriptors.FocusAutoControl{FocusAuto: true}
			err := iface.CameraTerminal.Set(setControl)
			if err != nil {
				t.Fatal(err)
			}

			control := &descriptors.FocusAutoControl{}
			if err = iface.CameraTerminal.Get(control); err != nil {
				t.Fatal(err)
			}

			if !control.FocusAuto {
				t.Fatalf("TestAutoFocus: expected true, got %t", control.FocusAuto)
			}
		}
	}
}
