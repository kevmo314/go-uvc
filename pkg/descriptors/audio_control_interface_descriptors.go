package descriptors

import "io"

type AudioControlInterfaceDescriptorSubtype byte

const (
	AudioControlInterfaceDescriptorSubtypeUndefined           AudioControlInterfaceDescriptorSubtype = 0x00
	AudioControlInterfaceDescriptorSubtypeHeader              AudioControlInterfaceDescriptorSubtype = 0x01
	AudioControlInterfaceDescriptorSubtypeInputTerminal       AudioControlInterfaceDescriptorSubtype = 0x02
	AudioControlInterfaceDescriptorSubtypeOutputTerminal      AudioControlInterfaceDescriptorSubtype = 0x03
	AudioControlInterfaceDescriptorSubtypeMixerUnit           AudioControlInterfaceDescriptorSubtype = 0x04
	AudioControlInterfaceDescriptorSubtypeSelectorUnit        AudioControlInterfaceDescriptorSubtype = 0x05
	AudioControlInterfaceDescriptorSubtypeFeatureUnit         AudioControlInterfaceDescriptorSubtype = 0x06
	AudioControlInterfaceDescriptorSubtypeProcessingUnit      AudioControlInterfaceDescriptorSubtype = 0x07
	AudioControlInterfaceDescriptorSubtypeExtensionUnit       AudioControlInterfaceDescriptorSubtype = 0x08
	AudioControlInterfaceDescriptorSubtypeClockSource         AudioControlInterfaceDescriptorSubtype = 0x0A // UAC2
	AudioControlInterfaceDescriptorSubtypeClockSelector       AudioControlInterfaceDescriptorSubtype = 0x0B // UAC2
	AudioControlInterfaceDescriptorSubtypeClockMultiplier     AudioControlInterfaceDescriptorSubtype = 0x0C // UAC2
	AudioControlInterfaceDescriptorSubtypeSampleRateConverter AudioControlInterfaceDescriptorSubtype = 0x0D // UAC2
)

type AudioControlInterface interface {
	Subtype() AudioControlInterfaceDescriptorSubtype
}

type AudioStreamingDescriptor interface {
	Subtype() AudioStreamingInterfaceDescriptorSubtype
}

// AudioControlHeaderDescriptor represents the header descriptor for audio control interface
type AudioControlHeaderDescriptor struct {
	BcdADC       uint16  // Audio Device Class Specification Release Number in BCD
	TotalLength  uint16  // Total number of bytes returned for the class-specific AudioControl interface descriptor
	InCollection uint8   // Number of AudioStreaming and MidiStreaming interfaces
	InterfaceNr  []uint8 // Interface numbers of the AudioStreaming or MidiStreaming interfaces
}

func (achd *AudioControlHeaderDescriptor) Subtype() AudioControlInterfaceDescriptorSubtype {
	return AudioControlInterfaceDescriptorSubtypeHeader
}

func (achd *AudioControlHeaderDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < 8 {
		return io.ErrShortBuffer
	}
	achd.BcdADC = uint16(buf[3]) | (uint16(buf[4]) << 8)
	achd.TotalLength = uint16(buf[5]) | (uint16(buf[6]) << 8)
	achd.InCollection = buf[7]

	if len(buf) < 8+int(achd.InCollection) {
		return io.ErrShortBuffer
	}

	achd.InterfaceNr = make([]uint8, achd.InCollection)
	for i := 0; i < int(achd.InCollection); i++ {
		achd.InterfaceNr[i] = buf[8+i]
	}
	return nil
}

// AudioInputTerminalDescriptor represents an input terminal descriptor
type AudioInputTerminalDescriptor struct {
	TerminalID    uint8
	TerminalType  uint16
	AssocTerminal uint8
	NrChannels    uint8
	ChannelConfig uint16
	ChannelNames  uint8
	Terminal      uint8
}

func (aitd *AudioInputTerminalDescriptor) Subtype() AudioControlInterfaceDescriptorSubtype {
	return AudioControlInterfaceDescriptorSubtypeInputTerminal
}

func (aitd *AudioInputTerminalDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < 12 {
		return io.ErrShortBuffer
	}
	aitd.TerminalID = buf[3]
	aitd.TerminalType = uint16(buf[4]) | (uint16(buf[5]) << 8)
	aitd.AssocTerminal = buf[6]
	aitd.NrChannels = buf[7]
	aitd.ChannelConfig = uint16(buf[8]) | (uint16(buf[9]) << 8)
	aitd.ChannelNames = buf[10]
	aitd.Terminal = buf[11]
	return nil
}

// AudioOutputTerminalDescriptor represents an output terminal descriptor
type AudioOutputTerminalDescriptor struct {
	TerminalID    uint8
	TerminalType  uint16
	AssocTerminal uint8
	SourceID      uint8
	Terminal      uint8
}

func (aotd *AudioOutputTerminalDescriptor) Subtype() AudioControlInterfaceDescriptorSubtype {
	return AudioControlInterfaceDescriptorSubtypeOutputTerminal
}

func (aotd *AudioOutputTerminalDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < 9 {
		return io.ErrShortBuffer
	}
	aotd.TerminalID = buf[3]
	aotd.TerminalType = uint16(buf[4]) | (uint16(buf[5]) << 8)
	aotd.AssocTerminal = buf[6]
	aotd.SourceID = buf[7]
	aotd.Terminal = buf[8]
	return nil
}

// AudioFeatureUnitDescriptor represents a feature unit descriptor
type AudioFeatureUnitDescriptor struct {
	UnitID      uint8
	SourceID    uint8
	ControlSize uint8
	Controls    []uint8
	Feature     uint8
}

func (afud *AudioFeatureUnitDescriptor) Subtype() AudioControlInterfaceDescriptorSubtype {
	return AudioControlInterfaceDescriptorSubtypeFeatureUnit
}

func (afud *AudioFeatureUnitDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < 7 {
		return io.ErrShortBuffer
	}
	afud.UnitID = buf[3]
	afud.SourceID = buf[4]
	afud.ControlSize = buf[5]

	// Calculate number of controls
	numControls := (len(buf) - 7) / int(afud.ControlSize)
	if len(buf) < 6+numControls*int(afud.ControlSize)+1 {
		return io.ErrShortBuffer
	}

	afud.Controls = make([]uint8, numControls*int(afud.ControlSize))
	copy(afud.Controls, buf[6:6+len(afud.Controls)])
	afud.Feature = buf[6+len(afud.Controls)]
	return nil
}
