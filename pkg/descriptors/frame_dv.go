package descriptors

import (
	"encoding/binary"
	"io"
)

type DVStreamHeader struct {
	BitFieldHeader uint8
	PTS            uint32
	SCR            uint64
}

func (dvsh *DVStreamHeader) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	dvsh.BitFieldHeader = buf[1]
	offset := 2
	if dvsh.HasPTS() {
		dvsh.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if dvsh.HasSCR() {
		dvsh.SCR = binary.LittleEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}
	return nil
}

func (dvsh *DVStreamHeader) FrameIdentifier() bool {
	return dvsh.BitFieldHeader&0b00000001 != 0
}

func (dvsh *DVStreamHeader) EndOfFrame() bool {
	return dvsh.BitFieldHeader&0b00000010 != 0
}

func (dvsh *DVStreamHeader) HasPTS() bool {
	return dvsh.BitFieldHeader&0b00000100 != 0
}

func (dvsh *DVStreamHeader) HasSCR() bool {
	return dvsh.BitFieldHeader&0b00001000 != 0
}

func (dvsh *DVStreamHeader) StillImage() bool {
	return dvsh.BitFieldHeader&0b00100000 != 0
}

func (dvsh *DVStreamHeader) Error() bool {
	return dvsh.BitFieldHeader&0b01000000 != 0
}

func (dvsh *DVStreamHeader) EndOfHeader() bool {
	return dvsh.BitFieldHeader&0b10000000 != 0
}

type DVFormatDescriptor struct {
	FormatIndex             uint8
	MaxVideoFrameBufferSize uint32
	FormatType              uint8
}

func (dvfd *DVFormatDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatDV {
		return ErrInvalidDescriptor
	}
	dvfd.FormatIndex = buf[3]
	dvfd.MaxVideoFrameBufferSize = binary.LittleEndian.Uint32(buf[4:8])
	dvfd.FormatType = buf[8]
	return nil
}
