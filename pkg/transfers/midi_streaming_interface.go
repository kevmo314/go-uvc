package transfers

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// MIDI Streaming subclass constants
const (
	MIDI_STREAMING_SUBCLASS = 0x03

	// MIDI Streaming descriptor subtypes
	MS_DESCRIPTOR_UNDEFINED = 0x00
	MS_HEADER               = 0x01
	MIDI_IN_JACK            = 0x02
	MIDI_OUT_JACK           = 0x03
	ELEMENT                 = 0x04

	// Jack types
	JACK_TYPE_UNDEFINED = 0x00
	JACK_TYPE_EMBEDDED  = 0x01
	JACK_TYPE_EXTERNAL  = 0x02
)

type MIDIStreamingInterface struct {
	ctx    *C.libusb_context
	handle *C.struct_libusb_device_handle
	iface  *C.struct_libusb_interface

	// MIDI specific fields
	NumInJacks  uint8
	NumOutJacks uint8
	InJacks     []MIDIJack
	OutJacks    []MIDIJack
	EndpointIn  uint8
	EndpointOut uint8
	NumCables   uint8
}

type MIDIJack struct {
	JackID    uint8
	JackType  uint8 // EMBEDDED or EXTERNAL
	AssocJack uint8 // Associated jack ID
	StringIdx uint8 // String descriptor index
}

func NewMIDIStreamingInterface(ctxp, handlep, ifacep unsafe.Pointer) *MIDIStreamingInterface {
	ctx := (*C.struct_libusb_context)(ctxp)
	handle := (*C.struct_libusb_device_handle)(handlep)
	iface := (*C.struct_libusb_interface)(ifacep)
	return &MIDIStreamingInterface{ctx: ctx, handle: handle, iface: iface}
}

func (msi *MIDIStreamingInterface) InterfaceNumber() uint8 {
	return uint8(msi.iface.altsetting.bInterfaceNumber)
}

func (msi *MIDIStreamingInterface) ParseDescriptor(block []byte) error {
	if len(block) < 3 {
		return fmt.Errorf("descriptor too short")
	}

	subtype := block[2]
	switch subtype {
	case MS_HEADER:
		// MS header descriptor
		if len(block) >= 7 {
			// bcdMSC at block[3:4]
			// wTotalLength at block[5:6]
		}

	case MIDI_IN_JACK:
		// MIDI IN jack descriptor
		if len(block) >= 6 {
			jack := MIDIJack{
				JackType:  block[3],
				JackID:    block[4],
				StringIdx: block[5],
			}
			msi.InJacks = append(msi.InJacks, jack)
			msi.NumInJacks++
		}

	case MIDI_OUT_JACK:
		// MIDI OUT jack descriptor
		if len(block) >= 9 {
			jack := MIDIJack{
				JackType: block[3],
				JackID:   block[4],
				// block[5] = number of input pins
				// block[6] = source ID
				// block[7] = source pin
				StringIdx: block[8],
			}
			msi.OutJacks = append(msi.OutJacks, jack)
			msi.NumOutJacks++
		}
	}

	return nil
}

func (msi *MIDIStreamingInterface) ParseEndpoint(epDesc unsafe.Pointer) {
	ep := (*C.struct_libusb_endpoint_descriptor)(epDesc)
	epAddr := uint8(ep.bEndpointAddress)

	// Check direction bit (bit 7)
	if epAddr&0x80 != 0 {
		msi.EndpointIn = epAddr
	} else {
		msi.EndpointOut = epAddr
	}
}

// ParseMIDIEndpoint parses the class-specific MIDI endpoint descriptor
func (msi *MIDIStreamingInterface) ParseMIDIEndpoint(block []byte) {
	if len(block) >= 4 && block[1] == 0x25 && block[2] == 0x01 {
		// CS_ENDPOINT descriptor, MS_GENERAL subtype
		msi.NumCables = block[3] // Number of embedded MIDI jacks
		// block[4:] contains cable/jack associations
	}
}

// SendMIDIMessage sends a MIDI message to the device
func (msi *MIDIStreamingInterface) SendMIDIMessage(cableNum uint8, message []byte) error {
	if msi.EndpointOut == 0 {
		return fmt.Errorf("no MIDI output endpoint found")
	}

	// USB-MIDI uses 4-byte packets
	// Byte 0: Cable Number (4 bits) | Code Index Number (4 bits)
	// Bytes 1-3: MIDI message

	packet := make([]byte, 4)
	packet[0] = (cableNum << 4) | getMIDICodeIndex(message)
	copy(packet[1:], message)

	var transferred C.int
	ret := C.libusb_bulk_transfer(
		msi.handle,
		C.uchar(msi.EndpointOut),
		(*C.uchar)(unsafe.Pointer(&packet[0])),
		4,
		&transferred,
		1000,
	)

	if ret < 0 {
		return fmt.Errorf("MIDI send failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	return nil
}

// ReadMIDIMessage reads a MIDI message from the device
func (msi *MIDIStreamingInterface) ReadMIDIMessage() (cableNum uint8, message []byte, err error) {
	if msi.EndpointIn == 0 {
		return 0, nil, fmt.Errorf("no MIDI input endpoint found")
	}

	packet := make([]byte, 64) // Read up to 64 bytes
	var transferred C.int

	ret := C.libusb_bulk_transfer(
		msi.handle,
		C.uchar(msi.EndpointIn),
		(*C.uchar)(unsafe.Pointer(&packet[0])),
		64,
		&transferred,
		100, // 100ms timeout
	)

	if ret < 0 {
		return 0, nil, fmt.Errorf("MIDI read failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	if transferred >= 4 {
		cableNum = packet[0] >> 4
		// Extract MIDI message based on Code Index Number
		cin := packet[0] & 0x0F
		msgLen := getMIDIMessageLength(cin)
		message = packet[1 : 1+msgLen]
	}

	return cableNum, message, nil
}

// getMIDICodeIndex returns the Code Index Number for a MIDI message
func getMIDICodeIndex(message []byte) uint8 {
	if len(message) == 0 {
		return 0
	}

	status := message[0]

	// System messages
	if status >= 0xF0 {
		switch status {
		case 0xF0:
			return 0x04 // SysEx starts
		case 0xF7:
			return 0x05 // SysEx ends
		case 0xF2:
			return 0x03 // Song Position
		case 0xF3:
			return 0x02 // Song Select
		case 0xF6:
			return 0x05 // Tune Request
		case 0xF8, 0xFA, 0xFB, 0xFC, 0xFE, 0xFF:
			return 0x05 // Single-byte system real-time
		default:
			return 0x05 // Single byte
		}
	}

	// Channel messages
	switch status & 0xF0 {
	case 0x80, 0x90, 0xA0, 0xB0, 0xE0:
		return 0x09 // 2-byte message
	case 0xC0, 0xD0:
		return 0x02 // 1-byte message
	default:
		return 0x00 // Unknown
	}
}

// getMIDIMessageLength returns the length of a MIDI message based on Code Index
func getMIDIMessageLength(cin uint8) int {
	switch cin {
	case 0x02, 0x0C, 0x0D:
		return 2 // 2-byte messages
	case 0x03, 0x04, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0E:
		return 3 // 3-byte messages
	case 0x05, 0x0F:
		return 1 // 1-byte messages
	default:
		return 0
	}
}
