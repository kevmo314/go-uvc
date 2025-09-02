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
	"unsafe"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type UACDevice struct {
	usbctx *C.libusb_context
	handle *C.libusb_device_handle
	device *C.libusb_device
	closed *atomic.Bool
}

func NewUACDevice(fd uintptr) (*UACDevice, error) {
	dev := &UACDevice{closed: &atomic.Bool{}}
	if ret := C.libusb_init(&dev.usbctx); ret < 0 {
		return nil, fmt.Errorf("libusb_init_context failed: %d", libusberror(ret))
	}
	if ret := C.libusb_wrap_sys_device(dev.usbctx, C.intptr_t(fd), &dev.handle); ret < 0 {
		return nil, fmt.Errorf("libusb_wrap_sys_device failed: %d", libusberror(ret))
	}
	if dev.device = C.libusb_get_device(dev.handle); dev.device == nil {
		return nil, fmt.Errorf("libusb_get_device failed")
	}
	return dev, nil
}

func (d *UACDevice) Close() error {
	d.closed.Store(true)
	return nil
}

type AudioControlInterface struct {
	Descriptor descriptors.AudioControlInterface
}

type AudioDeviceInfo struct {
	bcdADC              uint16
	deviceHandle        *C.struct_libusb_device_handle
	configDesc          *C.struct_libusb_config_descriptor
	ControlInterfaces   []*AudioControlInterface
	StreamingInterfaces []*transfers.AudioStreamingInterface
}

func (d *UACDevice) DeviceInfo() (*AudioDeviceInfo, error) {
	var configDesc *C.struct_libusb_config_descriptor
	if ret := C.libusb_get_config_descriptor(d.device, 0, &configDesc); ret < 0 {
		return nil, fmt.Errorf("libusb_get_active_config_descriptor failed: %d", libusberror(ret))
	}

	// scan audio control interfaces
	ifaceIdx := -1
	ifaces := unsafe.Slice(configDesc._interface, configDesc.bNumInterfaces)
	for i, iface := range ifaces {
		// UAC uses class 1 (Audio) and subclass 1 (Control)
		if iface.altsetting.bInterfaceClass == 1 && iface.altsetting.bInterfaceSubClass == 1 {
			ifaceIdx = i
			break
		}
	}
	if ifaceIdx == -1 {
		return nil, fmt.Errorf("audio control interface not found")
	}
	info := &AudioDeviceInfo{deviceHandle: d.handle, configDesc: configDesc}

	audioInterface := &ifaces[ifaceIdx]
	acbuf := unsafe.Slice((*byte)(audioInterface.altsetting.extra), audioInterface.altsetting.extra_length)

	// Parse audio control interface descriptors
	for i := 0; i != len(acbuf); i += int(acbuf[i]) {
		block := acbuf[i : i+int(acbuf[i])]
		if len(block) < 3 {
			continue
		}

		// Check for audio class-specific interface descriptors (0x24)
		if block[1] == 0x24 {
			subtype := block[2]
			switch subtype {
			case 0x01: // HEADER
				if len(block) >= 9 {
					info.bcdADC = uint16(block[3]) | (uint16(block[4]) << 8)
				}
			}
		}
	}

	// Find and parse audio streaming interfaces
	for _, iface := range ifaces {
		// UAC uses class 1 (Audio) and subclass 2 (Streaming)
		if iface.altsetting.bInterfaceClass == 1 && iface.altsetting.bInterfaceSubClass == 2 {
			// Check all alternate settings for this interface
			for alt := 0; alt < int(iface.num_altsetting); alt++ {
				altsetting := unsafe.Slice(iface.altsetting, iface.num_altsetting)[alt]

				// Skip settings with no endpoints (zero-bandwidth)
				// Alternate setting 0 is typically zero-bandwidth, but let's check endpoints
				if altsetting.bNumEndpoints == 0 {
					continue
				}

				// Create a temporary interface structure for this alternate setting
				tempIface := C.struct_libusb_interface{
					altsetting:     &altsetting,
					num_altsetting: 1,
				}

				streamingIface := transfers.NewAudioStreamingInterface(
					unsafe.Pointer(d.usbctx),
					unsafe.Pointer(d.handle),
					unsafe.Pointer(&tempIface),
					info.bcdADC,
				)

				// Parse streaming interface descriptors
				asbuf := unsafe.Slice((*byte)(altsetting.extra), altsetting.extra_length)
				for j := 0; j != len(asbuf); j += int(asbuf[j]) {
					block := asbuf[j : j+int(asbuf[j])]
					if len(block) >= 3 && block[1] == 0x24 {
						// Parse audio streaming descriptors
						streamingIface.ParseDescriptor(block)
					}
				}

				// Parse endpoint descriptor if available
				if altsetting.bNumEndpoints > 0 {
					streamingIface.ParseEndpoint(unsafe.Pointer(altsetting.endpoint))
				}

				// Only add interfaces with valid audio data
				if streamingIface.NrChannels > 0 && len(streamingIface.SamplingFreqs) > 0 {
					info.StreamingInterfaces = append(info.StreamingInterfaces, streamingIface)
				}
			}
		}
	}

	return info, nil
}
