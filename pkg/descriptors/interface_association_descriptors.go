// This file implements the descriptors as defined in the UVC spec 1.5, section 3.6.
package descriptors

import "io"

type InterfaceAssociationDescriptor struct {
	InterfaceCount   uint8
	DescriptionIndex uint8
}

func (iad *InterfaceAssociationDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if buf[1] != 0xEF { // TODO: where does this come from
		return ErrInvalidDescriptor
	}
	iad.InterfaceCount = buf[2]
	if ClassCode(buf[3]) != ClassCodeVideo {
		return ErrInvalidDescriptor
	}
	if SubclassCode(buf[4]) != SubclassCodeVideoInterfaceCollection {
		return ErrInvalidDescriptor
	}
	if ProtocolCode(buf[5]) != ProtocolCodeUndefined {
		return ErrInvalidDescriptor
	}
	iad.DescriptionIndex = buf[6]
	return nil
}
