// This file implements the descriptors as defined in the UVC spec 1.5, section 3.8.
package descriptors

import (
	"encoding/binary"
	"io"
)

type VideoControlEndpointDescriptorSubtype byte

const (
	VideoControlEndpointDescriptorSubtypeUndefined VideoControlEndpointDescriptorSubtype = 0x00
	VideoControlEndpointDescriptorSubtypeGeneral   VideoControlEndpointDescriptorSubtype = 0x01
	VideoControlEndpointDescriptorSubtypeEndpoint  VideoControlEndpointDescriptorSubtype = 0x02
	VideoControlEndpointDescriptorSubtypeInterrupt VideoControlEndpointDescriptorSubtype = 0x03
)

type StandardVideoControlInterruptEndpointDescriptor struct {
	MaxTransferSize uint16
}

func (svcie *StandardVideoControlInterruptEndpointDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeEndpoint {
		return ErrInvalidDescriptor
	}
	if VideoControlEndpointDescriptorSubtype(buf[2]) != VideoControlEndpointDescriptorSubtypeInterrupt {
		return ErrInvalidDescriptor
	}
	svcie.MaxTransferSize = binary.LittleEndian.Uint16(buf[2:4])
	return nil
}
