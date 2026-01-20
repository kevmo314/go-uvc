package uvc

import (
	"fmt"
	"time"

	usb "github.com/kevmo314/go-usb"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/requests"
)

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
	handle         *usb.DeviceHandle
	ifaceNum       uint8
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
	buf := make([]byte, 16)

	_, err := pu.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceGetRequest),
		uint8(requests.RequestCodeGetCur),
		uint16(desc.Value())<<8,
		uint16(pu.UnitDescriptor.UnitID)<<8|uint16(pu.ifaceNum),
		buf,
		5*time.Second,
	)
	if err != nil {
		return fmt.Errorf("control_transfer failed: %w", err)
	}

	if err := desc.UnmarshalBinary(buf); err != nil {
		return err
	}

	return nil
}

func (pu *ProcessingUnit) Set(desc descriptors.ProcessingUnitControlDescriptor) error {
	buf, err := desc.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = pu.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceSetRequest),
		uint8(requests.RequestCodeSetCur),
		uint16(desc.Value())<<8,
		uint16(pu.UnitDescriptor.UnitID)<<8|uint16(pu.ifaceNum),
		buf,
		5*time.Second,
	)
	if err != nil {
		return fmt.Errorf("control_transfer failed: %w", err)
	}

	return nil
}
