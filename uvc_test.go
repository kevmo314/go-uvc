package uvc

import (
	"bytes"
	"image/jpeg"
	"log"
	"os"
	"syscall"
	"testing"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

func TestDeviceInfo(t *testing.T) {
	fd, err := syscall.Open("/dev/bus/usb/001/006", syscall.O_RDWR, 0)
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
	}

	for _, iface := range info.StreamingInterfaces {
		for i, desc := range iface.Descriptors {
			log.Printf("got format descriptor: %T", desc)
			fd, ok := desc.(*descriptors.MJPEGFormatDescriptor)
			if !ok {
				continue
			}
			frd := iface.Descriptors[i+1].(*descriptors.MJPEGFrameDescriptor)

			resp, err := iface.ClaimFrameReader(fd.Index(), frd.Index())
			if err != nil {
				t.Fatal(err)
			}

			for i := 0; ; i++ {
				fr, err := resp.ReadFrame()
				if err != nil {
					t.Fatal(err)
				}
				// write fr to a file
				img, err := jpeg.Decode(bytes.NewReader(fr.Data))
				if err != nil {
					log.Printf("short on %d", i)
					t.Fatal(err)
				} else {
					log.Printf("got frame: %#v", img.Bounds())
				}
			}

			break
		}
	}
	t.Fail()
}

func TestJPEGDecode(t *testing.T) {
	// open frame-1.jpg and decode it
	f, err := os.Open("frame-0.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("got frame: %#v", img.Bounds())
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
