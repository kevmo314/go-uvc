package descriptors

import (
	"encoding/binary"
	"io"
)

type StreamBasedStreamHeader struct {
	BitFieldHeader uint8
	PTS            uint32
	SCR            uint64
}

func (sbsh *StreamBasedStreamHeader) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	sbsh.BitFieldHeader = buf[1]
	offset := 2
	if sbsh.HasPTS() {
		sbsh.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if sbsh.HasSCR() {
		sbsh.SCR = binary.LittleEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}
	return nil
}

func (sbsh *StreamBasedStreamHeader) FrameIdentifier() bool {
	return sbsh.BitFieldHeader&0b00000001 != 0
}

func (sbsh *StreamBasedStreamHeader) EndOfFrame() bool {
	return sbsh.BitFieldHeader&0b00000010 != 0
}

func (sbsh *StreamBasedStreamHeader) HasPTS() bool {
	return sbsh.BitFieldHeader&0b00000100 != 0
}

func (sbsh *StreamBasedStreamHeader) HasSCR() bool {
	return sbsh.BitFieldHeader&0b00001000 != 0
}

func (sbsh *StreamBasedStreamHeader) StillImage() bool {
	return sbsh.BitFieldHeader&0b00100000 != 0
}

func (sbsh *StreamBasedStreamHeader) Error() bool {
	return sbsh.BitFieldHeader&0b01000000 != 0
}

func (sbsh *StreamBasedStreamHeader) EndOfHeader() bool {
	return sbsh.BitFieldHeader&0b10000000 != 0
}

type StreamBasedFormatDescriptor struct {
	FormatIndex  uint8
	GUIDFormat   [16]byte
	PacketLength uint32
}

func (sbfd *StreamBasedFormatDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatStreamBased {
		return ErrInvalidDescriptor
	}
	sbfd.FormatIndex = buf[3]
	copy(sbfd.GUIDFormat[:], buf[4:20])
	sbfd.PacketLength = binary.LittleEndian.Uint32(buf[20:24])
	return nil
}
