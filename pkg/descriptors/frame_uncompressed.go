package descriptors

import (
	"encoding/binary"
	"io"
	"time"
)

type UncompressedStreamHeader struct {
	BitFieldHeader uint8
	PTS            uint32
	SCR            uint64
}

func (ush *UncompressedStreamHeader) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	ush.BitFieldHeader = buf[1]
	offset := 2
	if ush.HasPTS() {
		ush.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if ush.HasSCR() {
		ush.SCR = binary.LittleEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}
	return nil
}

func (ush *UncompressedStreamHeader) FrameIdentifier() bool {
	return ush.BitFieldHeader&0b00000001 != 0
}

func (ush *UncompressedStreamHeader) EndOfFrame() bool {
	return ush.BitFieldHeader&0b00000010 != 0
}

func (ush *UncompressedStreamHeader) HasPTS() bool {
	return ush.BitFieldHeader&0b00000100 != 0
}

func (ush *UncompressedStreamHeader) HasSCR() bool {
	return ush.BitFieldHeader&0b00001000 != 0
}

func (ush *UncompressedStreamHeader) StillImage() bool {
	return ush.BitFieldHeader&0b00100000 != 0
}

func (ush *UncompressedStreamHeader) Error() bool {
	return ush.BitFieldHeader&0b01000000 != 0
}

func (ush *UncompressedStreamHeader) EndOfHeader() bool {
	return ush.BitFieldHeader&0b10000000 != 0
}

type UncompressedFormatDescriptor struct {
	FormatIndex           uint8
	NumFrameDescriptors   uint8
	GUIDFormat            [16]byte
	BitsPerPixel          uint8
	DefaultFrameIndex     uint8
	AspectRatioX          uint8
	AspectRatioY          uint8
	InterlaceFlagsBitmask uint8
	CopyProtect           uint8
}

func (ufd *UncompressedFormatDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatUncompressed {
		return ErrInvalidDescriptor
	}
	ufd.FormatIndex = buf[3]
	ufd.NumFrameDescriptors = buf[4]
	copy(ufd.GUIDFormat[:], buf[5:21])
	ufd.BitsPerPixel = buf[21]
	ufd.DefaultFrameIndex = buf[22]
	ufd.AspectRatioX = buf[23]
	ufd.AspectRatioY = buf[24]
	ufd.InterlaceFlagsBitmask = buf[25]
	ufd.CopyProtect = buf[26]
	return nil
}

type UncompressedFrameDescriptor struct {
	FrameIndex              uint8
	Capabilities            uint8
	Width, Height           uint16
	MinBitRate, MaxBitRate  uint32
	MaxVideoFrameBufferSize uint32
	DefaultFrameInterval    time.Duration

	ContinuousFrameInterval struct {
		MinFrameInterval, MaxFrameInterval, FrameIntervalStep time.Duration
	}
	DiscreteFrameIntervals []time.Duration
}

func (ufd *UncompressedFrameDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFrameUncompressed {
		return ErrInvalidDescriptor
	}
	ufd.FrameIndex = buf[3]
	ufd.Capabilities = buf[4]
	ufd.Width = binary.LittleEndian.Uint16(buf[5:7])
	ufd.Height = binary.LittleEndian.Uint16(buf[7:9])
	ufd.MinBitRate = binary.LittleEndian.Uint32(buf[9:13])
	ufd.MaxBitRate = binary.LittleEndian.Uint32(buf[13:17])
	ufd.MaxVideoFrameBufferSize = binary.LittleEndian.Uint32(buf[17:21])
	ufd.DefaultFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[21:25])) * 100 * time.Nanosecond

	n := buf[25]

	if n == 0 {
		// Continuous frame intervals
		ufd.ContinuousFrameInterval.MinFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[26:30])) * 100 * time.Nanosecond
		ufd.ContinuousFrameInterval.MaxFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[30:34])) * 100 * time.Nanosecond
		ufd.ContinuousFrameInterval.FrameIntervalStep = time.Duration(binary.LittleEndian.Uint32(buf[34:38])) * 100 * time.Nanosecond
		return nil
	} else {
		ufd.DiscreteFrameIntervals = make([]time.Duration, n)
		for i := uint8(0); i < n; i++ {
			ufd.DiscreteFrameIntervals[i] = time.Duration(binary.LittleEndian.Uint32(buf[26+i*4:30+i*4])) * 100 * time.Nanosecond
		}
		return nil
	}
}
