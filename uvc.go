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
	"github.com/kevmo314/go-uvc/pkg/requests"
)

type UVCDevice struct {
	usbctx *C.libusb_context
	handle *C.libusb_device_handle
	device *C.libusb_device
	closed *atomic.Bool
}

func NewUVCDevice(fd uintptr) (*UVCDevice, error) {
	dev := &UVCDevice{closed: &atomic.Bool{}}
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

func (d *UVCDevice) EventLoop() error {
	for !d.closed.Load() {
		if ret := C.libusb_handle_events(d.usbctx); ret < 0 {
			return fmt.Errorf("libusb_handle_events failed: %d", libusberror(ret))
		}
	}
	return nil
}

func (d *UVCDevice) IsTISCamera() (bool, error) {
	var desc C.struct_libusb_device_descriptor
	if ret := C.libusb_get_device_descriptor(d.device, &desc); ret < 0 {
		return false, fmt.Errorf("libusb_get_device_descriptor failed: %d", libusberror(ret))
	}
	return desc.idVendor == 0x199e && (desc.idProduct == 0x8101 || desc.idProduct == 0x8102), nil
}

func (d *UVCDevice) Close() error {
	d.closed.Store(true)
	return nil
}

type ControlInterface struct {
	CameraTerminal *CameraTerminal
	Descriptor     descriptors.ControlInterface
}

type StreamingInterface struct {
	usbctx       *C.libusb_context
	bcdUVC       uint16 // cached since it's used a lot
	deviceHandle *C.struct_libusb_device_handle
	usb          *C.struct_libusb_interface
	Descriptors  []descriptors.StreamingInterface
}

func (si *StreamingInterface) InterfaceNumber() uint8 {
	return uint8(si.usb.altsetting.bInterfaceNumber)
}

func (si *StreamingInterface) UVCVersionString() string {
	return fmt.Sprintf("%x.%02x", si.bcdUVC>>8, si.bcdUVC&0xff)
}

func (si *StreamingInterface) FormatDescriptors() []descriptors.FormatDescriptor {
	var descs []descriptors.FormatDescriptor
	for _, desc := range si.Descriptors {
		if d, ok := desc.(descriptors.FormatDescriptor); ok {
			descs = append(descs, d)
		}
	}
	return descs
}

func (si *StreamingInterface) FrameDescriptors() []descriptors.FrameDescriptor {
	var descs []descriptors.FrameDescriptor
	for _, desc := range si.Descriptors {
		if d, ok := desc.(descriptors.FrameDescriptor); ok {
			descs = append(descs, d)
		}
	}
	return descs
}

func (si *StreamingInterface) InputHeaderDescriptors() []*descriptors.InputHeaderDescriptor {
	var descs []*descriptors.InputHeaderDescriptor
	for _, desc := range si.Descriptors {
		if d, ok := desc.(*descriptors.InputHeaderDescriptor); ok {
			descs = append(descs, d)
		}
	}
	return descs
}

type DeviceInfo struct {
	bcdUVC              uint16 // cached since it's used a lot
	deviceHandle        *C.struct_libusb_device_handle
	configDesc          *C.struct_libusb_config_descriptor
	videoInterface      *C.struct_libusb_interface // cached since it's used a lot
	ControlInterfaces   []*ControlInterface
	StreamingInterfaces []*StreamingInterface
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
	ifaces := (*[1 << 30]C.struct_libusb_interface)(unsafe.Pointer(configDesc._interface))[:configDesc.bNumInterfaces]
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

	info.videoInterface = &ifaces[ifaceIdx]

	vcbuf := (*[1 << 30]byte)(unsafe.Pointer(info.videoInterface.altsetting.extra))[:info.videoInterface.altsetting.extra_length]

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
		case *descriptors.InputTerminalDescriptor:
			it, err := descriptors.UnmarshalInputTerminal(block)
			if err != nil {
				return nil, err
			}

			switch camDesc := it.(type) {
			case *descriptors.CameraTerminalDescriptor:
				camera := &CameraTerminal{
					usb:              &ifaces[ifaceIdx],
					deviceHandle:     d.handle,
					CameraDescriptor: camDesc,
				}
				info.ControlInterfaces = append(info.ControlInterfaces, &ControlInterface{CameraTerminal: camera, Descriptor: camDesc})
			}
		case *descriptors.HeaderDescriptor:
			info.bcdUVC = ci.UVC
			// pull the streaming interfaces too
			for _, i := range ci.VideoStreamingInterfaceIndexes {
				vsbuf := (*[1 << 30]byte)(unsafe.Pointer(ifaces[i].altsetting.extra))[:ifaces[i].altsetting.extra_length]
				asi := &StreamingInterface{usbctx: d.usbctx, usb: &ifaces[i], deviceHandle: d.handle, bcdUVC: info.bcdUVC}
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

func (si *StreamingInterface) ClaimFrameReader(formatIndex, frameIndex uint8) (*FrameReader, error) {
	ifnum := si.usb.altsetting.bInterfaceNumber

	// claim the control interface
	if ret := C.libusb_detach_kernel_driver(si.deviceHandle, C.int(ifnum)); ret < 0 {
		// return nil, fmt.Errorf("libusb_detach_kernel_driver failed: %w", libusberror(ret))
	}
	if ret := C.libusb_claim_interface(si.deviceHandle, C.int(ifnum)); ret < 0 {
		return nil, fmt.Errorf("libusb_claim_interface failed: %w", libusberror(ret))
	}
	vpcc := &descriptors.VideoProbeCommitControl{}
	size := 48

	buf := C.malloc(C.ulong(size))
	defer C.free(buf)

	// get the bounds
	if ret := C.libusb_control_transfer(
		si.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeGetMax),                   /* bRequest */
		1<<8,                                                    /* wValue */
		C.uint16_t(ifnum),                                       /* wIndex */
		(*C.uchar)(buf),                                         /* data */
		C.uint16_t(size),                                        /* len */
		0,                                                       /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	// assign the values
	if err := vpcc.UnmarshalBinary((*[1 << 30]byte)(unsafe.Pointer(buf))[:size]); err != nil {
		return nil, err
	}

	vpcc.FormatIndex = formatIndex
	vpcc.FrameIndex = frameIndex

	if err := vpcc.MarshalInto((*[1 << 30]byte)(unsafe.Pointer(buf))[:size]); err != nil {
		return nil, err
	}

	// call set
	if ret := C.libusb_control_transfer(
		si.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeSetCur),                   /* bRequest */
		1<<8,                                                    /* wValue */
		C.uint16_t(ifnum),                                       /* wIndex */
		(*C.uchar)(buf),                                         /* data */
		C.uint16_t(size),                                        /* len */
		0,                                                       /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	// call get to get the negotiated values
	if ret := C.libusb_control_transfer(
		si.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeGetCur),                   /* bRequest */
		1<<8,                                                    /* wValue */
		C.uint16_t(ifnum),                                       /* wIndex */
		(*C.uchar)(buf),                                         /* data */
		C.uint16_t(size),                                        /* len */
		0,                                                       /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	// perform a commit set
	if ret := C.libusb_control_transfer(
		si.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeSetCur),                   /* bRequest */
		2<<8,                                                    /* wValue */
		C.uint16_t(ifnum),                                       /* wIndex */
		(*C.uchar)(buf),                                         /* data */
		C.uint16_t(size),                                        /* len */
		0,                                                       /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	// unmarshal the negotiated values
	if err := vpcc.UnmarshalBinary(C.GoBytes(unsafe.Pointer(buf), C.int(size))); err != nil {
		return nil, err
	}

	return NewFrameReader(si, vpcc)
}
