package descriptors

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/google/uuid"
)

type FrameBasedStreamHeader struct {
	BitFieldHeader uint8
	PTS            uint32
	SCR            uint64
}

func (fbsh *FrameBasedStreamHeader) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	fbsh.BitFieldHeader = buf[1]
	offset := 2
	if fbsh.HasPTS() {
		fbsh.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if fbsh.HasSCR() {
		fbsh.SCR = binary.LittleEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}
	return nil
}

func (fbsh *FrameBasedStreamHeader) FrameIdentifier() bool {
	return fbsh.BitFieldHeader&0b00000001 != 0
}

func (fbsh *FrameBasedStreamHeader) EndOfFrame() bool {
	return fbsh.BitFieldHeader&0b00000010 != 0
}

func (fbsh *FrameBasedStreamHeader) HasPTS() bool {
	return fbsh.BitFieldHeader&0b00000100 != 0
}

func (fbsh *FrameBasedStreamHeader) HasSCR() bool {
	return fbsh.BitFieldHeader&0b00001000 != 0
}

func (fbsh *FrameBasedStreamHeader) StillImage() bool {
	return fbsh.BitFieldHeader&0b00100000 != 0
}

func (fbsh *FrameBasedStreamHeader) Error() bool {
	return fbsh.BitFieldHeader&0b01000000 != 0
}

func (fbsh *FrameBasedStreamHeader) EndOfHeader() bool {
	return fbsh.BitFieldHeader&0b10000000 != 0
}

type FrameBasedFormatDescriptor struct {
	FormatIndex         uint8
	NumFrameDescriptors uint8
	GUIDFormat          uuid.UUID
	BitsPerPixel        uint8
	DefaultFrameIndex   uint8
	AspectRatioX        uint8
	AspectRatioY        uint8
	InterlaceFlags      uint8
	CopyProtect         uint8
}

func (fbfd *FrameBasedFormatDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatFrameBased {
		return ErrInvalidDescriptor
	}
	fbfd.FormatIndex = buf[3]
	fbfd.NumFrameDescriptors = buf[4]
	copy(fbfd.GUIDFormat[:], buf[5:21])
	fbfd.BitsPerPixel = buf[21]
	fbfd.DefaultFrameIndex = buf[22]
	fbfd.AspectRatioX = buf[23]
	fbfd.AspectRatioY = buf[24]
	fbfd.InterlaceFlags = buf[25]
	fbfd.CopyProtect = buf[26]
	return nil
}

func (fbfd *FrameBasedFormatDescriptor) isStreamingInterface() {}

func (fbfd *FrameBasedFormatDescriptor) isFormatDescriptor() {}

type FrameBasedFrameDescriptor struct {
	FrameIndex             uint8
	Capabilities           uint8
	Width, Height          uint16
	MinBitRate, MaxBitRate uint32
	DefaultFrameInterval   time.Duration

	BytesPerLine uint32

	ContinuousFrameInterval struct {
		MinFrameInterval, MaxFrameInterval, FrameIntervalStep time.Duration
	}
	DiscreteFrameIntervals []time.Duration
}

func (fbfd *FrameBasedFrameDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFrameFrameBased {
		return ErrInvalidDescriptor
	}
	fbfd.FrameIndex = buf[3]
	fbfd.Capabilities = buf[4]
	fbfd.Width = binary.LittleEndian.Uint16(buf[5:7])
	fbfd.Height = binary.LittleEndian.Uint16(buf[7:9])
	fbfd.MinBitRate = binary.LittleEndian.Uint32(buf[9:13])
	fbfd.MaxBitRate = binary.LittleEndian.Uint32(buf[13:17])
	fbfd.DefaultFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[17:21])) * 100 * time.Nanosecond

	n := buf[21]

	fbfd.BytesPerLine = binary.LittleEndian.Uint32(buf[22:26])

	if n == 0 {
		// Continuous frame intervals
		fbfd.ContinuousFrameInterval.MinFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[26:30])) * 100 * time.Nanosecond
		fbfd.ContinuousFrameInterval.MaxFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[30:34])) * 100 * time.Nanosecond
		fbfd.ContinuousFrameInterval.FrameIntervalStep = time.Duration(binary.LittleEndian.Uint32(buf[34:38])) * 100 * time.Nanosecond
		return nil
	} else {
		fbfd.DiscreteFrameIntervals = make([]time.Duration, n)
		for i := uint8(0); i < n; i++ {
			fbfd.DiscreteFrameIntervals[i] = time.Duration(binary.LittleEndian.Uint32(buf[26+i*4:30+i*4])) * 100 * time.Nanosecond
		}
		return nil
	}
}

func (fbfd *FrameBasedFrameDescriptor) isStreamingInterface() {}

func (fbfd *FrameBasedFrameDescriptor) isFrameDescriptor() {}
