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
	"github.com/kevmo314/go-uvc/pkg/requests"
)

var availableDescriptors = []descriptors.CameraTerminalControlDescriptor{
	&descriptors.ScanningModeControl{},
	&descriptors.AutoExposureModeControl{},
	&descriptors.AutoExposurePriorityControl{},
	&descriptors.DigitalWindowControl{},
	&descriptors.PrivacyControl{},
	&descriptors.FocusAbsoluteControl{},
	&descriptors.FocusAutoControl{},
	&descriptors.ExposureTimeAbsoluteControl{},
	&descriptors.ExposureTimeRelativeControl{},
	&descriptors.FocusRelativeControl{},
	&descriptors.FocusSimpleRangeControl{},
	&descriptors.RollAbsoluteControl{},
	&descriptors.IrisAbsoluteControl{},
	&descriptors.IrisRelativeControl{},
	&descriptors.PanTiltAbsoluteControl{},
	&descriptors.PanTiltRelativeControl{},
	&descriptors.RegionOfInterestControl{},
	&descriptors.RollRelativeControl{},
	&descriptors.ZoomAbsoluteControl{},
	&descriptors.ZoomRelativeControl{},
}

type CameraTerminal struct {
	usb              *C.struct_libusb_interface
	deviceHandle     *C.struct_libusb_device_handle
	CameraDescriptor *descriptors.CameraTerminalDescriptor
}

func (ct *CameraTerminal) GetSupportedControls() []descriptors.CameraTerminalControlDescriptor {
	var supportedControls []descriptors.CameraTerminalControlDescriptor

	for _, desc := range availableDescriptors {
		if ct.IsControlRequestSupported(desc) {
			supportedControls = append(supportedControls, desc)
		}
	}
	return supportedControls
}

func (ct *CameraTerminal) IsControlRequestSupported(desc descriptors.CameraTerminalControlDescriptor) bool {
	byteIndex := desc.FeatureBit() / 8
	bitIndex := desc.FeatureBit() % 8
	return (ct.CameraDescriptor.ControlsBitmask[byteIndex] & (1 << bitIndex)) != 0
}

func (ct *CameraTerminal) Get(desc descriptors.CameraTerminalControlDescriptor) error {
	ifnum := ct.usb.altsetting.bInterfaceNumber

	bufLen := 16
	buf := C.malloc(C.ulong(bufLen))
	defer C.free(buf)

	if ret := C.libusb_control_transfer(
		ct.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest),                                     /* bmRequestType */
		C.uint8_t(requests.RequestCodeGetCur),                                                       /* bRequest*/
		C.uint16_t(desc.Value()<<8),                                                                 /* wValue: on the hight byte */
		C.uint16_t(uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ifnum)), /* wIndex*/
		(*C.uchar)(buf),    /* data */
		C.uint16_t(bufLen), /* len */
		0,                  /* timeout */
	); ret < 0 {
		return fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	if err := desc.UnmarshalBinary(C.GoBytes(unsafe.Pointer(buf), C.int(bufLen))); err != nil {
		return err
	}

	return nil
}

func (ct *CameraTerminal) Set(desc descriptors.CameraTerminalControlDescriptor) error {
	ifnum := ct.usb.altsetting.bInterfaceNumber

	buf, err := desc.MarshalBinary()
	if err != nil {
		return err
	}

	cPtr := (*C.uchar)(C.CBytes(buf))
	defer C.free(unsafe.Pointer(cPtr))

	if ret := C.libusb_control_transfer(
		ct.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest),                                     /* bmRequestType */
		C.uint8_t(requests.RequestCodeSetCur),                                                       /* bRequest */
		C.uint16_t(desc.Value()<<8),                                                                 /* wValue: on the hight byte */
		C.uint16_t(uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ifnum)), /* wIndex */
		(*C.uchar)(cPtr),     /* data */
		C.uint16_t(len(buf)), /* len */
		0,                    /* timeout */
	); ret < 0 {
		return fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	return nil
}
