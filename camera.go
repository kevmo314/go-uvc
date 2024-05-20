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

type CameraTerminal struct {
	usb              *C.struct_libusb_interface
	deviceHandle     *C.struct_libusb_device_handle
	CameraDescriptor *descriptors.CameraTerminalDescriptor
}

func (ct *CameraTerminal) GetAutoExposureMode() (descriptors.AutoExposureMode, error) {
	ifnum := ct.usb.altsetting.bInterfaceNumber

	fac := &descriptors.AutoExposureModeControl{}
	size := fac.MarshalSize()

	buf := C.malloc(C.ulong(size))
	defer C.free(buf)

	if ret := C.libusb_control_transfer(
		ct.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest),                                     /* bmRequestType */
		C.uint8_t(requests.RequestCodeGetCur),                                                       /* bRequest*/
		CameraTerminalControlSelectorAutoExposureModeControl<<8,                                     /* wValue: CT_AE_MODE_CONTROL on the hight byte */
		C.uint16_t(uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ifnum)), /* wIndex*/
		(*C.uchar)(buf),  /* data */
		C.uint16_t(size), /* len */
		0,                /* timeout */
	); ret < 0 {
		return 0, fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	if err := fac.UnmarshalBinary(C.GoBytes(unsafe.Pointer(buf), C.int(size))); err != nil {
		return 0, err
	}

	return fac.Mode, nil
}

func (ct *CameraTerminal) SetAutoExposureMode(mode descriptors.AutoExposureMode) error {
	ifnum := ct.usb.altsetting.bInterfaceNumber

	fac := &descriptors.AutoExposureModeControl{Mode: mode}
	size := fac.MarshalSize()

	buf, err := fac.MarshalBinary()
	if err != nil {
		return err
	}

	cPtr := (*C.uchar)(C.CBytes(buf))
	defer C.free(unsafe.Pointer(cPtr))

	if ret := C.libusb_control_transfer(
		ct.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest),                                     /* bmRequestType */
		C.uint8_t(requests.RequestCodeSetCur),                                                       /* bRequest */
		CameraTerminalControlSelectorAutoExposureModeControl<<8,                                     /* wValue: CT_AE_MODE_CONTROL on the hight byte */
		C.uint16_t(uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ifnum)), /* wIndex */
		(*C.uchar)(cPtr), /* data */
		C.uint16_t(size), /* len */
		0,                /* timeout */
	); ret < 0 {
		return fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	return nil
}

func (ct *CameraTerminal) GetAutoFocus() (bool, error) {
	ifnum := ct.usb.altsetting.bInterfaceNumber

	fac := &descriptors.FocusAutoControl{}
	size := fac.MarshalSize()

	buf := C.malloc(C.ulong(size))
	defer C.free(buf)

	//GET_CUR
	if ret := C.libusb_control_transfer(
		ct.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest),                                     /* bmRequestType */
		C.uint8_t(requests.RequestCodeGetCur),                                                       /* bRequest*/
		CameraTerminalControlSelectorFocusAutoControl<<8,                                            /* wValue: CT_FOCUS_AUTO_CONTROL on the hight byte */
		C.uint16_t(uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ifnum)), /* wIndex*/
		(*C.uchar)(buf),  /* data */
		C.uint16_t(size), /* len */
		0,                /* timeout */
	); ret < 0 {
		return false, fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	if err := fac.UnmarshalBinary(C.GoBytes(unsafe.Pointer(buf), C.int(size))); err != nil {
		return false, err
	}

	return fac.FocusAuto, nil
}

func (ct *CameraTerminal) SetAutoFocus(on bool) error {
	ifnum := ct.usb.altsetting.bInterfaceNumber

	fac := &descriptors.FocusAutoControl{FocusAuto: on}
	size := fac.MarshalSize()

	buf, err := fac.MarshalBinary()
	if err != nil {
		return err
	}

	cPtr := (*C.uchar)(C.CBytes(buf))
	defer C.free(unsafe.Pointer(cPtr))

	//SET_CUR
	if ret := C.libusb_control_transfer(
		ct.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest),                                     /* bmRequestType */
		C.uint8_t(requests.RequestCodeSetCur),                                                       /* bRequest */
		CameraTerminalControlSelectorFocusAutoControl<<8,                                            /* wValue: CT_FOCUS_AUTO_CONTROL on the hight byte */
		C.uint16_t(uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ifnum)), /* wIndex */
		(*C.uchar)(cPtr), /* data */
		C.uint16_t(size), /* len */
		0,                /* timeout */
	); ret < 0 {
		return fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	return nil
}
