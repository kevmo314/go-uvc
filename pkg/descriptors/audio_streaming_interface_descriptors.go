package descriptors

import "io"

type AudioStreamingInterfaceDescriptorSubtype byte

const (
	AudioStreamingInterfaceDescriptorSubtypeUndefined      AudioStreamingInterfaceDescriptorSubtype = 0x00
	AudioStreamingInterfaceDescriptorSubtypeHeader         AudioStreamingInterfaceDescriptorSubtype = 0x01
	AudioStreamingInterfaceDescriptorSubtypeInputTerminal  AudioStreamingInterfaceDescriptorSubtype = 0x02
	AudioStreamingInterfaceDescriptorSubtypeOutputTerminal AudioStreamingInterfaceDescriptorSubtype = 0x03
	AudioStreamingInterfaceDescriptorSubtypeMixerUnit      AudioStreamingInterfaceDescriptorSubtype = 0x04
	AudioStreamingInterfaceDescriptorSubtypeSelectorUnit   AudioStreamingInterfaceDescriptorSubtype = 0x05
	AudioStreamingInterfaceDescriptorSubtypeFeatureUnit    AudioStreamingInterfaceDescriptorSubtype = 0x06
	AudioStreamingInterfaceDescriptorSubtypeProcessingUnit AudioStreamingInterfaceDescriptorSubtype = 0x07
	AudioStreamingInterfaceDescriptorSubtypeExtensionUnit  AudioStreamingInterfaceDescriptorSubtype = 0x08
)

// StandardAudioStreamingInterfaceDescriptor as defined in UAC spec 1.0, section 4.5.1
type StandardAudioStreamingInterfaceDescriptor struct {
	InterfaceNumber   uint8
	AlternateSetting  uint8
	NumEndpoints      uint8
	InterfaceClass    uint8
	InterfaceSubClass uint8
	InterfaceProtocol uint8
	DescriptionIndex  uint8
}

func (sasid *StandardAudioStreamingInterfaceDescriptor) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	// TODO: check the descriptor type, this is not the class specific one.
	// if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
	// 	return ErrInvalidDescriptor
	// }
	sasid.InterfaceNumber = buf[2]
	sasid.AlternateSetting = buf[3]
	sasid.NumEndpoints = buf[4]
	sasid.InterfaceClass = buf[5]
	sasid.InterfaceSubClass = buf[6]
	sasid.InterfaceProtocol = buf[7]
	sasid.DescriptionIndex = buf[8]
	return nil
}
