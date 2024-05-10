// This file implements the descriptors as defined in the UVC spec 1.5, section 3.10.
package descriptors

import (
	"encoding/binary"
	"io"
)

// StandardVideoStreamingIsochronousVideoDataEndpointDescriptor as defined in UVC spec 1.5, 3.10.1.1
type StandardVideoStreamingIsochronousVideoDataEndpointDescriptor struct {
	EndpointAddress   uint8
	AttributesBitmask uint8
	MaxPacketSize     uint16
	Interval          uint8
}

func (svsived *StandardVideoStreamingIsochronousVideoDataEndpointDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	// TODO: fix the descriptor type, this is not the class specific one.
	// if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeEndpoint {
	// 	return ErrInvalidDescriptor
	// }
	svsived.EndpointAddress = buf[2]
	svsived.AttributesBitmask = buf[3]
	svsived.MaxPacketSize = binary.LittleEndian.Uint16(buf[4:6])
	svsived.Interval = buf[6]
	return nil
}

// StandardVideoStreamingBulkVideoDataEndpointDescriptor as defined in UVC spec 1.5, 3.10.1.2
type StandardVideoStreamingBulkVideoDataEndpointDescriptor struct {
	EndpointAddress uint8
	MaxPacketSize   uint16
	Interval        uint8
}

func (svsbded *StandardVideoStreamingBulkVideoDataEndpointDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	// TODO: fix the descriptor type, this is not the class specific one.
	// if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeEndpoint {
	// 	return ErrInvalidDescriptor
	// }
	svsbded.EndpointAddress = buf[2]
	if buf[3] != 0b10 { // 0b10 == Bulk
		return ErrInvalidDescriptor
	}
	svsbded.MaxPacketSize = binary.LittleEndian.Uint16(buf[4:6])
	svsbded.Interval = buf[6]
	return nil
}

// StandardVideoStreamingBulkStillImageDataEndpointDescriptor as defined in UVC spec 1.5, 3.10.1.3
type StandardVideoStreamingBulkStillImageDataEndpointDescriptor struct {
	EndpointAddress uint8
	MaxPacketSize   uint16
}

func (svsbied *StandardVideoStreamingBulkStillImageDataEndpointDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	// TODO: fix the descriptor type, this is not the class specific one.
	// if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeEndpoint {
	// 	return ErrInvalidDescriptor
	// }
	svsbied.EndpointAddress = buf[2]
	if svsbied.EndpointAddress&0b10000000 == 0 { // Direction == IN
		return ErrInvalidDescriptor
	}
	if svsbied.EndpointAddress&0b01111000 != 0 { // Reserved bits
		return ErrInvalidDescriptor
	}
	if buf[3] != 0b10 { // 0b10 == Bulk
		return ErrInvalidDescriptor
	}
	svsbied.MaxPacketSize = binary.LittleEndian.Uint16(buf[4:6])
	if buf[6] != 0 {
		return ErrInvalidDescriptor
	}
	return nil
}
