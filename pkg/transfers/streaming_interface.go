package transfers

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
	ctx         *C.libusb_context
	handle      *C.struct_libusb_device_handle
	iface       *C.struct_libusb_interface
	Descriptors []descriptors.StreamingInterface
}

func NewStreamingInterface(ctxp, handlep, ifacep unsafe.Pointer, bcdUVC uint16) *StreamingInterface {
	ctx := (*C.struct_libusb_context)(ctxp)
	handle := (*C.struct_libusb_device_handle)(handlep)
	iface := (*C.struct_libusb_interface)(ifacep)
	return &StreamingInterface{ctx: ctx, handle: handle, iface: iface, bcdUVC: bcdUVC}
}

func (si *StreamingInterface) InterfaceNumber() uint8 {
	return uint8(si.iface.altsetting.bInterfaceNumber)
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
	ifnum := si.iface.altsetting.bInterfaceNumber

	// claim the control interface
	if ret := C.libusb_detach_kernel_driver(si.handle, C.int(ifnum)); ret < 0 {
		// return nil, fmt.Errorf("libusb_detach_kernel_driver failed: %s", C.GoString(C.libusb_error_name(ret)))
	}
	if ret := C.libusb_claim_interface(si.handle, C.int(ifnum)); ret < 0 {
		return nil, fmt.Errorf("libusb_claim_interface failed: %s", C.GoString(C.libusb_error_name(ret)))
	}
	vpcc := &descriptors.VideoProbeCommitControl{}
	size := 48

	buf := C.malloc(C.size_t(size))
	defer C.free(buf)

	// get the bounds
	if ret := C.libusb_control_transfer(
		si.handle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeGetMax),                   /* bRequest */
		VideoStreamingInterfaceControlSelectorProbeControl<<8,   /* wValue */
		C.uint16_t(ifnum), /* wIndex */
		(*C.uchar)(buf),   /* data */
		C.uint16_t(size),  /* len */
		0,                 /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	// assign the values
	if err := vpcc.UnmarshalBinary(unsafe.Slice((*byte)(buf), size)); err != nil {
		return nil, err
	}

	vpcc.FormatIndex = formatIndex
	vpcc.FrameIndex = frameIndex

	if err := vpcc.MarshalInto(unsafe.Slice((*byte)(buf), size)); err != nil {
		return nil, err
	}

	// call set
	if ret := C.libusb_control_transfer(
		si.handle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeSetCur),                   /* bRequest */
		VideoStreamingInterfaceControlSelectorProbeControl<<8,   /* wValue */
		C.uint16_t(ifnum), /* wIndex */
		(*C.uchar)(buf),   /* data */
		C.uint16_t(size),  /* len */
		0,                 /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	// call get to get the negotiated values
	if ret := C.libusb_control_transfer(
		si.handle,
		C.uint8_t(requests.RequestTypeVideoInterfaceGetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeGetCur),                   /* bRequest */
		VideoStreamingInterfaceControlSelectorProbeControl<<8,   /* wValue */
		C.uint16_t(ifnum), /* wIndex */
		(*C.uchar)(buf),   /* data */
		C.uint16_t(size),  /* len */
		0,                 /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	// perform a commit set
	if ret := C.libusb_control_transfer(
		si.handle,
		C.uint8_t(requests.RequestTypeVideoInterfaceSetRequest), /* bmRequestType */
		C.uint8_t(requests.RequestCodeSetCur),                   /* bRequest */
		VideoStreamingInterfaceControlSelectorCommitControl<<8,  /* wValue */
		C.uint16_t(ifnum), /* wIndex */
		(*C.uchar)(buf),   /* data */
		C.uint16_t(size),  /* len */
		0,                 /* timeout */
	); ret < 0 {
		return nil, fmt.Errorf("libusb_control_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	// unmarshal the negotiated values
	if err := vpcc.UnmarshalBinary(unsafe.Slice((*byte)(buf), size)); err != nil {
		return nil, err
	}

	inputs := si.InputHeaderDescriptors()
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no input header descriptors found")
	}
	endpointAddress := inputs[0].EndpointAddress // take the first input header. TODO: should we select an input header?

	return si.NewFrameReader(endpointAddress, vpcc)
}
