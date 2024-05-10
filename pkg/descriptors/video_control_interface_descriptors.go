// This file implements the descriptors as defined in the UVC spec 1.5, section 3.7.
package descriptors

import (
	"encoding/binary"
	"io"
)

type ControlInterface struct {
	InputTerminalDescriptor  InputTerminalDescriptor
	OutputTerminalDescriptor OutputTerminalDescriptor
	SelectorUnitDescriptor   SelectorUnitDescriptor
	ProcessingUnitDescriptor ProcessingUnitDescriptor
	EncodingUnitDescriptor   EncodingUnitDescriptor
	ExtensionUnitDescriptor  ExtensionUnitDescriptor
}

type VideoControlInterfaceDescriptorSubtype byte

const (
	VideoControlInterfaceDescriptorSubtypeUndefined      VideoControlInterfaceDescriptorSubtype = 0x00
	VideoControlInterfaceDescriptorSubtypeHeader         VideoControlInterfaceDescriptorSubtype = 0x01
	VideoControlInterfaceDescriptorSubtypeInputTerminal  VideoControlInterfaceDescriptorSubtype = 0x02
	VideoControlInterfaceDescriptorSubtypeOutputTerminal VideoControlInterfaceDescriptorSubtype = 0x03
	VideoControlInterfaceDescriptorSubtypeSelectorUnit   VideoControlInterfaceDescriptorSubtype = 0x04
	VideoControlInterfaceDescriptorSubtypeProcessingUnit VideoControlInterfaceDescriptorSubtype = 0x05
	VideoControlInterfaceDescriptorSubtypeExtensionUnit  VideoControlInterfaceDescriptorSubtype = 0x06
	VideoControlInterfaceDescriptorSubtypeEncodingUnit   VideoControlInterfaceDescriptorSubtype = 0x07
)

type TerminalType uint16

const (
	TerminalTypeVendorSpecific TerminalType = 0x0100
	TerminalTypeStreaming      TerminalType = 0x0101
)

type InputTerminalType uint16

const (
	InputTerminalTypeVendorSpecific      InputTerminalType = 0x0200
	InputTerminalTypeCamera              InputTerminalType = 0x0201
	InputTerminalTypeMediaTransportInput InputTerminalType = 0x0202
)

type OutputTerminalType uint16

const (
	OutputTerminalTypeVendorSpecific       OutputTerminalType = 0x0300
	OutputTerminalTypeCamera               OutputTerminalType = 0x0301
	OutputTerminalTypeMediaTransportOutput OutputTerminalType = 0x0302
)

type ExternalTerminalType uint16

const (
	ExternalTerminalTypeVendorSpecific     ExternalTerminalType = 0x0400
	ExternalTerminalTypeCompositeConnector ExternalTerminalType = 0x0401
	ExternalTerminalTypeSVideoConnector    ExternalTerminalType = 0x0402
	ExternalTerminalTypeComponentConnector ExternalTerminalType = 0x0403
)

// StandardVideoControlInterfaceDescriptor as defined in UVC spec 1.5, 3.7.1
type StandardVideoControlInterfaceDescriptor struct {
	InterfaceNumber  uint8
	AlternateSetting uint8
	NumEndpoints     uint8
	DescriptionIndex uint8
}

func (svcid *StandardVideoControlInterfaceDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	// TODO: check the descriptor type, this is not the class specific one.
	// if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
	// 	return ErrInvalidDescriptor
	// }
	svcid.InterfaceNumber = buf[2]
	svcid.AlternateSetting = buf[3]
	svcid.NumEndpoints = buf[4]
	if ClassCode(buf[5]) != ClassCodeVideo {
		return ErrInvalidDescriptor
	}
	if SubclassCode(buf[6]) != SubclassCodeVideoControl {
		return ErrInvalidDescriptor
	}
	if ProtocolCode(buf[7]) != ProtocolCode15 {
		return ErrInvalidDescriptor
	}
	svcid.DescriptionIndex = buf[8]
	return nil
}

// InputTerminalDescriptor as defined in UVC spec 1.5, 3.7.2.1
type InputTerminalDescriptor struct {
	TerminalID           uint8
	TerminalType         InputTerminalType
	AssociatedTerminalID uint8
	DescriptionIndex     uint8
}

func (itd *InputTerminalDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoControlInterfaceDescriptorSubtype(buf[2]) != VideoControlInterfaceDescriptorSubtypeInputTerminal {
		return ErrInvalidDescriptor
	}
	itd.TerminalID = buf[3]
	itd.TerminalType = InputTerminalType(binary.LittleEndian.Uint16(buf[4:6]))
	itd.AssociatedTerminalID = buf[6]
	itd.DescriptionIndex = buf[7]
	return nil
}

// OutputTerminalDescriptor as defined in UVC spec 1.5, 3.7.2.2
type OutputTerminalDescriptor struct {
	TerminalID           uint8
	TerminalType         OutputTerminalType
	AssociatedTerminalID uint8
	SourceID             uint8
}

func (otd *OutputTerminalDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoControlInterfaceDescriptorSubtype(buf[2]) != VideoControlInterfaceDescriptorSubtypeOutputTerminal {
		return ErrInvalidDescriptor
	}
	otd.TerminalID = buf[3]
	otd.TerminalType = OutputTerminalType(binary.LittleEndian.Uint16(buf[4:6]))
	otd.AssociatedTerminalID = buf[6]
	otd.SourceID = buf[7]
	return nil
}

// CameraTerminalDescriptor as defined in UVC spec 1.5, 3.7.2.3
type CameraTerminalDescriptor struct {
	ObjectiveFocalLengthMin uint16
	ObjectiveFocalLengthMax uint16
	OcularFocalLength       uint16
	ControlsBitmask         uint32
}

func (ctd *CameraTerminalDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoControlInterfaceDescriptorSubtype(buf[2]) != VideoControlInterfaceDescriptorSubtypeInputTerminal {
		return ErrInvalidDescriptor
	}
	if InputTerminalType(binary.LittleEndian.Uint16(buf[4:6])) != InputTerminalTypeCamera {
		return ErrInvalidDescriptor
	}
	ctd.ObjectiveFocalLengthMin = binary.LittleEndian.Uint16(buf[8:10])
	ctd.ObjectiveFocalLengthMax = binary.LittleEndian.Uint16(buf[10:12])
	ctd.OcularFocalLength = binary.LittleEndian.Uint16(buf[12:14])
	ctd.ControlsBitmask = binary.LittleEndian.Uint32(buf[14:18])
	return nil
}

// SelectorUnitDescriptor as defined in UVC spec 1.5, 3.7.2.4
type SelectorUnitDescriptor struct {
	UnitID           uint8
	SourceID         []uint8
	DescriptionIndex uint8
}

func (sud *SelectorUnitDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoControlInterfaceDescriptorSubtype(buf[2]) != VideoControlInterfaceDescriptorSubtypeSelectorUnit {
		return ErrInvalidDescriptor
	}
	sud.UnitID = buf[3]
	p := buf[4]
	sud.SourceID = buf[5 : 5+p]
	sud.DescriptionIndex = buf[5+p]
	return nil
}

// ProcessingUnitDescriptor as defined in UVC spec 1.5, 3.7.2.5
type ProcessingUnitDescriptor struct {
	UnitID                uint8
	SourceID              uint8
	MaxMultiplier         uint16
	ControlsBitmask       uint32
	DescriptionIndex      uint8
	VideoStandardsBitmask uint8
}

func (pud *ProcessingUnitDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoControlInterfaceDescriptorSubtype(buf[2]) != VideoControlInterfaceDescriptorSubtypeProcessingUnit {
		return ErrInvalidDescriptor
	}
	pud.UnitID = buf[3]
	pud.SourceID = buf[4]
	pud.MaxMultiplier = binary.LittleEndian.Uint16(buf[6:8])
	pud.ControlsBitmask = binary.LittleEndian.Uint32(buf[8:12])
	pud.DescriptionIndex = buf[12]
	pud.VideoStandardsBitmask = buf[13]
	return nil
}

// EncodingUnitDescriptor as defined in UVC spec 1.5, 3.7.2.6
type EncodingUnitDescriptor struct {
	UnitID                 uint8
	SourceID               uint8
	DescriptionIndex       uint8
	ControlsBitmask        uint32
	ControlsRuntimeBitmask uint32
}

func (eud *EncodingUnitDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoControlInterfaceDescriptorSubtype(buf[2]) != VideoControlInterfaceDescriptorSubtypeEncodingUnit {
		return ErrInvalidDescriptor
	}
	eud.UnitID = buf[3]
	eud.SourceID = buf[4]
	eud.DescriptionIndex = buf[5]
	eud.ControlsBitmask = binary.LittleEndian.Uint32(buf[6:10])
	// this is off by one because the bitmask is actually only the lower 3 bytes.
	eud.ControlsRuntimeBitmask = binary.LittleEndian.Uint32(buf[9:13])
	return nil
}

// ExtensionUnitDescriptor as defined in UVC spec 1.5, 3.7.2.7
type ExtensionUnitDescriptor struct {
	UnitID            uint8
	GUIDExtensionCode [16]byte
	NumControls       uint8
	SourceIDs         []uint8
	ControlsBitmask   []byte
	DescriptionIndex  uint8
}

func (eud *ExtensionUnitDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoControlInterfaceDescriptorSubtype(buf[2]) != VideoControlInterfaceDescriptorSubtypeExtensionUnit {
		return ErrInvalidDescriptor
	}
	eud.UnitID = buf[3]
	copy(eud.GUIDExtensionCode[:], buf[4:20])
	eud.NumControls = buf[20]
	p := buf[21]
	eud.SourceIDs = buf[22 : 22+p]
	n := buf[22+p]
	eud.ControlsBitmask = buf[23+p : 23+p+n]
	eud.DescriptionIndex = buf[23+p+n]
	return nil
}
