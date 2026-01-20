package transfers

import (
	"fmt"
	"time"

	usb "github.com/kevmo314/go-usb"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/requests"
)

type VideoStreamingInterfaceControlSelector int

const (
	VideoStreamingInterfaceControlSelectorUndefined                 VideoStreamingInterfaceControlSelector = 0x00
	VideoStreamingInterfaceControlSelectorProbeControl                                                     = 0x01
	VideoStreamingInterfaceControlSelectorCommitControl                                                    = 0x02
	VideoStreamingInterfaceControlSelectorStillProbeControl                                                = 0x03
	VideoStreamingInterfaceControlSelectorStillCommitControl                                               = 0x04
	VideoStreamingInterfaceControlSelectorStillImageTriggerControl                                         = 0x05
	VideoStreamingInterfaceControlSelectorStreamErrorCodeControl                                           = 0x06
	VideoStreamingInterfaceControlSelectorGenerateKeyFrameControl                                          = 0x07
	VideoStreamingInterfaceControlSelectorUpdateFrameSegmentControl                                        = 0x08
	VideoStreamingInterfaceControlSelectorSynchDelayControl                                                = 0x09
)

type StreamingInterface struct {
	bcdUVC      uint16 // cached since it's used a lot
	handle      *usb.DeviceHandle
	iface       *usb.Interface
	Descriptors []descriptors.StreamingInterface
}

func NewStreamingInterface(handle *usb.DeviceHandle, iface *usb.Interface, bcdUVC uint16) *StreamingInterface {
	return &StreamingInterface{handle: handle, iface: iface, bcdUVC: bcdUVC}
}

func (si *StreamingInterface) InterfaceNumber() uint8 {
	if len(si.iface.AltSettings) == 0 {
		return 0
	}
	return si.iface.AltSettings[0].InterfaceNumber
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

func (si *StreamingInterface) ClaimFrameReader(formatIndex, frameIndex uint8) (*FrameReader, error) {
	ifnum := si.InterfaceNumber()

	// Also claim the control interface (interface 0) for UVC control requests
	si.handle.DetachKernelDriver(0)
	if err := si.handle.ClaimInterface(0); err != nil {
		// Control interface claim failure is not fatal, but log it
		// Some devices may not require it
	}

	// Detach and claim the streaming interface
	si.handle.DetachKernelDriver(ifnum)
	if err := si.handle.ClaimInterface(ifnum); err != nil {
		return nil, fmt.Errorf("claim_interface failed: %w", err)
	}

	vpcc := &descriptors.VideoProbeCommitControl{}
	size := 48
	buf := make([]byte, size)

	// get the bounds
	_, err := si.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceGetRequest),
		uint8(requests.RequestCodeGetMax),
		uint16(VideoStreamingInterfaceControlSelectorProbeControl)<<8,
		uint16(ifnum),
		buf,
		5*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("control_transfer GET_MAX failed: %w", err)
	}

	// assign the values
	if err := vpcc.UnmarshalBinary(buf); err != nil {
		return nil, err
	}

	vpcc.FormatIndex = formatIndex
	vpcc.FrameIndex = frameIndex

	if err := vpcc.MarshalInto(buf); err != nil {
		return nil, err
	}

	// call set
	_, err = si.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceSetRequest),
		uint8(requests.RequestCodeSetCur),
		uint16(VideoStreamingInterfaceControlSelectorProbeControl)<<8,
		uint16(ifnum),
		buf,
		5*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("control_transfer SET_CUR probe failed: %w", err)
	}

	// call get to get the negotiated values
	_, err = si.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceGetRequest),
		uint8(requests.RequestCodeGetCur),
		uint16(VideoStreamingInterfaceControlSelectorProbeControl)<<8,
		uint16(ifnum),
		buf,
		5*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("control_transfer GET_CUR probe failed: %w", err)
	}

	// perform a commit set
	_, err = si.handle.ControlTransfer(
		uint8(requests.RequestTypeVideoInterfaceSetRequest),
		uint8(requests.RequestCodeSetCur),
		uint16(VideoStreamingInterfaceControlSelectorCommitControl)<<8,
		uint16(ifnum),
		buf,
		5*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("control_transfer SET_CUR commit failed: %w", err)
	}

	// unmarshal the negotiated values
	if err := vpcc.UnmarshalBinary(buf); err != nil {
		return nil, err
	}

	inputs := si.InputHeaderDescriptors()
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no input header descriptors found")
	}
	endpointAddress := inputs[0].EndpointAddress // take the first input header. TODO: should we select an input header?

	return si.NewFrameReader(endpointAddress, vpcc)
}

func (si *StreamingInterface) Handle() *usb.DeviceHandle {
	return si.handle
}

func (si *StreamingInterface) Interface() *usb.Interface {
	return si.iface
}
