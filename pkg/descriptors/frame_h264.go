package descriptors

import (
	"encoding/binary"
	"io"
	"time"
)

type H264StreamHeader struct {
	BitFieldHeader uint8
	PTS            uint32
	SCR            uint64
	SLI            uint16
}

func (hsh *H264StreamHeader) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	hsh.BitFieldHeader = buf[1]
	offset := 2
	if hsh.HasPTS() {
		hsh.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if hsh.HasSCR() {
		hsh.SCR = binary.LittleEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}
	if len(buf) >= offset+2 {
		hsh.SLI = binary.LittleEndian.Uint16(buf[offset : offset+2])
		offset += 2
	}
	return nil
}

func (hsh *H264StreamHeader) FrameIdentifier() bool {
	return hsh.BitFieldHeader&0b00000001 != 0
}

func (hsh *H264StreamHeader) EndOfFrame() bool {
	return hsh.BitFieldHeader&0b00000010 != 0
}

func (hsh *H264StreamHeader) HasPTS() bool {
	return hsh.BitFieldHeader&0b00000100 != 0
}

func (hsh *H264StreamHeader) HasSCR() bool {
	return hsh.BitFieldHeader&0b00001000 != 0
}

func (hsh *H264StreamHeader) EndOfSlice() bool {
	return hsh.BitFieldHeader&0b00010000 != 0
}

func (hsh *H264StreamHeader) StillImage() bool {
	return hsh.BitFieldHeader&0b00100000 != 0
}

func (hsh *H264StreamHeader) Error() bool {
	return hsh.BitFieldHeader&0b01000000 != 0
}

func (hsh *H264StreamHeader) EndOfHeader() bool {
	return hsh.BitFieldHeader&0b10000000 != 0
}

type H264FormatDescriptor struct {
	FormatIndex                                           uint8
	NumFrameDescriptors                                   uint8
	DefaultFrameIndex                                     uint8
	MaxCodecConfigDelay                                   uint8
	SupportedSliceModesBitmask                            uint8
	SupportedSyncFrameTypesBitmask                        uint8
	ResolutionScaling                                     uint8
	SupportedRateControlModesBitmask                      uint8
	MaxMBPerSecOneResolutionNoScalability                 uint16
	MaxMBPerSecTwoResolutionsNoScalability                uint16
	MaxMBPerSecThreeResolutionsNoScalability              uint16
	MaxMBPerSecFourResolutionsNoScalability               uint16
	MaxMBPerSecOneResolutionTemporalScalability           uint16
	MaxMBPerSecTwoResolutionsTemporalScalability          uint16
	MaxMBPerSecThreeResolutionsTemporalScalability        uint16
	MaxMBPerSecFourResolutionsTemporalScalability         uint16
	MaxMBPerSecOneResolutionTemporalQualityScalability    uint16
	MaxMBPerSecTwoResolutionsTemporalQualityScalability   uint16
	MaxMBPerSecThreeResolutionsTemporalQualityScalability uint16
	MaxMBPerSecFourResolutionsTemporalQualityScalability  uint16
	MaxMBPerSecOneResolutionTemporalSpatialScalability    uint16
	MaxMBPerSecTwoResolutionsTemporalSpatialScalability   uint16
	MaxMBPerSecThreeResolutionsTemporalSpatialScalability uint16
	MaxMBPerSecFourResolutionsTemporalSpatialScalability  uint16
	MaxMBPerSecOneResolutionFullScalability               uint16
	MaxMBPerSecTwoResolutionsFullScalability              uint16
	MaxMBPerSecThreeResolutionsFullScalability            uint16
	MaxMBPerSecFourResolutionsFullScalability             uint16
}

func (hfd *H264FormatDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatH264 &&
		VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFormatH264Simulcast {
		return ErrInvalidDescriptor
	}
	hfd.FormatIndex = buf[3]
	hfd.NumFrameDescriptors = buf[4]
	hfd.DefaultFrameIndex = buf[5]
	hfd.MaxCodecConfigDelay = buf[6]
	hfd.SupportedSliceModesBitmask = buf[7]
	hfd.SupportedSyncFrameTypesBitmask = buf[8]
	hfd.ResolutionScaling = buf[9]
	// buf[10] reserved
	hfd.SupportedRateControlModesBitmask = buf[11]
	hfd.MaxMBPerSecOneResolutionNoScalability = binary.LittleEndian.Uint16(buf[12:14])
	hfd.MaxMBPerSecTwoResolutionsNoScalability = binary.LittleEndian.Uint16(buf[14:16])
	hfd.MaxMBPerSecThreeResolutionsNoScalability = binary.LittleEndian.Uint16(buf[16:18])
	hfd.MaxMBPerSecFourResolutionsNoScalability = binary.LittleEndian.Uint16(buf[18:20])
	hfd.MaxMBPerSecOneResolutionTemporalScalability = binary.LittleEndian.Uint16(buf[20:22])
	hfd.MaxMBPerSecTwoResolutionsTemporalScalability = binary.LittleEndian.Uint16(buf[22:24])
	hfd.MaxMBPerSecThreeResolutionsTemporalScalability = binary.LittleEndian.Uint16(buf[24:26])
	hfd.MaxMBPerSecFourResolutionsTemporalScalability = binary.LittleEndian.Uint16(buf[26:28])
	hfd.MaxMBPerSecOneResolutionTemporalQualityScalability = binary.LittleEndian.Uint16(buf[28:30])
	hfd.MaxMBPerSecTwoResolutionsTemporalQualityScalability = binary.LittleEndian.Uint16(buf[30:32])
	hfd.MaxMBPerSecThreeResolutionsTemporalQualityScalability = binary.LittleEndian.Uint16(buf[32:34])
	hfd.MaxMBPerSecFourResolutionsTemporalQualityScalability = binary.LittleEndian.Uint16(buf[34:36])
	hfd.MaxMBPerSecOneResolutionTemporalSpatialScalability = binary.LittleEndian.Uint16(buf[36:38])
	hfd.MaxMBPerSecTwoResolutionsTemporalSpatialScalability = binary.LittleEndian.Uint16(buf[38:40])
	hfd.MaxMBPerSecThreeResolutionsTemporalSpatialScalability = binary.LittleEndian.Uint16(buf[40:42])
	hfd.MaxMBPerSecFourResolutionsTemporalSpatialScalability = binary.LittleEndian.Uint16(buf[42:44])
	hfd.MaxMBPerSecOneResolutionFullScalability = binary.LittleEndian.Uint16(buf[44:46])
	hfd.MaxMBPerSecTwoResolutionsFullScalability = binary.LittleEndian.Uint16(buf[46:48])
	hfd.MaxMBPerSecThreeResolutionsFullScalability = binary.LittleEndian.Uint16(buf[48:50])
	hfd.MaxMBPerSecFourResolutionsFullScalability = binary.LittleEndian.Uint16(buf[50:52])
	return nil
}

type H264FrameDescriptor struct {
	FrameIndex             uint8
	Width, Height          uint16
	SARWidth, SARHeight    uint16
	Profile                uint16
	LevelIDC               uint8
	SupportedUsagesBitmask uint32
	CapabilitiesBitmask    uint16
	SVCCapabilitiesBitmask uint32
	MVCCapabilitiesBitmask uint32
	MinBitRate, MaxBitRate uint32
	DefaultFrameInterval   time.Duration
	FrameIntervals         []time.Duration
}

func (hfd *H264FrameDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeFrameH264 {
		return ErrInvalidDescriptor
	}
	hfd.FrameIndex = buf[3]
	hfd.Width = binary.LittleEndian.Uint16(buf[4:6])
	hfd.Height = binary.LittleEndian.Uint16(buf[6:8])
	hfd.SARWidth = binary.LittleEndian.Uint16(buf[8:10])
	hfd.SARHeight = binary.LittleEndian.Uint16(buf[10:12])
	hfd.Profile = binary.LittleEndian.Uint16(buf[12:14])
	hfd.LevelIDC = buf[14]
	// buf[15:17] reserved
	hfd.SupportedUsagesBitmask = binary.LittleEndian.Uint32(buf[17:21])
	hfd.CapabilitiesBitmask = binary.LittleEndian.Uint16(buf[21:23])
	hfd.SVCCapabilitiesBitmask = binary.LittleEndian.Uint32(buf[23:27])
	hfd.MVCCapabilitiesBitmask = binary.LittleEndian.Uint32(buf[27:31])
	hfd.MinBitRate = binary.LittleEndian.Uint32(buf[31:35])
	hfd.MaxBitRate = binary.LittleEndian.Uint32(buf[35:39])
	hfd.DefaultFrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[39:43])) * 100 * time.Nanosecond
	n := buf[43]
	for i := uint8(0); i < n; i++ {
		hfd.FrameIntervals[i] = time.Duration(binary.LittleEndian.Uint32(buf[44+i*4:48+i*4])) * 100 * time.Nanosecond
	}
	return nil
}
