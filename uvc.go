package uvc

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type UVCDevice struct {
	usbctx *C.libusb_context
	handle *C.libusb_device_handle
	device *C.libusb_device
}

func (d *UVCDevice) IsTISCamera() (bool, error) {
	var desc C.struct_libusb_device_descriptor
	if ret := C.libusb_get_device_descriptor(d.device, &desc); ret < 0 {
		return false, fmt.Errorf("libusb_get_device_descriptor failed: %d", libusberror(ret))
	}
	return desc.idVendor == 0x199e && (desc.idProduct == 0x8101 || desc.idProduct == 0x8102), nil
}

type ControlInterface struct {
	CameraTerminal *CameraTerminal
	ProcessingUnit *ProcessingUnit
	Descriptor     descriptors.ControlInterface
}

type DeviceInfo struct {
	bcdUVC              uint16 // cached since it's used a lot
	deviceHandle        *C.struct_libusb_device_handle
	configDesc          *C.struct_libusb_config_descriptor
	ControlInterfaces   []*ControlInterface
	StreamingInterfaces []*transfers.StreamingInterface
}

func (d *UVCDevice) DeviceInfo() (*DeviceInfo, error) {
	var configDesc *C.struct_libusb_config_descriptor
	if ret := C.libusb_get_config_descriptor(d.device, 0, &configDesc); ret < 0 {
		return nil, fmt.Errorf("libusb_get_active_config_descriptor failed: %d", libusberror(ret))
	}

	// scan control interfaces
	isTISCamera, err := d.IsTISCamera()
	if err != nil {
		return nil, err
	}
	ifaceIdx := -1
	ifaces := unsafe.Slice(configDesc._interface, configDesc.bNumInterfaces)
	for i, iface := range ifaces {
		if isTISCamera && iface.altsetting.bInterfaceClass == 255 && iface.altsetting.bInterfaceSubClass == 1 {
			ifaceIdx = i
			break
		} else if !isTISCamera && iface.altsetting.bInterfaceClass == 14 && iface.altsetting.bInterfaceSubClass == 1 {
			ifaceIdx = i
			break
		}
	}
	if ifaceIdx == -1 {
		return nil, fmt.Errorf("control interface not found")
	}
	info := &DeviceInfo{deviceHandle: d.handle, configDesc: configDesc}

	videoInterface := &ifaces[ifaceIdx]

	vcbuf := unsafe.Slice((*byte)(videoInterface.altsetting.extra), videoInterface.altsetting.extra_length)

	for i := 0; i != len(vcbuf); i += int(vcbuf[i]) {
		block := vcbuf[i : i+int(vcbuf[i])]
		if block[1] != 0x24 {
			// ignore blocks that are not CS_INTERFACE 0x24
			continue
		}
		ci, err := descriptors.UnmarshalControlInterface(block)
		if err != nil {
			return nil, err
		}
		switch ci := ci.(type) {
		case *descriptors.ProcessingUnitDescriptor:
			processingUnit := &ProcessingUnit{
				usb:            &ifaces[ifaceIdx],
				deviceHandle:   d.handle,
				UnitDescriptor: ci,
			}
			info.ControlInterfaces = append(info.ControlInterfaces, &ControlInterface{ProcessingUnit: processingUnit, Descriptor: ci})
		case *descriptors.InputTerminalDescriptor:
			it, err := descriptors.UnmarshalInputTerminal(block)
			if err != nil {
				return nil, err
			}

			switch descriptor := it.(type) {
			case *descriptors.CameraTerminalDescriptor:
				camera := &CameraTerminal{
					usb:              &ifaces[ifaceIdx],
					deviceHandle:     d.handle,
					CameraDescriptor: descriptor,
				}
				info.ControlInterfaces = append(info.ControlInterfaces, &ControlInterface{CameraTerminal: camera, Descriptor: descriptor})
			}
		case *descriptors.HeaderDescriptor:
			info.bcdUVC = ci.UVC
			// pull the streaming interfaces too
			for _, i := range ci.VideoStreamingInterfaceIndexes {
				vsbuf := unsafe.Slice((*byte)(ifaces[i].altsetting.extra), ifaces[i].altsetting.extra_length)
				asi := transfers.NewStreamingInterface(unsafe.Pointer(d.usbctx), unsafe.Pointer(d.handle), unsafe.Pointer(&ifaces[i]), ci.UVC)
				for j := 0; j != len(vsbuf); j += int(vsbuf[j]) {
					block := vsbuf[j : j+int(vsbuf[j])]
					si, err := descriptors.UnmarshalStreamingInterface(block)
					if err != nil {
						return nil, err
					}
					asi.Descriptors = append(asi.Descriptors, si)
				}
				info.StreamingInterfaces = append(info.StreamingInterfaces, asi)
			}
		default:
			// This is an interface that we have not yet parsed
			info.ControlInterfaces = append(info.ControlInterfaces, &ControlInterface{Descriptor: ci})
		}
	}

	return info, nil
}

func (d *DeviceInfo) Close() error {
	C.libusb_free_config_descriptor(d.configDesc)
	return nil
}
