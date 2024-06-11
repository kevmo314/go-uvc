package uvc

import (
	"fmt"
	"unsafe"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/requests"
)

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"

var puControls = []descriptors.ProcessingUnitControlDescriptor{
	&descriptors.BacklightCompensationControl{},
	&descriptors.BrightnessControl{},
	&descriptors.ContrastControl{},
	&descriptors.ContrastAutoControl{},
	&descriptors.GainControl{},
	&descriptors.PowerLineFrequencyControl{},
	&descriptors.HueControl{},
	&descriptors.HueAutoControl{},
	&descriptors.SaturationControl{},
	&descriptors.SharpnessControl{},
	&descriptors.GammaControl{},
	&descriptors.WhiteBalanceTemperatureControl{},
	&descriptors.WhiteBalanceTemperatureAutoControl{},
	&descriptors.WhiteBalanceComponentControl{},
	&descriptors.WhiteBalanceComponentAutoControl{},
	&descriptors.DigitalMultiplerControl{},
	&descriptors.DigitalMultiplerLimitControl{},
	&descriptors.AnalogVideoStandardControl{},
	&descriptors.AnalogVideoLockStatusControl{},
}

type ProcessingUnit struct {
	usb            *C.struct_libusb_interface
	deviceHandle   *C.struct_libusb_device_handle
	UnitDescriptor *descriptors.ProcessingUnitDescriptor
}

func (pu *ProcessingUnit) GetSupportedControls() []descriptors.ProcessingUnitControlDescriptor {
	var supportedControls []descriptors.ProcessingUnitControlDescriptor

	for _, desc := range puControls {
		if pu.IsControlRequestSupported(desc) {
			supportedControls = append(supportedControls, desc)
		}
	}
	return supportedControls
}

func (pu *ProcessingUnit) IsControlRequestSupported(desc descriptors.ProcessingUnitControlDescriptor) bool {
	byteIndex := desc.FeatureBit() / 8
	bitIndex := desc.FeatureBit() % 8

	// Support devices that follow older UVC versions (PUD lenght 10+n vs 13). See UVC 1.1
	if byteIndex >= len(pu.UnitDescriptor.ControlsBitmask) {
		return false
	}

	return (pu.UnitDescriptor.ControlsBitmask[byteIndex] & (1 << bitIndex)) != 0
}

func (pu *ProcessingUnit) Get(desc descriptors.ProcessingUnitControlDescriptor) error {
	ifnum := pu.usb.altsetting.bInterfaceNumber

	bufLen := 16
	buf := C.malloc(C.ulong(bufLen))
	defer C.free(buf)

	if ret := C.libusb_control_transfer(
		pu.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest),
		C.uint8_t(requests.RequestCodeGetCur),
		C.uint16_t(desc.Value()<<8),
		C.uint16_t(uint16(pu.UnitDescriptor.UnitID)<<8|uint16(ifnum)),
		(*C.uchar)(buf),
		C.uint16_t(bufLen),
		0,
	); ret < 0 {
		return fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	if err := desc.UnmarshalBinary(unsafe.Slice((*byte)(buf), bufLen)); err != nil {
		return err
	}

	return nil
}

func (pu *ProcessingUnit) Set(desc descriptors.ProcessingUnitControlDescriptor) error {
	ifnum := pu.usb.altsetting.bInterfaceNumber

	buf, err := desc.MarshalBinary()
	if err != nil {
		return err
	}

	cPtr := (*C.uchar)(C.CBytes(buf))
	defer C.free(unsafe.Pointer(cPtr))

	if ret := C.libusb_control_transfer(
		pu.deviceHandle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest),
		C.uint8_t(requests.RequestCodeSetCur),
		C.uint16_t(desc.Value()<<8),
		C.uint16_t(uint16(pu.UnitDescriptor.UnitID)<<8|uint16(ifnum)),
		(*C.uchar)(cPtr),
		C.uint16_t(len(buf)),
		0,
	); ret < 0 {
		return fmt.Errorf("libusb_control_transfer failed: %w", libusberror(ret))
	}

	return nil
}
