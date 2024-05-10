package descriptors

import (
	"encoding/binary"
	"io"
	"time"
)

type VP8StreamHeader struct {
	BitFieldHeader0 uint8
	BitFieldHeader1 uint8
	BitFieldHeader2 uint8
	PTS             uint32
	SCR             uint64
	SLI             uint16
}

func (vph *VP8StreamHeader) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	vph.BitFieldHeader0 = buf[1]
	vph.BitFieldHeader1 = buf[2]
	vph.BitFieldHeader2 = buf[3]
	offset := 4
	if vph.HasPTS() {
		vph.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if vph.HasSCR() {
		vph.SCR = binary.LittleEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}
	if vph.HasSLI() {
		vph.SLI = binary.LittleEndian.Uint16(buf[offset : offset+2])
		offset += 2
	}
	return nil
}

func (vph *VP8StreamHeader) FrameIdentifier() bool {
	return vph.BitFieldHeader0&0b00000001 != 0
}

func (vph *VP8StreamHeader) EndOfFrame() bool {
	return vph.BitFieldHeader0&0b00000010 != 0
}

func (vph *VP8StreamHeader) HasPTS() bool {
	return vph.BitFieldHeader0&0b00000100 != 0
}

func (vph *VP8StreamHeader) HasSCR() bool {
	return vph.BitFieldHeader0&0b00001000 != 0
}

func (vph *VP8StreamHeader) EndOfSlice() bool {
	return vph.BitFieldHeader0&0b00010000 != 0
}

func (vph *VP8StreamHeader) StillImage() bool {
	return vph.BitFieldHeader0&0b00100000 != 0
}

func (vph *VP8StreamHeader) Error() bool {
	return vph.BitFieldHeader0&0b01000000 != 0
}

func (vph *VP8StreamHeader) HasSLI() bool {
	return vph.BitFieldHeader0&0b10000000 != 0
}

func (vph *VP8StreamHeader) PreviousReferenceFrame() bool {
	return vph.BitFieldHeader1&0b00000001 != 0
}

func (vph *VP8StreamHeader) AlternateReferenceFrame() bool {
	return vph.BitFieldHeader1&0b00000010 != 0
}

func (vph *VP8StreamHeader) GoldenReferenceFrame() bool {
	return vph.BitFieldHeader1&0b00000100 != 0
}

func (vph *VP8StreamHeader) EndOfHeader() bool {
	return vph.BitFieldHeader2&0b10000000 != 0
}

type VP8FormatDescriptor struct {
	FormatIndex                      uint8
	NumFrameDescriptors              uint8
	DefaultFrameIndex                uint8
	MaxCodecConfigDelay              uint8
	SupportedPartitionCount          uint8
	SupportedSyncFrameTypesBitmask   uint8
	ResolutionScaling                uint8
	SupportedRateControlModesBitmask uint8
	MaxMBPerSec                      uint16
}

func (vfd *VP8FormatDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatVP8 {
		return ErrInvalidDescriptor
	}
	vfd.FormatIndex = buf[3]
	vfd.NumFrameDescriptors = buf[4]
	vfd.DefaultFrameIndex = buf[5]
	vfd.MaxCodecConfigDelay = buf[6]
	vfd.SupportedPartitionCount = buf[7]
	vfd.SupportedSyncFrameTypesBitmask = buf[8]
	vfd.ResolutionScaling = buf[9]
	vfd.SupportedRateControlModesBitmask = buf[10]
	vfd.MaxMBPerSec = binary.LittleEndian.Uint16(buf[11:13])
	return nil
}

type VP8FrameDescriptor struct {
	FrameIndex                     uint8
	Width, Height                  uint16
	SupportedUsagesBitmask         uint32
	CapabilitiesBitmask            uint16
	ScalabilityCapabilitiesBitmask uint32
	MinBitRate, MaxBitRate         uint32
	DefaultFrameInterval           time.Duration
	FrameIntervals                 []time.Duration
}

func (vfd *VP8FrameDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFrameVP8 {
		return ErrInvalidDescriptor
	}
	vfd.FrameIndex = buf[3]
	vfd.Width = binary.LittleEndian.Uint16(buf[4:6])
	vfd.Height = binary.LittleEndian.Uint16(buf[6:8])
	vfd.SupportedUsagesBitmask = binary.LittleEndian.Uint32(buf[8:12])
	vfd.CapabilitiesBitmask = binary.LittleEndian.Uint16(buf[12:14])
	vfd.ScalabilityCapabilitiesBitmask = binary.LittleEndian.Uint32(buf[14:18])
	vfd.MinBitRate = binary.LittleEndian.Uint32(buf[18:22])
	vfd.MaxBitRate = binary.LittleEndian.Uint32(buf[22:26])
	vfd.DefaultFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[26:30])) * 100 * time.Nanosecond
	n := buf[30]
	for i := uint8(0); i < n; i++ {
		vfd.FrameIntervals[i] = time.Duration(binary.LittleEndian.Uint32(buf[31+i*4:35+i*4])) * 100 * time.Nanosecond
	}
	return nil
}
