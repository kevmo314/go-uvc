package descriptors

import (
	"encoding/binary"
	"io"
	"time"
)

type MJPEGStreamHeader struct {
	BitFieldHeader uint8
	PTS            uint32
	SCR            uint64
}

func (msh *MJPEGStreamHeader) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	msh.BitFieldHeader = buf[1]
	offset := 2
	if msh.HasPTS() {
		msh.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if msh.HasSCR() {
		msh.SCR = binary.LittleEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}
	return nil
}

func (msh *MJPEGStreamHeader) FrameIdentifier() bool {
	return msh.BitFieldHeader&0b00000001 != 0
}

func (msh *MJPEGStreamHeader) EndOfFrame() bool {
	return msh.BitFieldHeader&0b00000010 != 0
}

func (msh *MJPEGStreamHeader) HasPTS() bool {
	return msh.BitFieldHeader&0b00000100 != 0
}

func (msh *MJPEGStreamHeader) HasSCR() bool {
	return msh.BitFieldHeader&0b00001000 != 0
}

func (msh *MJPEGStreamHeader) StillImage() bool {
	return msh.BitFieldHeader&0b00100000 != 0
}

func (msh *MJPEGStreamHeader) Error() bool {
	return msh.BitFieldHeader&0b01000000 != 0
}

func (msh *MJPEGStreamHeader) EndOfHeader() bool {
	return msh.BitFieldHeader&0b10000000 != 0
}

type MJPEGFormatDescriptor struct {
	FormatIndex                uint8
	NumFrameDescriptors        uint8
	Flags                      uint8
	DefaultFrameIndex          uint8
	AspectRatioX, AspectRatioY uint8
	InterlaceFlags             uint8
	CopyProtect                uint8
}

func (mfd *MJPEGFormatDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatMJPEG {
		return ErrInvalidDescriptor
	}
	mfd.FormatIndex = buf[3]
	mfd.NumFrameDescriptors = buf[4]
	mfd.Flags = buf[5]
	mfd.DefaultFrameIndex = buf[6]
	mfd.AspectRatioX = buf[7]
	mfd.AspectRatioY = buf[8]
	mfd.InterlaceFlags = buf[9]
	mfd.CopyProtect = buf[10]
	return nil
}

func (mfd *MJPEGFormatDescriptor) isStreamingInterface() {}

func (mfd *MJPEGFormatDescriptor) isFormatDescriptor() {}

type MJPEGFrameDescriptor struct {
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

func (mfd *MJPEGFrameDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFrameMJPEG {
		return ErrInvalidDescriptor
	}
	mfd.FrameIndex = buf[3]
	mfd.Capabilities = buf[4]
	mfd.Width = binary.LittleEndian.Uint16(buf[5:7])
	mfd.Height = binary.LittleEndian.Uint16(buf[7:9])
	mfd.MinBitRate = binary.LittleEndian.Uint32(buf[9:13])
	mfd.MaxBitRate = binary.LittleEndian.Uint32(buf[13:17])
	mfd.MaxVideoFrameBufferSize = binary.LittleEndian.Uint32(buf[17:21])
	mfd.DefaultFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[21:25])) * 100 * time.Nanosecond

	n := buf[25]

	if n == 0 {
		// Continuous frame intervals
		mfd.ContinuousFrameInterval.MinFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[26:30])) * 100 * time.Nanosecond
		mfd.ContinuousFrameInterval.MaxFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[30:34])) * 100 * time.Nanosecond
		mfd.ContinuousFrameInterval.FrameIntervalStep = time.Duration(binary.LittleEndian.Uint32(buf[34:38])) * 100 * time.Nanosecond
		return nil
	} else {
		mfd.DiscreteFrameIntervals = make([]time.Duration, n)
		for i := uint8(0); i < n; i++ {
			mfd.DiscreteFrameIntervals[i] = time.Duration(binary.LittleEndian.Uint32(buf[26+i*4:30+i*4])) * 100 * time.Nanosecond
		}
		return nil
	}
}

func (mfd *MJPEGFrameDescriptor) isStreamingInterface() {}

func (mfd *MJPEGFrameDescriptor) isFrameDescriptor() {}
