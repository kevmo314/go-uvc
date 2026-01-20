//go:build !windows

package uvc

import (
	"fmt"
	"time"

	usb "github.com/kevmo314/go-usb"
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
	handle           *usb.DeviceHandle
	ifaceNum         uint8
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
	buf := make([]byte, 16)

	_, err := ct.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceGetRequest),
		uint8(requests.RequestCodeGetCur),
		uint16(desc.Value())<<8,
		uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ct.ifaceNum),
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

func (ct *CameraTerminal) Set(desc descriptors.CameraTerminalControlDescriptor) error {
	buf, err := desc.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = ct.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceSetRequest),
		uint8(requests.RequestCodeSetCur),
		uint16(desc.Value())<<8,
		uint16(ct.CameraDescriptor.InputTerminalDescriptor.TerminalID)<<8|uint16(ct.ifaceNum),
		buf,
		5*time.Second,
	)
	if err != nil {
		return fmt.Errorf("control_transfer failed: %w", err)
	}

	return nil
}
