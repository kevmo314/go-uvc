// This file implements the descriptors as defined in the UVC spec 1.5, section 3.9.
package descriptors

import (
	"encoding/binary"
	"io"
)

type StreamingInterface struct {
	FormatDescriptor
	FrameDescriptor
	InputHeaderDescriptor     InputHeaderDescriptor
	OutputHeaderDescriptor    OutputHeaderDescriptor
	StillImageFrameDescriptor StillImageFrameDescriptor
	// TODO: add colorformat descriptor?
}

type VideoStreamingInterfaceDescriptorSubtype byte

const (
	VideoStreamingInterfaceDescriptorSubtypeUndefined           VideoStreamingInterfaceDescriptorSubtype = 0x00
	VideoStreamingInterfaceDescriptorSubtypeInputHeader         VideoStreamingInterfaceDescriptorSubtype = 0x01
	VideoStreamingInterfaceDescriptorSubtypeOutputHeader        VideoStreamingInterfaceDescriptorSubtype = 0x02
	VideoStreamingInterfaceDescriptorSubtypeStillImageFrame     VideoStreamingInterfaceDescriptorSubtype = 0x03
	VideoStreamingInterfaceDescriptorSubtypeFormatUncompressed  VideoStreamingInterfaceDescriptorSubtype = 0x04
	VideoStreamingInterfaceDescriptorSubtypeFrameUncompressed   VideoStreamingInterfaceDescriptorSubtype = 0x05
	VideoStreamingInterfaceDescriptorSubtypeFormatMJPEG         VideoStreamingInterfaceDescriptorSubtype = 0x06
	VideoStreamingInterfaceDescriptorSubtypeFrameMJPEG          VideoStreamingInterfaceDescriptorSubtype = 0x07
	VideoStreamingInterfaceDescriptorSubtypeFormatMPEG2TS       VideoStreamingInterfaceDescriptorSubtype = 0x0A
	VideoStreamingInterfaceDescriptorSubtypeFormatDV            VideoStreamingInterfaceDescriptorSubtype = 0x0C
	VideoStreamingInterfaceDescriptorSubtypeColorFormat         VideoStreamingInterfaceDescriptorSubtype = 0x0D
	VideoStreamingInterfaceDescriptorSubtypeFormatFrameBased    VideoStreamingInterfaceDescriptorSubtype = 0x10
	VideoStreamingInterfaceDescriptorSubtypeFrameFrameBased     VideoStreamingInterfaceDescriptorSubtype = 0x11
	VideoStreamingInterfaceDescriptorSubtypeFormatStreamBased   VideoStreamingInterfaceDescriptorSubtype = 0x12
	VideoStreamingInterfaceDescriptorSubtypeFormatH264          VideoStreamingInterfaceDescriptorSubtype = 0x13
	VideoStreamingInterfaceDescriptorSubtypeFrameH264           VideoStreamingInterfaceDescriptorSubtype = 0x14
	VideoStreamingInterfaceDescriptorSubtypeFormatH264Simulcast VideoStreamingInterfaceDescriptorSubtype = 0x15
	VideoStreamingInterfaceDescriptorSubtypeFormatVP8           VideoStreamingInterfaceDescriptorSubtype = 0x16
	VideoStreamingInterfaceDescriptorSubtypeFrameVP8            VideoStreamingInterfaceDescriptorSubtype = 0x17
	VideoStreamingInterfaceDescriptorSubtypeFormatVP8Simulcast  VideoStreamingInterfaceDescriptorSubtype = 0x18
)

// StandardVideoStreamingInterfaceDescriptor as defined in UVC spec 1.5, 3.9.1
type StandardVideoStreamingInterfaceDescriptor struct {
	InterfaceNumber  uint8
	AlternateSetting uint8
	NumEndpoints     uint8
	DescriptionIndex uint8
}

func (svsid *StandardVideoStreamingInterfaceDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	// TODO: fix the descriptor type, this is not the class specific one.
	// if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
	// 	return ErrInvalidDescriptor
	// }
	svsid.InterfaceNumber = buf[2]
	svsid.AlternateSetting = buf[3]
	svsid.NumEndpoints = buf[4]
	if ClassCode(buf[5]) != ClassCodeVideo {
		return ErrInvalidDescriptor
	}
	if SubclassCode(buf[6]) != SubclassCodeVideoStreaming {
		return ErrInvalidDescriptor
	}
	if ProtocolCode(buf[7]) != ProtocolCode15 {
		return ErrInvalidDescriptor
	}
	svsid.DescriptionIndex = buf[8]
	return nil
}

// InputHeaderDescriptor as defined in UVC spec 1.5, 3.9.2.1
type InputHeaderDescriptor struct {
	TotalLength        uint16
	EndpointAddress    uint8
	InfoBitmask        uint8
	TerminalLink       uint8
	StillCaptureMethod uint8
	TriggerSupport     uint8
	TriggerUsage       uint8
	ControlBitmasks    [][]byte
}

func (ihd *InputHeaderDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeInputHeader {
		return ErrInvalidDescriptor
	}
	p := buf[3]
	ihd.TotalLength = binary.LittleEndian.Uint16(buf[4:6])
	ihd.EndpointAddress = buf[6]
	ihd.InfoBitmask = buf[7]
	ihd.TerminalLink = buf[8]
	ihd.StillCaptureMethod = buf[9]
	ihd.TriggerSupport = buf[10]
	ihd.TriggerUsage = buf[11]
	n := buf[12]
	for i := uint8(0); i < p; i++ {
		ihd.ControlBitmasks[i] = buf[13+i*n : 13+(i+1)*n]
	}
	return nil
}

// OutputHeaderDescriptor as defined in UVC spec 1.5, 3.9.2.2
type OutputHeaderDescriptor struct {
	TotalLength     uint16
	EndpointAddress uint8
	TerminalLink    uint8
	ControlBitmasks [][]byte
}

func (ohd *OutputHeaderDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeOutputHeader {
		return ErrInvalidDescriptor
	}
	p := buf[3]
	ohd.TotalLength = binary.LittleEndian.Uint16(buf[4:6])
	ohd.EndpointAddress = buf[6]
	ohd.TerminalLink = buf[7]
	n := buf[8]
	for i := uint8(0); i < p; i++ {
		ohd.ControlBitmasks[i] = buf[9+i*n : 9+(i+1)*n]
	}
	return nil
}

// PayloadFormatDescriptor and VideoFrameDescriptor are implemented in the corresponding subpackages.

// StillImageFrameDescriptor as defined in UVC spec 1.5, 3.9.2.5
type StillImageFrameDescriptor struct {
	EndpointAddress   uint8
	ImageSizePatterns []struct {
		Width, Height uint16
	}
	CompressionPatterns []uint8
}

func (sifd *StillImageFrameDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeStillImageFrame {
		return ErrInvalidDescriptor
	}
	sifd.EndpointAddress = buf[3]
	n := buf[4]
	for i := uint8(0); i < n; i++ {
		sifd.ImageSizePatterns[i].Width = binary.LittleEndian.Uint16(buf[5+4*i : 7+4*i])
		sifd.ImageSizePatterns[i].Height = binary.LittleEndian.Uint16(buf[7+4*i : 9+4*i])
	}
	m := buf[5+n*4]
	for i := uint8(0); i < m; i++ {
		sifd.CompressionPatterns[i] = buf[6+n*4+i]
	}
	return nil
}

// ColorMatchingDescriptor as defined in UVC spec 1.5, 3.9.2.6
type ColorMatchingDescriptor struct {
	ColorPrimaries          uint8
	TransferCharacteristics uint8
	MatrixCoefficients      uint8
}

func (cmd *ColorMatchingDescriptor) Unmarshal(buf []byte) error {
	if len(buf) != int(buf[0]) {
		return io.ErrShortBuffer
	}
	if ClassSpecificDescriptorType(buf[1]) != ClassSpecificDescriptorTypeInterface {
		return ErrInvalidDescriptor
	}
	if VideoStreamingInterfaceDescriptorSubtype(buf[2]) != VideoStreamingInterfaceDescriptorSubtypeColorFormat {
		return ErrInvalidDescriptor
	}
	cmd.ColorPrimaries = buf[3]
	cmd.TransferCharacteristics = buf[4]
	cmd.MatrixCoefficients = buf[5]
	return nil
}
