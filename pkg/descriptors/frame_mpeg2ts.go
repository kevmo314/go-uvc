package descriptors

import (
	"io"
)

type MPEG2TSStreamHeader struct {
	BitFieldHeader uint8
}

func (msh *MPEG2TSStreamHeader) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	msh.BitFieldHeader = buf[1]
	return nil
}

func (msh *MPEG2TSStreamHeader) FrameIdentifier() bool {
	return msh.BitFieldHeader&0b00000001 != 0
}

func (msh *MPEG2TSStreamHeader) EndOfFrame() bool {
	return msh.BitFieldHeader&0b00000010 != 0
}

func (msh *MPEG2TSStreamHeader) StillImage() bool {
	return msh.BitFieldHeader&0b00100000 != 0
}

func (msh *MPEG2TSStreamHeader) Error() bool {
	return msh.BitFieldHeader&0b01000000 != 0
}

func (msh *MPEG2TSStreamHeader) EndOfHeader() bool {
	return msh.BitFieldHeader&0b10000000 != 0
}

type MPEG2TSFormatDescriptor struct {
	FormatIndex      uint8
	DataOffset       uint8
	PacketLength     uint8
	StrideLength     uint8
	GUIDStrideFormat [16]byte
}

func (mfd *MPEG2TSFormatDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatMPEG2TS {
		return ErrInvalidDescriptor
	}
	mfd.FormatIndex = buf[3]
	mfd.DataOffset = buf[4]
	mfd.PacketLength = buf[5]
	mfd.StrideLength = buf[6]
	copy(mfd.GUIDStrideFormat[:], buf[7:23])
	return nil
}

func (mfd *MPEG2TSFormatDescriptor) isStreamingInterface() {}

func (mfd *MPEG2TSFormatDescriptor) isFormatDescriptor() {}
